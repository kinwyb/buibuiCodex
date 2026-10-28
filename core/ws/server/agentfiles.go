// 本文件提供「操作 agent 工作区」的 HTTP 接口：列目录、读文件、上传、删除、建目录。
//
// 与 files.go 的区别：
//
//	files.go      临时文件的中转站（浏览器 → agent 的附件、agent → 浏览器的产物），
//	              按 fileID 索引，有 TTL，不关心它在 agent 眼里叫什么。
//	agentfiles.go agent 自己工作区的真实目录，按容器路径寻址，改动会直接影响 agent。
//
// 前端只知道容器路径（/workspace、/tmp），由请求里的 agent 名决定用哪套映射。
// 路径解析与越界校验全在 core/agentfs 里，这里只做 HTTP 与错误码翻译。
//
// 这是**受信前端**才能开的能力：它能读写整个工作区，等于把 agent 的文件系统
// 摊开给页面。默认关闭，需要显式配置 websocket.agent_files = true。

package server

import (
	"errors"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/agentfs"
)

// agentRootsData GET /ws/agents 的响应
type agentRootsData struct {
	Agents []agentRoots `json:"agents"`
}

type agentRoots struct {
	Name string `json:"name"`
	// Roots 该 agent 可浏览的根（容器视角），含目录是否已创建
	Roots []agentfs.Root `json:"roots"`
}

// agentDirData GET /ws/agents/{agent}/files 的响应
type agentDirData struct {
	Agent string `json:"agent"`
	// Path 当前目录的容器路径
	Path string `json:"path"`
	// Parent 上一级目录，位于根时为空
	Parent string `json:"parent,omitempty"`
	// Roots 便于前端渲染根切换，不必再查一次 /ws/agents
	Roots []agentfs.Root  `json:"roots"`
	Items []agentfs.Entry `json:"items"`
}

// agentEntryData 写操作（上传/建目录）的响应
type agentEntryData struct {
	Agent string        `json:"agent"`
	Entry agentfs.Entry `json:"entry"`
}

// agentsBase agent 端点前缀，随升级路径走
func (s *Server) agentsBase() string {
	return strings.TrimSuffix(s.cfg.Path, "/") + "/agents"
}

// registerAgentFileRoutes 注册 agent 工作区端点。
//
// 用 query 传路径而不是路径通配符：容器路径本身含 /（/workspace/a/b.txt），
// 放进 URL path 里既要转义又容易和 {agent} 段混淆。
func (s *Server) registerAgentFileRoutes(mux *http.ServeMux) {
	base := s.agentsBase()
	mux.HandleFunc("GET "+base, s.handleAgentRoots)
	mux.HandleFunc("GET "+base+"/{agent}/files", s.handleAgentFileList)
	mux.HandleFunc("GET "+base+"/{agent}/file", s.handleAgentFileGet)
	mux.HandleFunc("HEAD "+base+"/{agent}/file", s.handleAgentFileGet)
	mux.HandleFunc("PUT "+base+"/{agent}/file", s.handleAgentFilePut)
	mux.HandleFunc("POST "+base+"/{agent}/file", s.handleAgentFileUpload)
	mux.HandleFunc("DELETE "+base+"/{agent}/file", s.handleAgentFileDelete)
	mux.HandleFunc("POST "+base+"/{agent}/dir", s.handleAgentMkdir)
}

// handleAgentRoots 列出可用的 agent 及其可浏览的根
func (s *Server) handleAgentRoots(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	names := make([]string, 0, 4)
	if s.opts.Agents != nil {
		names = s.opts.Agents.ListAgents()
	}
	sort.Strings(names)

	out := agentRootsData{Agents: make([]agentRoots, 0, len(names))}
	for _, name := range names {
		fs, err := s.agentFS(name)
		if err != nil {
			// 没有映射的 agent（理论上不该出现）跳过，不影响其它 agent 展示
			continue
		}
		out.Agents = append(out.Agents, agentRoots{Name: name, Roots: fs.Roots()})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAgentFileList 列目录。path 为空时用第一个根。
func (s *Server) handleAgentFileList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	agent := r.PathValue("agent")
	fs, err := s.agentFS(agent)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}

	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" && len(fs.Roots()) > 0 {
		dir = fs.Roots()[0].Container
	}
	cleaned := path.Clean(dir)

	items, err := fs.List(cleaned)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}

	out := agentDirData{
		Agent: agent,
		Path:  cleaned,
		Roots: fs.Roots(),
		Items: items,
	}
	// 已经在根上就没有上级（避免 /workspace 的父目录 / 被当成可浏览路径）
	for _, root := range fs.Roots() {
		if root.Container == cleaned {
			out.Parent = ""
			break
		}
		out.Parent = path.Dir(cleaned)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAgentFileGet 读取文件内容。默认内联，?download=1 另存，?meta=1 只给元信息。
func (s *Server) handleAgentFileGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	fs, err := s.agentFS(r.PathValue("agent"))
	if err != nil {
		writeAgentFSError(w, err)
		return
	}

	target := strings.TrimSpace(r.URL.Query().Get("path"))
	if target == "" {
		writeJSONError(w, http.StatusBadRequest, "path_required", "缺少 path 参数")
		return
	}

	if r.URL.Query().Get("meta") == "1" {
		entry, err := fs.Stat(target)
		if err != nil {
			writeAgentFSError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, entry)
		return
	}

	file, entry, err := fs.Open(target)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}
	defer file.Close()

	// 与 files.go 同一套规则：可执行类型（HTML/SVG/XML）一律降级为另存，
	// 避免同源内联变成 XSS 入口
	mimeType := entry.Mime
	if mimeType == "" {
		mimeType = sniffMime(file)
	}
	inline := r.URL.Query().Get("download") != "1" && canInline(mimeType, entry.Name)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", contentDisposition(disposition, entry.Name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if inline {
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src data:; style-src 'unsafe-inline'")
	}
	w.Header().Set("Cache-Control", "no-cache")

	http.ServeContent(w, r, entry.Name, time.UnixMilli(entry.ModTime), file)
}

// handleAgentFilePut 用请求体覆盖写入文件（?path 必须是完整文件路径）。
//
// 适合前端编辑器保存：请求体就是文件内容，配合 fetch + ReadableStream
// 可以直接把 File 对象流上来，不必构造 multipart。
func (s *Server) handleAgentFilePut(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	agent := r.PathValue("agent")
	fs, err := s.agentFS(agent)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("path"))
	if target == "" {
		writeJSONError(w, http.StatusBadRequest, "path_required", "缺少 path 参数")
		return
	}

	limit := s.uploadLimit()
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	entry, err := fs.Save(target, r.Body, limit)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentEntryData{Agent: agent, Entry: entry})
}

// handleAgentFileUpload 表单上传（multipart），目标目录用 ?path 指定，
// 文件名取自表单；也可以直接在 ?path 里给完整路径覆盖同名文件。
func (s *Server) handleAgentFileUpload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	agent := r.PathValue("agent")
	fs, err := s.agentFS(agent)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}

	limit := s.uploadLimit()
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))

	reader, err := r.MultipartReader()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "需要 multipart/form-data 请求，或用 PUT 直接上传")
		return
	}
	base := strings.TrimSpace(r.URL.Query().Get("path"))
	if base == "" && len(fs.Roots()) > 0 {
		base = fs.Roots()[0].Container
	}

	var entry agentfs.Entry
	saved := false
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		if part.FormName() != fileFieldName || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		target := joinContainerPath(base, part.FileName())
		entry, err = fs.Save(target, part, limit)
		_ = part.Close()
		if err != nil {
			writeAgentFSError(w, err)
			return
		}
		saved = true
		break
	}
	if !saved {
		writeJSONError(w, http.StatusBadRequest, "file_required", "未找到文件字段，请用 file 作为字段名")
		return
	}
	writeJSON(w, http.StatusOK, agentEntryData{Agent: agent, Entry: entry})
}

// handleAgentFileDelete 删除文件；目录需要 ?recursive=1。
func (s *Server) handleAgentFileDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	fs, err := s.agentFS(r.PathValue("agent"))
	if err != nil {
		writeAgentFSError(w, err)
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("path"))
	if target == "" {
		writeJSONError(w, http.StatusBadRequest, "path_required", "缺少 path 参数")
		return
	}
	recursive := r.URL.Query().Get("recursive") == "1"
	if err := fs.Remove(target, recursive); err != nil {
		writeAgentFSError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAgentMkdir 创建目录（含父目录）
func (s *Server) handleAgentMkdir(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authUser(w, r); !ok {
		return
	}
	agent := r.PathValue("agent")
	fs, err := s.agentFS(agent)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("path"))
	if target == "" {
		writeJSONError(w, http.StatusBadRequest, "path_required", "缺少 path 参数")
		return
	}
	entry, err := fs.Mkdir(target)
	if err != nil {
		writeAgentFSError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentEntryData{Agent: agent, Entry: entry})
}

// ---- 小工具 ----

var errAgentFilesDisabled = errors.New("agent 文件接口未启用")

// agentFS 取某 agent 的文件视图。未启用与 agent 不存在要分开报，
// 否则前端只会看到一个笼统的 404，分不清是没开功能还是名字写错。
func (s *Server) agentFS(name string) (*agentfs.FS, error) {
	if s.opts.AgentFiles == nil {
		return nil, errAgentFilesDisabled
	}
	fs, ok := s.opts.AgentFiles.AgentFS(name)
	if !ok {
		return nil, errAgentNotFound
	}
	return fs, nil
}

var errAgentNotFound = errors.New("指定的 agent 不存在或未配置工作区")

// writeAgentFSError 把 agentfs 的错误翻译成 HTTP 状态码
func writeAgentFSError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errAgentFilesDisabled):
		// 501：服务端没开这个能力，不是请求的问题
		writeJSONError(w, http.StatusNotImplemented, "agent_files_disabled",
			"agent 文件接口未启用，请设置 websocket.agent_files = true")
	case errors.Is(err, errAgentNotFound):
		writeJSONError(w, http.StatusNotFound, "agent_not_found", err.Error())
	case errors.Is(err, agentfs.ErrOutOfScope):
		// 越界与软链接逃逸都归到这里：不回显真实路径，避免探测目录结构
		writeJSONError(w, http.StatusForbidden, "out_of_scope", "路径不在该 agent 的工作区内")
	case errors.Is(err, agentfs.ErrNotAbsolute):
		writeJSONError(w, http.StatusBadRequest, "path_not_absolute", "path 必须是容器内绝对路径，如 /workspace/a.txt")
	case errors.Is(err, agentfs.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "file_not_found", "文件或目录不存在")
	case errors.Is(err, agentfs.ErrIsDir):
		writeJSONError(w, http.StatusBadRequest, "is_directory", "目标是目录；读目录用 files 接口，删目录加 recursive=1")
	case errors.Is(err, agentfs.ErrNotDir):
		writeJSONError(w, http.StatusBadRequest, "not_directory", "目标不是目录")
	case errors.Is(err, agentfs.ErrTooLarge):
		writeJSONError(w, http.StatusRequestEntityTooLarge, "file_too_large", "文件超过上限")
	case errors.Is(err, agentfs.ErrRootProtected):
		writeJSONError(w, http.StatusForbidden, "root_protected", "不允许删除工作区根目录")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal", err.Error())
	}
}

// joinContainerPath 把表单文件名拼到目录后面。文件名只取末段，
// 防止上传方用 "../../etc/passwd" 当文件名。
func joinContainerPath(base, name string) string {
	if base == "" {
		base = "/"
	}
	safe := name
	if i := strings.LastIndexAny(safe, `/\`); i >= 0 {
		safe = safe[i+1:]
	}
	if safe == "" || safe == "." || safe == ".." {
		safe = "upload.bin"
	}
	return path.Join(base, safe)
}

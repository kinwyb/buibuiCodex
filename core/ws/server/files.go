// 本文件实现 ws 的文件收发：HTTP 上传/下载端点，以及 agent 侧 send_file 的回流。
//
// 为什么文件不走 WebSocket：
//
//	ws 单帧上限 1MB（protocol.MaxMessageSize），base64 再膨胀 4/3，
//	稍大的附件就撞墙；而且帧内传输没法断点续传、没法用浏览器自身的下载/预览能力。
//	所以字节走 HTTP，ws 只传 fileID —— 一条消息里几十字节的引用。
//
// 两条链路：
//
//	上行  POST /ws/files            -> fileID -> chat.send{fileIDs}
//	下行  agent send_file           -> 登记 -> 推一帧带 media.URL 的 tool_completed
//
// 路径转换交给 core.Manager：入站附件会在投递前由 mapMediaToContainer
// 移动到目标 agent 工作区的 tmp 并换成容器路径；出站的 send_file 请求
// 在工具层已经 ToHost，这里拿到的是宿主机路径，可以直接读。

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/filestore"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

const (
	// filePathSegment 文件端点挂在升级路径下：/ws -> /ws/files
	filePathSegment = "/files"
	// fileFieldName multipart 表单里的文件字段名
	fileFieldName = "file"
	// downloadTimeout 下载远程文件（send_file 传 URL 时）的超时
	downloadTimeout = 60 * time.Second
	// gcInterval 过期文件回收间隔
	gcInterval = time.Hour
)

// files 返回文件存储，首次访问时按配置创建。
//
// Options.Files 允许外部注入（测试、或多个实例共享同一目录），
// 不注入时按 cfg.FileDir / cfg.FileTTLDuration() 自建。
func (s *Server) files() *filestore.Store {
	s.filesOnce.Do(func() {
		if s.opts.Files != nil {
			s.fileStore = s.opts.Files
			return
		}
		s.fileStore = filestore.New(s.cfg.FileDir, s.cfg.FileTTLDuration())
	})
	return s.fileStore
}

// fileBase 文件端点的路径前缀，随升级路径走
func (s *Server) fileBase() string {
	return strings.TrimSuffix(s.cfg.Path, "/") + filePathSegment
}

// fileURL 单个文件的相对访问地址。
//
// 用相对路径而不是绝对地址：服务端通常躲在反向代理后面，自己并不知道
// 浏览器实际用的 host 与端口，拼死一个 host 反而会把请求带偏。
func (s *Server) fileURL(id string) string {
	return s.fileBase() + "/" + id
}

// registerFileRoutes 注册文件相关端点。
//
// 这里不对 /ws 与 /ws/files 的先后做任何假设：Go 1.22 起的 ServeMux
// 按模式精确匹配，"/ws" 不会吃掉 "/ws/files"。
func (s *Server) registerFileRoutes(mux *http.ServeMux) {
	base := s.fileBase()
	mux.HandleFunc("POST "+base, s.handleFileUpload)
	mux.HandleFunc("PUT "+base, s.handleFileRawUpload)
	mux.HandleFunc("GET "+base, s.handleFileList)
	mux.HandleFunc("GET "+base+"/{id}", s.handleFileContent)
	mux.HandleFunc("HEAD "+base+"/{id}", s.handleFileContent)
	mux.HandleFunc("DELETE "+base+"/{id}", s.handleFileDelete)
}

// ---- 上行：上传 ----

// handleFileUpload 接收 multipart/form-data，字段名 file，可带多个。
//
// 响应统一是 {"items":[FileInfo,...]}：一个文件也是数组，
// 前端不必为「传一个」和「传多个」写两套解析。
func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.authUser(w, r)
	if !ok {
		return
	}

	// 预留 1MB 给 multipart 的边界与表单头，避免把正常请求误判成超限
	limit := s.uploadLimit()
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))

	reader, err := r.MultipartReader()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, protocol.CodeBadRequest, "需要 multipart/form-data 请求")
		return
	}

	items := make([]protocol.FileInfo, 0, 1)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if isTooLarge(err) {
				writeJSONError(w, http.StatusRequestEntityTooLarge, protocol.CodeFileTooLarge,
					fmt.Sprintf("文件超过上限 %d 字节", limit))
				return
			}
			writeJSONError(w, http.StatusBadRequest, protocol.CodeBadRequest, "表单解析失败")
			return
		}
		if part.FormName() != fileFieldName || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		item, err := s.files().Save(userID, part.FileName(), part, limit)
		_ = part.Close()
		if err != nil {
			writeUploadError(w, err, limit)
			return
		}
		items = append(items, s.fileInfo(item))
	}

	if len(items) == 0 {
		writeJSONError(w, http.StatusBadRequest, protocol.CodeFileRequired,
			"未找到文件字段，请用 file 作为字段名")
		return
	}
	writeJSON(w, http.StatusOK, protocol.FileListData{Items: items})
}

// handleFileRawUpload 直接上传原始字节（PUT），文件名取 ?name= 或 Content-Disposition。
//
// 相比 multipart 少了一层封装，适合大文件：请求体就是文件本身，
// 浏览器用 XHR/fetch 上传能直接拿到上传进度。
func (s *Server) handleFileRawUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.authUser(w, r)
	if !ok {
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = filenameFromDisposition(r.Header.Get("Content-Disposition"))
	}
	if name == "" {
		name = "upload.bin"
	}

	limit := s.uploadLimit()
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	item, err := s.files().Save(userID, name, r.Body, limit)
	if err != nil {
		writeUploadError(w, err, limit)
		return
	}
	writeJSON(w, http.StatusOK, protocol.FileListData{Items: []protocol.FileInfo{s.fileInfo(item)}})
}

// ---- 下载 / 预览 ----

// handleFileContent 按 fileID 返回文件内容。
//
// 默认内联（浏览器能直接渲染图片、文本、PDF，前端 <img src> / fetch 都能用），
// 加 ?download=1 改为另存。可渲染但可执行脚本的类型（HTML、SVG）一律降级为
// 附件下载——同源内联等于把 XSS 开口子交给上传者。
//
// ?meta=1 只返回元信息，前端可以先看大小和类型再决定怎么展示。
func (s *Server) handleFileContent(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.authUser(w, r)
	if !ok {
		return
	}
	item, err := s.files().GetOwned(r.PathValue("id"), userID)
	if err != nil {
		writeFileAccessError(w, err)
		return
	}

	if r.URL.Query().Get("meta") == "1" {
		writeJSON(w, http.StatusOK, s.fileInfo(item))
		return
	}

	file, err := os.Open(item.Path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, protocol.CodeFileNotFound, "文件已不存在")
		return
	}
	defer file.Close()

	mimeType := item.Mime
	if mimeType == "" {
		mimeType = sniffMime(file)
	}

	inline := r.URL.Query().Get("download") != "1" && canInline(mimeType, item.Name)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", contentDisposition(disposition, item.Name))
	// nosniff 必须加：否则浏览器可能忽略我们给的 Content-Type 自行猜测，
	// 把一个声明为 text/plain 的文件当 HTML 执行
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if inline {
		// 内联内容跑在独立沙箱里，即使页面里藏了脚本也拿不到本域的 cookie 与存储
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src data:; style-src 'unsafe-inline'")
	}
	// 文件是私有的，别让中间代理缓存
	w.Header().Set("Cache-Control", "private, max-age=300")

	http.ServeContent(w, r, item.Name, time.UnixMilli(item.CreatedAt), file)
}

// handleFileList 列出当前用户仍在有效期内的文件
func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.authUser(w, r)
	if !ok {
		return
	}
	items := s.files().List(userID)
	out := make([]protocol.FileInfo, 0, len(items))
	for _, item := range items {
		out = append(out, s.fileInfo(item))
	}
	writeJSON(w, http.StatusOK, protocol.FileListData{Items: out})
}

// handleFileDelete 删除自己上传的文件
func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.authUser(w, r)
	if !ok {
		return
	}
	item, err := s.files().GetOwned(r.PathValue("id"), userID)
	if err != nil {
		writeFileAccessError(w, err)
		return
	}
	if err := s.files().Delete(item.ID); err != nil && !errors.Is(err, filestore.ErrNotFound) {
		writeJSONError(w, http.StatusInternalServerError, protocol.CodeInternal, "删除失败")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- 下行：agent 的 send_file ----

// handleSyncRequest 实现 bus.SyncMessageHandler，承接 agent 侧投给本通道的同步请求。
//
// 注册到 MessageBus 后，core.SendFileTool 按 event.Channel 路由到这里——
// 这就是 ws 必须有独立渠道标识的原因：渠道标识就是这条路由的键，
// 与 wxcom 共用标识会把 web 端的文件发到企业微信会话里去。
func (s *Server) handleSyncRequest(ctx context.Context, tp types.SyncMessageType, event *types.Event) (*types.InputMessage, error) {
	switch tp {
	case types.SyncMessageTypeSendFile:
		return s.handleSendFile(ctx, event)
	default:
		return nil, nil
	}
}

// handleSendFile 把 agent 产出的文件登记成可下载资源，并把地址推给会话。
func (s *Server) handleSendFile(ctx context.Context, event *types.Event) (*types.InputMessage, error) {
	if event == nil || event.Message == nil || len(event.Message.Media) == 0 {
		return nil, errors.New("发送失败：未提供文件")
	}

	media := event.Message.Media[0]
	filename := ""
	caption := ""
	if media.Metadata != nil {
		if v, ok := media.Metadata["filename"].(string); ok {
			filename = v
		}
		if v, ok := media.Metadata["caption"].(string); ok {
			caption = v
		}
	}
	if filename == "" {
		filename = filepath.Base(media.URL)
	}

	// owner 用发送者身份：event 由 state.BuildEvent() 构造，
	// AccountID 就是 chat.send 里写入的 userID
	owner := event.AccountID

	var (
		item *filestore.Item
		err  error
	)
	if isRemoteURL(media.URL) {
		item, err = s.saveRemoteFile(ctx, owner, media.URL, filename)
	} else {
		item, err = s.files().Adopt(owner, media.URL, filename)
	}
	if err != nil {
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	// send_file 工具会按后缀给出 image / document，缺失时这里补一次
	kind := media.Type
	if kind == "" {
		kind = mediaTypeOf(item.Name)
	}
	s.pushFileFrame(event, item, kind, caption)
	return &types.InputMessage{Content: "✅ 文件已发送: " + item.Name}, nil
}

// pushFileFrame 向该会话的所有连接推一帧带附件的事件。
//
// 用 tool_completed 而不是 message_completed：后者是「全量覆盖当前气泡」，
// 会把正文冲掉；tool_completed 属于状态事件，前端按 toolCalls/media 渲染
// 一张附件卡片即可。
//
// 这条帧不落库（turn_events）：附件是瞬时产物，历史回放重新打开时
// 文件多半已经过期，留着反而是死链。
func (s *Server) pushFileFrame(event *types.Event, item *filestore.Item, kind types.MediaType, caption string) {
	if event.SessionID == "" {
		return
	}
	out := types.Media{
		Type:     kind,
		URL:      s.fileURL(item.ID),
		MimeType: item.Mime,
		Metadata: map[string]any{
			"filename": item.Name,
			"fileID":   item.ID,
			"size":     item.Size,
		},
	}
	if caption != "" {
		out.Metadata["caption"] = caption
	}

	env := protocol.NewEnvelope(protocol.KindAgent, string(types.EventToolCompleted)).
		WithSession(event.SessionID, event.ReqID).
		// eventID 用来让前端把这条附件归到 send_file 这次调用上
		WithEventID("send_file_" + item.ID).
		WithData(&protocol.AgentData{
			Event:   string(types.EventToolCompleted),
			Role:    string(types.RoleTool),
			Content: caption,
			ToolCalls: []types.ToolCall{
				{
					ID:     "send_file_" + item.ID,
					Type:   types.ToolTypeFunction,
					Name:   "send_file",
					Result: "文件已发送: " + item.Name,
				},
			},
			Media: []types.Media{out},
		})

	s.hub.Dispatch(event.SessionID, env)
}

// saveRemoteFile 把远程地址的文件落到本地，走与上传相同的容量限制与过期策略。
func (s *Server) saveRemoteFile(ctx context.Context, owner, rawURL, filename string) (*filestore.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("文件地址无效: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败: 状态码 %d", resp.StatusCode)
	}
	return s.files().Save(owner, filename, resp.Body, s.uploadLimit())
}

// gcLoop 定期回收过期文件，直到 ctx 结束
func (s *Server) gcLoop(ctx context.Context) {
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n := s.files().GC(); n > 0 {
				s.log.Info("websocket file gc", "removed", n)
			}
		}
	}
}

// ---- 小工具 ----

func (s *Server) uploadLimit() int64 {
	if s.cfg.MaxUploadSize > 0 {
		return s.cfg.MaxUploadSize
	}
	return defaultUploadLimit
}

// authUser 取请求身份，失败时已经写好 401 响应
func (s *Server) authUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, _, ok := s.authenticate(r)
	if !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, protocol.CodeUnauthorized, "身份校验失败")
		return "", false
	}
	return userID, true
}

func (s *Server) fileInfo(item *filestore.Item) protocol.FileInfo {
	return protocol.FileInfo{
		ID:        item.ID,
		Name:      item.Name,
		Size:      item.Size,
		Mime:      item.Mime,
		URL:       s.fileURL(item.ID),
		ExpiresAt: item.ExpiresAt,
	}
}

// mediaTypeOf 按扩展名判断媒体类型，图片走 image 通道，其余按文档处理
func mediaTypeOf(name string) types.MediaType {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".heic":
		return types.MediaTypeImage
	case ".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac", ".amr", ".silk":
		return types.MediaTypeAudio
	default:
		return types.MediaTypeFile
	}
}

// canInline 判断该类型能否安全地内联到页面里。
//
// 图片、文本、PDF、音视频可以；HTML / SVG / XML 不行——它们能在同源下
// 执行脚本，内联等于让上传者往页面里注入代码。
func canInline(mimeType, name string) bool {
	mt := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch mt {
	case "text/html", "application/xhtml+xml", "image/svg+xml",
		"text/xml", "application/xml", "application/javascript", "text/javascript":
		return false
	}
	if strings.HasPrefix(mt, "image/") ||
		strings.HasPrefix(mt, "audio/") ||
		strings.HasPrefix(mt, "video/") ||
		strings.HasPrefix(mt, "text/") {
		return true
	}
	switch mt {
	case "application/pdf", "application/json":
		return true
	}
	// MIME 缺失时按后缀兜一道：.svg/.html 即便嗅探成 text/plain 也不内联
	switch strings.ToLower(filepath.Ext(name)) {
	case ".svg", ".html", ".htm", ".xhtml", ".xml":
		return false
	}
	return false
}

// sniffMime 读前 512 字节判断类型，随后把读指针复位
func sniffMime(file *os.File) string {
	var head [512]byte
	n, err := file.Read(head[:])
	if err != nil && err != io.EOF {
		return "application/octet-stream"
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "application/octet-stream"
	}
	if n == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(head[:n])
}

// contentDisposition 兼顾中文文件名：filename 给老客户端兜底（ASCII），
// filename* 按 RFC 5987 传 UTF-8
func contentDisposition(disposition, name string) string {
	ascii := make([]rune, 0, len(name))
	for _, r := range name {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			ascii = append(ascii, '_')
			continue
		}
		ascii = append(ascii, r)
	}
	fallback := strings.TrimSpace(string(ascii))
	if fallback == "" {
		fallback = "file"
	}
	return fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`,
		disposition, fallback, url.PathEscape(name))
}

// filenameFromDisposition 从 Content-Disposition 里取文件名（支持 filename*）
func filenameFromDisposition(v string) string {
	if v == "" {
		return ""
	}
	for _, part := range strings.Split(v, ";") {
		part = strings.TrimSpace(part)
		lower := strings.ToLower(part)
		if after, ok := strings.CutPrefix(lower, "filename*="); ok {
			_ = after
			// 形如 UTF-8''%E4%B8%AD.txt
			if _, rest, found := strings.Cut(part, "''"); found {
				if decoded, err := url.PathUnescape(rest); err == nil {
					return filepath.Base(decoded)
				}
			}
			continue
		}
		if after, ok := strings.CutPrefix(lower, "filename="); ok {
			_ = after
			raw := strings.TrimSpace(part[len("filename="):])
			raw = strings.Trim(raw, `"`)
			if raw != "" {
				return filepath.Base(raw)
			}
		}
	}
	return ""
}

func isRemoteURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// isTooLarge 识别 MaxBytesReader 的超限错误
func isTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr) || errors.Is(err, filestore.ErrTooLarge)
}

func writeUploadError(w http.ResponseWriter, err error, limit int64) {
	switch {
	case errors.Is(err, filestore.ErrTooLarge), isTooLarge(err):
		writeJSONError(w, http.StatusRequestEntityTooLarge, protocol.CodeFileTooLarge,
			fmt.Sprintf("文件超过上限 %d 字节", limit))
	case errors.Is(err, filestore.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, protocol.CodeFileNotFound, "文件不存在")
	default:
		writeJSONError(w, http.StatusInternalServerError, protocol.CodeUploadFailed, err.Error())
	}
}

func writeFileAccessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filestore.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, protocol.CodeFileNotFound, "文件不存在或已过期")
	case errors.Is(err, filestore.ErrForbidden):
		writeJSONError(w, http.StatusForbidden, protocol.CodeForbidden, "无权访问该文件")
	default:
		writeJSONError(w, http.StatusInternalServerError, protocol.CodeInternal, "读取失败")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}

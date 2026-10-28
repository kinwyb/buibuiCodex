// 本文件验证「操作 agent 工作区」的 HTTP 接口：列目录、读、上传、删除、越界拒绝。

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/agentfs"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/pathmap"
)

// fakeAgentFS 顶替 core.Manager：测试不需要拉起整个 agent 栈
type fakeAgentFS struct {
	agents map[string]*agentfs.FS
}

func (f *fakeAgentFS) AgentFS(name string) (*agentfs.FS, bool) {
	fs, ok := f.agents[name]
	return fs, ok
}

type agentEnv struct {
	server *Server
	ts     *httptest.Server
	// wsHost 该 agent 的 <WorkSpace>/workspace 宿主机目录，断言落盘用
	wsHost string
}

// newAgentEnv 起一个带 agent 文件端点的测试服务。enabled=false 时不注册路由，
// 对应配置里 agent_files 关着的情况。
func newAgentEnv(t *testing.T, enabled bool) *agentEnv {
	t.Helper()
	msgBus := bus.NewMessageBus(8)
	t.Cleanup(func() { _ = msgBus.Close() })

	root := t.TempDir()
	wsHost := filepath.Join(root, "codex", "workspace")
	tmpHost := filepath.Join(root, "codex", "tmp")
	for _, dir := range []string{wsHost, tmpHost} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("准备工作区失败: %v", err)
		}
	}

	agentFS := &fakeAgentFS{agents: map[string]*agentfs.FS{
		"codex": agentfs.New([]pathmap.PathMapping{
			{Host: wsHost, Container: "/workspace"},
			{Host: tmpHost, Container: "/tmp"},
		}),
	}}

	s := New(Options{
		Config: &config.WebSocketConfig{
			Enabled:       true,
			SendBuffer:    16,
			FileDir:       t.TempDir(),
			MaxUploadSize: 4096,
			AgentFiles:    enabled,
		},
		Bus:        msgBus,
		Agents:     fakeAgents{"codex"},
		AgentFiles: agentFS,
		Authenticator: func(token string) (string, string, bool) {
			if token == "" {
				return "", "", false
			}
			return token, "", true
		},
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	s.registerFileRoutes(mux)
	if enabled {
		s.registerAgentFileRoutes(mux)
	}

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return &agentEnv{server: s, ts: ts, wsHost: wsHost}
}

// doJSON 发请求并解析 JSON 响应
func doJSON(t *testing.T, method, url string, body io.Reader, contentType string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestAgentRoots(t *testing.T) {
	env := newAgentEnv(t, true)

	resp, body := doJSON(t, http.MethodGet, env.ts.URL+"/ws/agents?token=alice", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("应 200，实际 %d: %s", resp.StatusCode, body)
	}
	var out agentRootsData
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("解析失败: %v (%s)", err, body)
	}
	if len(out.Agents) != 1 || out.Agents[0].Name != "codex" {
		t.Fatalf("应返回 codex: %+v", out)
	}
	roots := out.Agents[0].Roots
	if len(roots) != 2 || roots[0].Container != "/workspace" || !roots[0].Exists {
		t.Fatalf("根不符: %+v", roots)
	}
}

func TestAgentFileList(t *testing.T) {
	env := newAgentEnv(t, true)
	if err := os.WriteFile(filepath.Join(env.wsHost, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.wsHost, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	resp, body := doJSON(t, http.MethodGet,
		env.ts.URL+"/ws/agents/codex/files?token=alice&path=/workspace", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("应 200，实际 %d: %s", resp.StatusCode, body)
	}
	var out agentDirData
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("解析失败: %v (%s)", err, body)
	}
	if out.Path != "/workspace" || out.Parent != "" {
		t.Fatalf("根目录不该有 parent: %+v", out)
	}
	if len(out.Items) != 2 {
		t.Fatalf("应有 2 项，实际 %d", len(out.Items))
	}
	for _, item := range out.Items {
		if !strings.HasPrefix(item.Path, "/workspace/") {
			t.Fatalf("条目应带容器路径: %+v", item)
		}
	}

	// 子目录要看得到 parent
	resp2, body2 := doJSON(t, http.MethodGet,
		env.ts.URL+"/ws/agents/codex/files?token=alice&path=/workspace/docs", nil, "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("应 200，实际 %d: %s", resp2.StatusCode, body2)
	}
	var sub agentDirData
	_ = json.Unmarshal(body2, &sub)
	if sub.Parent != "/workspace" {
		t.Fatalf("子目录 parent 应为 /workspace，实际 %q", sub.Parent)
	}
}

func TestAgentFileOutOfScope(t *testing.T) {
	env := newAgentEnv(t, true)

	cases := []struct {
		path string
		want int
		code string
	}{
		{"/etc", http.StatusForbidden, "out_of_scope"},
		{"/workspace/../../etc", http.StatusForbidden, "out_of_scope"},
		{"relative/a.txt", http.StatusBadRequest, "path_not_absolute"},
		{"/workspace/nope.txt", http.StatusNotFound, "file_not_found"},
	}
	for _, c := range cases {
		resp, body := doJSON(t, http.MethodGet,
			env.ts.URL+"/ws/agents/codex/files?token=alice&path="+c.path, nil, "")
		if resp.StatusCode != c.want {
			t.Errorf("path=%s 应 %d，实际 %d: %s", c.path, c.want, resp.StatusCode, body)
			continue
		}
		var errBody map[string]string
		_ = json.Unmarshal(body, &errBody)
		if errBody["code"] != c.code {
			t.Errorf("path=%s 错误码应 %s，实际 %s", c.path, c.code, errBody["code"])
		}
	}
}

func TestAgentFilePutGetDelete(t *testing.T) {
	env := newAgentEnv(t, true)
	target := env.ts.URL + "/ws/agents/codex/file?token=alice&path=/workspace/notes.md"

	// 上传（原始字节）
	resp, body := doJSON(t, http.MethodPut, target, strings.NewReader("# 标题"), "text/markdown")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("写入应 200，实际 %d: %s", resp.StatusCode, body)
	}
	var saved agentEntryData
	if err := json.Unmarshal(body, &saved); err != nil {
		t.Fatalf("解析失败: %v (%s)", err, body)
	}
	if saved.Entry.Name != "notes.md" || saved.Entry.Path != "/workspace/notes.md" {
		t.Fatalf("条目不符: %+v", saved.Entry)
	}

	// 内容落到宿主机工作区里
	onDisk, err := os.ReadFile(filepath.Join(env.wsHost, "notes.md"))
	if err != nil {
		t.Fatalf("文件未落盘: %v", err)
	}
	if string(onDisk) != "# 标题" {
		t.Fatalf("内容不符: %q", onDisk)
	}

	// 读回来：默认内联
	resp2, got := doJSON(t, http.MethodGet, target, nil, "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("读取应 200，实际 %d", resp2.StatusCode)
	}
	if string(got) != "# 标题" {
		t.Fatalf("内容不符: %q", got)
	}
	if disp := resp2.Header.Get("Content-Disposition"); !strings.HasPrefix(disp, "inline") {
		t.Fatalf("默认应内联，实际 %q", disp)
	}

	// meta 只给元信息
	resp3, metaBody := doJSON(t, http.MethodGet, target+"&meta=1", nil, "")
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("meta 应 200，实际 %d", resp3.StatusCode)
	}
	var entry agentfs.Entry
	if err := json.Unmarshal(metaBody, &entry); err != nil {
		t.Fatalf("meta 应为 JSON: %v", err)
	}
	if entry.Size != int64(len("# 标题")) {
		t.Fatalf("大小不符: %+v", entry)
	}

	// 删除
	resp4, _ := doJSON(t, http.MethodDelete, target, nil, "")
	if resp4.StatusCode != http.StatusNoContent {
		t.Fatalf("删除应 204，实际 %d", resp4.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(env.wsHost, "notes.md")); !os.IsNotExist(err) {
		t.Fatal("文件应已删除")
	}
}

func TestAgentFileMultipartUpload(t *testing.T) {
	env := newAgentEnv(t, true)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "报表.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("a,b\n1,2\n"))
	_ = mw.Close()

	resp, body := doJSON(t, http.MethodPost,
		env.ts.URL+"/ws/agents/codex/file?token=alice&path=/workspace", &buf, mw.FormDataContentType())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("表单上传应 200，实际 %d: %s", resp.StatusCode, body)
	}
	var out agentEntryData
	_ = json.Unmarshal(body, &out)
	if out.Entry.Path != "/workspace/报表.csv" {
		t.Fatalf("路径应拼接目录与文件名: %+v", out.Entry)
	}
	if _, err := os.Stat(filepath.Join(env.wsHost, "报表.csv")); err != nil {
		t.Fatalf("文件未落盘: %v", err)
	}
}

func TestAgentFileDeleteDirNeedsRecursive(t *testing.T) {
	env := newAgentEnv(t, true)
	dir := filepath.Join(env.wsHost, "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	base := env.ts.URL + "/ws/agents/codex/file?token=alice&path=/workspace/docs"

	resp, body := doJSON(t, http.MethodDelete, base, nil, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非递归删目录应 400，实际 %d: %s", resp.StatusCode, body)
	}
	var errBody map[string]string
	_ = json.Unmarshal(body, &errBody)
	if errBody["code"] != "is_directory" {
		t.Fatalf("错误码应为 is_directory，实际 %s", errBody["code"])
	}

	resp2, _ := doJSON(t, http.MethodDelete, base+"&recursive=1", nil, "")
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("递归删除应 204，实际 %d", resp2.StatusCode)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("目录应已删除")
	}
}

func TestAgentFileRootProtected(t *testing.T) {
	env := newAgentEnv(t, true)
	resp, body := doJSON(t, http.MethodDelete,
		env.ts.URL+"/ws/agents/codex/file?token=alice&path=/workspace&recursive=1", nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("删根应 403，实际 %d: %s", resp.StatusCode, body)
	}
	if _, err := os.Stat(env.wsHost); err != nil {
		t.Fatalf("工作区根不应被删: %v", err)
	}
}

func TestAgentFileUnknownAgent(t *testing.T) {
	env := newAgentEnv(t, true)
	resp, body := doJSON(t, http.MethodGet,
		env.ts.URL+"/ws/agents/nobody/files?token=alice&path=/workspace", nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("未知 agent 应 404，实际 %d: %s", resp.StatusCode, body)
	}
	var errBody map[string]string
	_ = json.Unmarshal(body, &errBody)
	if errBody["code"] != "agent_not_found" {
		t.Fatalf("错误码应为 agent_not_found，实际 %s", errBody["code"])
	}
}

func TestAgentFileAuthRequired(t *testing.T) {
	env := newAgentEnv(t, true)
	resp, err := http.Get(env.ts.URL + "/ws/agents/codex/files?path=/workspace")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("无 token 应 401，实际 %d", resp.StatusCode)
	}
}

// TestAgentFilesDisabled 关掉开关时端点不注册：请求落到 mux 的默认 404，
// 与「路径写错」无法区分，这正是关闭状态下应该有的表现。
func TestAgentFilesDisabled(t *testing.T) {
	env := newAgentEnv(t, false)
	resp, err := http.Get(env.ts.URL + "/ws/agents?token=alice")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("未启用时应 404，实际 %d", resp.StatusCode)
	}
}

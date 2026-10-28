// 本文件验证 ws 的文件收发：HTTP 上传/下载端点，以及 fileID 在 chat.send 里的还原。

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/websocket"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/config"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

type fileTestEnv struct {
	server *Server
	ts     *httptest.Server
	bus    *bus.MessageBus
}

// newFileTestEnv 起一个带文件端点的测试服务：文件端点与 ws 挂在同一个 mux 上，
// 与实际部署一致（同一个端口）。
func newFileTestEnv(t *testing.T) *fileTestEnv {
	t.Helper()
	msgBus := bus.NewMessageBus(8)
	t.Cleanup(func() { _ = msgBus.Close() })

	s := New(Options{
		Config: &config.WebSocketConfig{
			Enabled:       true,
			SendBuffer:    16,
			FileDir:       t.TempDir(),
			MaxUploadSize: 4096,
		},
		Bus:    msgBus,
		Agents: fakeAgents{"codex"},
		// 默认鉴权把任何 token 都折成同一个固定用户，归属校验就测不出来了。
		// 这里让 token 直接充当 userID，和后端接入真实鉴权后的形态一致。
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

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return &fileTestEnv{server: s, ts: ts, bus: msgBus}
}

// upload 走 multipart 上传，返回响应与解析后的载荷
func upload(t *testing.T, ts *httptest.Server, token, name string, content []byte) (*http.Response, protocol.FileListData) {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("构造表单失败: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("写表单失败: %v", err)
	}
	_ = mw.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/ws/files?token="+token, &buf)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("上传请求失败: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	var out protocol.FileListData
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

// getFile 以指定身份取文件。query 为附加查询串（如 "download=1"），可为空。
func getFile(t *testing.T, ts *httptest.Server, token, path string, query ...string) (*http.Response, []byte) {
	t.Helper()
	target := ts.URL + path + "?token=" + token
	for _, q := range query {
		if q != "" {
			target += "&" + q
		}
	}
	resp, err := http.Get(target)
	if err != nil {
		t.Fatalf("下载请求失败: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

func TestFileUploadAndDownload(t *testing.T) {
	env := newFileTestEnv(t)
	content := []byte("列1,列2\n1,2\n")

	resp, data := upload(t, env.ts, "alice", "数据.csv", content)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("上传应成功，实际 %d", resp.StatusCode)
	}
	if len(data.Items) != 1 {
		t.Fatalf("应返回 1 个文件，实际 %d", len(data.Items))
	}
	info := data.Items[0]
	if info.ID == "" || info.Name != "数据.csv" || info.Size != int64(len(content)) {
		t.Fatalf("文件元信息不符: %+v", info)
	}
	if info.URL != "/ws/files/"+info.ID {
		t.Fatalf("URL 应是相对地址，实际 %q", info.URL)
	}

	// 默认内联：前端 <img>/fetch 能直接拿到内容
	resp2, body := getFile(t, env.ts, "alice", info.URL)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("下载失败: %d", resp2.StatusCode)
	}
	if !bytes.Equal(body, content) {
		t.Fatalf("内容不一致: %q", body)
	}
	if disp := resp2.Header.Get("Content-Disposition"); !strings.HasPrefix(disp, "inline") {
		t.Fatalf("默认应内联，实际 %q", disp)
	}
	if resp2.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("应设置 nosniff，避免浏览器自行猜测类型")
	}

	// ?download=1 改为另存
	resp3, _ := getFile(t, env.ts, "alice", info.URL, "download=1")
	if disp := resp3.Header.Get("Content-Disposition"); !strings.HasPrefix(disp, "attachment") {
		t.Fatalf("download=1 应为附件，实际 %q", disp)
	}
}

func TestFileDownloadOwnership(t *testing.T) {
	env := newFileTestEnv(t)
	_, data := upload(t, env.ts, "alice", "secret.txt", []byte("secret"))
	path := data.Items[0].URL

	if resp, _ := getFile(t, env.ts, "bob", path); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("他人应被拒绝，实际 %d", resp.StatusCode)
	}

	resp, err := http.Get(env.ts.URL + path)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("无 token 应 401，实际 %d", resp.StatusCode)
	}
}

func TestFileDownloadNotFound(t *testing.T) {
	env := newFileTestEnv(t)
	resp, _ := getFile(t, env.ts, "alice", "/ws/files/not-a-real-id")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在应 404，实际 %d", resp.StatusCode)
	}
}

func TestFileUploadTooLarge(t *testing.T) {
	env := newFileTestEnv(t) // 上限 4096
	resp, _ := upload(t, env.ts, "alice", "big.bin", make([]byte, 8192))
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("超限应 413，实际 %d", resp.StatusCode)
	}
}

func TestFileUploadMissingField(t *testing.T) {
	env := newFileTestEnv(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("other", "x")
	_ = mw.Close()

	req, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/ws/files?token=alice", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺文件字段应 400，实际 %d", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["code"] != protocol.CodeFileRequired {
		t.Fatalf("错误码应为 %s，实际 %q", protocol.CodeFileRequired, body["code"])
	}
}

func TestFileRawUploadAndDelete(t *testing.T) {
	env := newFileTestEnv(t)

	req, _ := http.NewRequest(http.MethodPut, env.ts.URL+"/ws/files?token=alice&name=raw.txt", strings.NewReader("raw-body"))
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT 上传应成功，实际 %d", resp.StatusCode)
	}
	var out protocol.FileListData
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Items) != 1 || out.Items[0].Name != "raw.txt" {
		t.Fatalf("文件名应取 query 参数: %+v", out.Items)
	}

	id := out.Items[0].ID
	delReq, _ := http.NewRequest(http.MethodDelete, env.ts.URL+"/ws/files/"+id+"?token=alice", nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("删除请求失败: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("删除应 204，实际 %d", delResp.StatusCode)
	}

	if resp, _ := getFile(t, env.ts, "alice", "/ws/files/"+id); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("删除后应 404，实际 %d", resp.StatusCode)
	}
}

func TestFileMeta(t *testing.T) {
	env := newFileTestEnv(t)
	_, data := upload(t, env.ts, "alice", "note.md", []byte("# hi"))

	resp, body := getFile(t, env.ts, "alice", data.Items[0].URL, "meta=1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("meta 应 200，实际 %d", resp.StatusCode)
	}
	var info protocol.FileInfo
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("meta 应为 JSON: %v (%s)", err, body)
	}
	if info.ID != data.Items[0].ID || info.Name != "note.md" {
		t.Fatalf("meta 不符: %+v", info)
	}
}

// dialURL 直连指定地址的 ws
func dialURL(t *testing.T, wsURL string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("握手失败: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// dialFileEnv 连上 ws 并读完 hello
func dialFileEnv(t *testing.T, env *fileTestEnv, token string) (*websocket.Conn, protocol.HelloData) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(env.ts.URL, "http") + "/ws?token=" + token
	conn := dialURL(t, wsURL)
	var hello protocol.HelloData
	first := readEnvelope(t, conn)
	if first.Type != protocol.TypeHello {
		t.Fatalf("首帧应为 hello，实际 %s", first.Type)
	}
	if err := first.DecodeData(&hello); err != nil {
		t.Fatalf("解析 hello 失败: %v", err)
	}
	return conn, hello
}

// TestChatSendWithFileID 验证 fileID 被还原成 media，且路径是宿主机本地路径。
// 容器路径转换由 core.Manager 在投递后完成，不在命令层。
func TestChatSendWithFileID(t *testing.T) {
	env := newFileTestEnv(t)
	_, data := upload(t, env.ts, "alice", "报表.xlsx", []byte("fake-xlsx"))

	sub, err := env.bus.SubscribeInput()
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}
	defer sub.Close()

	conn, hello := dialFileEnv(t, env, "alice")

	send := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend)
	send.WithData(&protocol.ChatSendData{Content: "看看这个", FileIDs: []string{data.Items[0].ID}})
	if err := conn.WriteJSON(send); err != nil {
		t.Fatalf("发送失败: %v", err)
	}

	select {
	case msg := <-sub.Channel:
		if len(msg.Media) != 1 {
			t.Fatalf("应还原出 1 个 media，实际 %d", len(msg.Media))
		}
		media := msg.Media[0]
		if media.Type != types.MediaTypeFile {
			t.Fatalf("xlsx 应归类为 document，实际 %s", media.Type)
		}
		if !filepath.IsAbs(media.URL) {
			t.Fatalf("应给宿主机绝对路径，实际 %q", media.URL)
		}
		if media.Metadata["filename"] != "报表.xlsx" {
			t.Fatalf("必须带 filename，否则 agent 侧取文件名会出错: %+v", media.Metadata)
		}
		if media.Metadata["fileID"] != data.Items[0].ID {
			t.Fatalf("metadata 应带回 fileID: %+v", media.Metadata)
		}
		if _, err := os.Stat(media.URL); err != nil {
			t.Fatalf("media 指向的文件应存在: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("未收到投递的消息")
	}
	_ = hello
}

// TestChatSendFileIDOwnership 别人的 fileID 不能引用
func TestChatSendFileIDOwnership(t *testing.T) {
	env := newFileTestEnv(t)
	_, data := upload(t, env.ts, "alice", "a.txt", []byte("x"))

	conn, _ := dialFileEnv(t, env, "bob")
	send := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend)
	send.WithData(&protocol.ChatSendData{Content: "hi", FileIDs: []string{data.Items[0].ID}})
	if err := conn.WriteJSON(send); err != nil {
		t.Fatalf("发送失败: %v", err)
	}

	env1 := readEnvelope(t, conn)
	if env1.Type != protocol.TypeError {
		t.Fatalf("应返回错误，实际 %s", env1.Type)
	}
	var body protocol.ErrorBody
	_ = env1.DecodeData(&body)
	if body.Code != protocol.CodeBadRequest {
		t.Fatalf("错误码应为 %s，实际 %s", protocol.CodeBadRequest, body.Code)
	}
}

// TestSendFilePushesFrame 验证 agent 的 send_file 会把下载地址推给会话。
func TestSendFilePushesFrame(t *testing.T) {
	env := newFileTestEnv(t)

	// 准备一个 agent 产出的文件（send_file 工具此前已 ToHost，这里给的就是宿主机路径）
	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.csv")
	if err := os.WriteFile(outPath, []byte("a,b\n"), 0o644); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}

	conn, hello := dialFileEnv(t, env, "alice")
	sessionID := "ws_" + hello.UserID + "_" + hello.ChatID

	// 先建立会话订阅：hub 只投给订阅了该会话的连接
	send := protocol.NewEnvelope(protocol.KindClient, protocol.TypeChatSend)
	send.WithData(&protocol.ChatSendData{Content: "生成报表", ChatID: hello.ChatID})
	if err := conn.WriteJSON(send); err != nil {
		t.Fatalf("chat.send 失败: %v", err)
	}
	_ = readEnvelope(t, conn) // ack

	event := &types.Event{
		SessionID: sessionID,
		ReqID:     "req-1",
		Channel:   env.server.opts.Channel,
		AccountID: "alice",
		Message: &types.Message{
			Media: []types.Media{{
				Type:     types.MediaTypeFile,
				URL:      outPath,
				Metadata: map[string]any{"caption": "本月报表"},
			}},
		},
	}

	resp, err := env.server.handleSyncRequest(context.Background(), types.SyncMessageTypeSendFile, event)
	if err != nil {
		t.Fatalf("处理 send_file 失败: %v", err)
	}
	if resp == nil || !strings.Contains(resp.Content, "report.csv") {
		t.Fatalf("工具结果应说明已发送: %+v", resp)
	}

	env1 := readEnvelope(t, conn)
	if env1.Type != string(types.EventToolCompleted) {
		t.Fatalf("应收到 tool_completed，实际 %s", env1.Type)
	}
	var data protocol.AgentData
	if err := env1.DecodeData(&data); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(data.Media) != 1 {
		t.Fatalf("应带 1 个附件，实际 %d", len(data.Media))
	}
	media := data.Media[0]
	if !strings.HasPrefix(media.URL, "/ws/files/") {
		t.Fatalf("附件地址应是下载端点，实际 %q", media.URL)
	}
	if media.Metadata["filename"] != "report.csv" {
		t.Fatalf("附件应带文件名: %+v", media.Metadata)
	}

	// 拿这个地址真的能下载到内容
	id := strings.TrimPrefix(media.URL, "/ws/files/")
	respHTTP, body := getFile(t, env.ts, "alice", "/ws/files/"+id)
	if respHTTP.StatusCode != http.StatusOK {
		t.Fatalf("下载失败: %d", respHTTP.StatusCode)
	}
	if string(body) != "a,b\n" {
		t.Fatalf("内容不符: %q", body)
	}
}

// TestSyncHandlerRegistered 验证 Start 之后本通道接管了同步请求，
// 这是 send_file 能回流的前提（渠道标识就是路由键）。
func TestSyncHandlerRegistered(t *testing.T) {
	env := newFileTestEnv(t)
	// 未 Start 时不应注册
	if env.server.syncHandlerRegistered {
		t.Fatal("未启动时不应注册")
	}
	if err := env.server.attachSyncHandler(); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if !env.server.syncHandlerRegistered {
		t.Fatal("应记录已注册")
	}
	// 重复注册要幂等，避免 Start 重试时把订阅搞乱
	if err := env.server.attachSyncHandler(); err != nil {
		t.Fatalf("重复注册应幂等: %v", err)
	}
	env.server.detachSyncHandler()
	if env.server.syncHandlerRegistered {
		t.Fatal("解除后应复位")
	}
}

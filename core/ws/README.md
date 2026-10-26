# core/ws — WebSocket 消息服务

面向 Web 前端的实时消息通道：把 agent 的流式输出转发给浏览器，同时接收浏览器发来的提问与中断指令。

```
下行：agent ──> core.Manager ──> MessageBus ──> bridge ──> conn.Hub ──> Client ──WS──> 浏览器
上行：浏览器 ──WS──> Client ──> server ──> command ──> MessageBus ──> core.Manager
```

ws 不直接调用 agent，也不经过 `channel` 层：上行把消息投入 `MessageBus`，由现有的 `core.Manager` 消费；下行订阅同一总线的 `chatEvents`，转成自己的协议再分发。

---

## 1. 快速开始

### 1.1 配置

`buibui.json` 增加（该文件已被 `.gitignore` 忽略，需在本机配置）：

```json
"websocket": {
  "enabled": true,
  "host": "0.0.0.0",
  "port": 8765,
  "path": "/ws",
  "auth_token": "",
  "users": { "tok-alice": "alice", "tok-bob": "bob" },
  "allow_origins": ["*"],
  "max_clients": 100,
  "send_buffer": 256,
  "heartbeat_interval": 30,
  "read_timeout": 60,
  "write_timeout": 10,
  "forward_all_channels": false,
  "default_agent": "codex"
}
```

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `enabled` | false | 是否启用 |
| `host` / `port` | 0.0.0.0 / 8765 | 监听地址 |
| `path` | `/ws` | 升级路径 |
| `auth_token` | 空 | 单用户模式：校验通过后身份固定为 `web`；为空则不校验 |
| `users` | 空 | 多用户模式：`token → userID`，配置后优先于 `auth_token`。userID 只允许字母、数字、短横线 |
| `allow_origins` | 空 | 为空不限制；支持 `*`、完整 Origin、主机名 |
| `max_clients` | 100 | 超过返回 503 |
| `send_buffer` | 256 | 单连接发送队列长度，满时的策略见第 6 节 |
| `heartbeat_interval` | 30 | 控制帧 ping 间隔（秒） |
| `read_timeout` | 60 | 读超时（秒），需大于心跳间隔 |
| `write_timeout` | 10 | 单次写超时（秒） |
| `forward_all_channels` | false | 是否转发其他渠道（如 wxcom）的事件 |
| `default_agent` | 空 | 前端未指定 agent 时使用 |

### 1.2 启动

`main.go` 已在 `manager.Start` 之后自动启动，接入代码（节选）：

```go
wsServer := ws.New(ws.Options{
    Config:    cfg.WebSocket,
    Bus:       msgBus,
    Canceller: manager, // 接口：Cancel(sessionKey)
    Agents:    manager, // 接口：ListAgents() []string
    Logger:    slog.Default(),
})
defer wsServer.Stop(shutdownCtx) // 建议带 5s 超时的 context
```

### 1.3 联调

```bash
wscat -c "ws://127.0.0.1:8765/ws?token=xxx"
# 连上后先收到 hello，然后可直接发：
{"v":1,"id":"c1","kind":"client","type":"chat.send","data":{"chatID":"demo","content":"你好"}}
```

浏览器端最小示例：

```js
const ws = new WebSocket("ws://127.0.0.1:8765/ws");
ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  switch (msg.kind) {
    case "agent":
      if (msg.data.delta) el.append(msg.data.content);   // 增量：追加
      else if (msg.type === "message_completed") el.textContent = msg.data.content; // 全量：覆盖
      break;
    case "system":
      if (msg.type === "error") console.error(msg.data);
      break;
  }
};
```

---

## 2. 目录结构

```
core/ws/
├── ws.go            对外门面：Options / Server / New（main 只用这一层）
├── protocol/        协议定义，无内部依赖
│   ├── envelope.go  信封、Kind、版本、IsDelta
│   ├── payload.go   各消息域的 Data 结构 + 流式渲染语义
│   └── codec.go     编解码与校验、1MB 帧上限
├── conn/            连接层
│   ├── client.go    单连接：读写泵、心跳、背压、ack/error 回复
│   └── hub.go       连接注册表、按会话订阅、广播、全局序号
├── bridge/          总线桥接：types.Event → protocol.Envelope
├── command/         上行命令：注册表 + 内置 4 个命令
└── server/          HTTP 升级、鉴权、Origin、生命周期、组装
```

依赖方向（单向，无环）：

```
server ──> command ──> conn ──> protocol
   │         │
   ├──> bridge ──> protocol
   └──> config / bus
```

两个关键接口让下游可以反向调用而不形成环：

- `conn.Handler`：连接收到上行消息时回调 `DispatchCommand`，由 `server.Server` 实现（所以 `conn` 不 import `server`）。
- `bridge.Dispatcher`：只有 `Dispatch(sessionID, env) int`，由 `conn.Hub` 实现。

---

## 3. 协议

### 3.1 信封

上下行统一结构，字段不可随意增删（前端按此解析）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `v` | int | 协议版本，当前 `1` |
| `id` | string | 消息唯一 ID，发送方生成 |
| `kind` | string | 消息域，`agent` / `system` / `client` / 预留 `cron`、`notice` |
| `type` | string | 具体类型，含义由 `kind` 决定 |
| `sessionID` | string | 会话标识，下行事件必带 |
| `reqID` | string | 一次提问的 ID，同一轮事件共享 |
| `replyTo` | string | 响应关联的上行 `id` |
| `seq` | int64 | 服务端全局单调序号，用于排序与排查 |
| `chunk` | int | 流内序号，对齐 `Event.ChunkIndex` |
| `ts` | int64 | 毫秒时间戳 |
| `data` | object | 载荷，按 `kind + type` 解释 |

`kind` 是扩展位：前端遇到未知 `kind` 应**忽略而不是报错**，这样服务端新增消息域不会打断老页面。

版本策略：新增 `kind` / `type` 属于向后兼容，不升 `v`；字段语义变更或删除才升版本。

### 3.2 下行 · system

| type | 时机 | data |
| --- | --- | --- |
| `hello` | 连接建立后第一帧 | `clientID`、`userID`、`protocol`、`agents`、`defaultAgent`、`heartbeat` |
| `ack` | 上行请求受理 | `ok`、`sessionID`、`reqID` |
| `pong` | 收到 `ping` | `clientTS`、`serverTS` |
| `error` | 任何失败 | `code`、`message`、`detail` |
| `agent.list` | 收到同名上行命令 | `agents`、`defaultAgent` |
| `session.history` | 收到同名上行命令 | `sessionID`、`threads[]`（见 5.1） |
| `shutdown` | 服务停止前 | `reason` |

### 3.3 下行 · agent

`type` 取值与 `types.EventType` 一致：

| type | 含义 |
| --- | --- |
| `user_message_start` / `user_message_completed` | 用户提问开始 / 受理完成 |
| `reasoning_start` / `reasoning_delta` / `reasoning_completed` | 思考过程 |
| `message_start` / `message_delta` / `message_completed` | 回复正文 |
| `tool_start` / `tool_completed` | 工具调用与结果 |
| `approval` | 需要审批 |
| `error` | agent 执行失败 |
| `finished` | 本轮结束 |

data 结构见 `protocol.AgentData`：`event`、`agent`、`role`、`delta`、`content`、`reasoning`、`toolCalls`、`media`、`approval`、`error`、`meta`。

#### 流式渲染语义（重要）

前端只需要看 `delta` 一个字段：

```
delta == true    增量片段，追加到当前消息
delta == false   全量快照，整体覆盖当前消息
```

| 事件 | delta | 前端处理 |
| --- | --- | --- |
| `message_start` | false | `content` 通常为空，重置当前气泡 |
| `message_delta` | true | 追加 |
| `message_completed` | false | 全量覆盖，收敛 |
| `reasoning_delta` | true | 追加思考 |
| `reasoning_completed` | false | 覆盖思考 |
| `tool_*` / `approval` / `error` / `finished` | false | 状态事件，不参与文本追加 |

这条约定不是可选的：服务端在网络拥塞时**会丢弃 delta**，依靠 `message_completed` 的全量内容让界面收敛。如果前端把 `completed` 当增量拼接，丢帧后内容就永远修不回来了。

### 3.4 上行 · client

`kind` 必须是 `client`，否则服务端返回 `unsupported`。

| type | data | 说明 |
| --- | --- | --- |
| `chat.send` | `sessionID?`、`chatID?`、`agent?`、`content`、`media`、`metadata` | 发送提问 |
| `chat.cancel` | `sessionID?`、`chatID?` | 中断当前会话 |
| `ping` | `ts` | 应用层心跳，测 RTT |
| `agent.list` | — | 拉取可用 agent |
| `session.history` | `sessionID?`、`chatID?`、`threadID?`、`beforeSeq?`、`since?`、`limit?` | 拉取历史消息，见 5.1 |

`chat.send` 的 `chatID` 为空时由服务端生成，并在 `ack.sessionID` 中返回完整会话键；后续请求建议同时带上 `sessionID`。

### 3.5 错误码

| code | 场景 |
| --- | --- |
| `bad_request` | JSON 非法、缺字段、content 与 media 同时为空 |
| `unauthorized` | 实际以 HTTP 401 返回，不进协议层 |
| `forbidden` | 访问不属于当前用户 `sessionID` 的会话 |
| `unsupported` | 未知 `kind` 或 `type` |
| `agent_not_found` | 指定的 agent 不存在 |
| `too_many_clients` | 以 HTTP 503 返回 |
| `internal` | 投递失败等服务端异常 |
| `busy` | 预留 |

---

## 4. 一次完整交互

```
浏览器                                   服务端                        Manager/Agent
  │  chat.send {chatID:"c1",content:"你好"}  │                              │
  ├────────────────────────────────────────>│                              │
  │                                         │ 构造 InputMessage{Channel:"ws"}
  │                                         │ PublishInput ────────────────>│
  │  ack {sessionID:"ws_web_c1",reqID:"r1"} │                              │
  │<────────────────────────────────────────┤                              │
  │                                         │        user_message_start    │
  │<────────────────────────────────────────┤<─────────────────────────────┤
  │  message_start (delta=false)            │                              │
  │<────────────────────────────────────────┤                              │
  │  message_delta (delta=true) ... N 帧    │                              │
  │<────────────────────────────────────────┤                              │
  │  message_completed (delta=false,全量)   │                              │
  │<────────────────────────────────────────┤                              │
  │  finished                               │                              │
  │<────────────────────────────────────────┤                              │
```

---

## 5. 会话与路由

- 会话键为 `Channel_UserID_ChatID`，即 `ws_<userID>_<chatID>`。`AccountID` 承载用户身份，不同用户的同名 chatID 落在不同会话；单用户模式下 userID 固定为 `web`，键仍是 `ws_web_<chatID>`，与改造前兼容。
- `chat.send` 与 `session.history` 都会让连接**自动订阅**该 sessionID；`Hub` 按订阅关系分发，多标签、多用户之间不会串消息。
- 事件只转发给订阅了对应 `sessionID` 的连接，未订阅则不投递（返回 0）。
- 默认只转发 `event.Channel == "ws"` 的事件，微信等渠道的回复不会串到 Web。`forward_all_channels=true` 可放开，但仍需客户端订阅对应会话才会收到。

### 5.1 历史消息

数据层级是 `session → thread → turn`：

- `session`：一个会话（`ws_<userID>_<chatID>`）
- `thread`：会话下某个 agent 的线程，换过 agent 就会新建，所以一个会话可能有多个 thread
- `turn`：一次问答，`Question` / `Answer` 落库，`Answer` 是完成时的全量文本

上行 `session.history` 的三种用法：

| 场景 | 参数 | 说明 |
| --- | --- | --- |
| 打开历史对话 | `{sessionID}` | 返回该会话下所有线程与轮次 |
| 向上翻页 | `{sessionID, threadID, beforeSeq}` | 传上一页第一条的 `seq` |
| 断线补发 | `{sessionID, since}` | 传本地最后一条轮次的 `createdAt`（毫秒），只取之后的 |

响应示例：

```json
{
  "sessionID": "ws_web_c1",
  "threads": [
    {
      "threadID": "th-1", "agent": "codex",
      "createdAt": 1761432000000, "lastUpdate": 1761432600000,
      "turns": [
        {"seq": 12, "reqID": "r1", "threadID": "th-1", "turnID": "tu-1", "agent": "codex",
         "question": "分析昨天良率", "answer": "生产良率为 98.2%", "completed": true,
         "createdAt": 1761432000000, "completedAt": 1761432120000}
      ],
      "hasMore": false
    }
  ]
}
```

约定：

- `threads` 按创建时间升序，`turns` 按 `seq` 升序，前端可直接顺序渲染，无需再排序。
- 拉取历史会自动订阅该 `sessionID`，后续实时事件都会推给这个连接。断线重连的推荐流程：**重连 → `session.history`（带 `since`）→ 继续接收事件**。
- `limit`：0 用默认 200，负数取全量（上限 1000），超过 1000 会被截断。
- `hasMore` 是保守判断（本页条数达到 limit 即为 true），翻页以返回空列表为准。
- `question` 只有媒体时是媒体 JSON，前端需要兜底展示。
- `completed=false` 的轮次表示请求已落库但未完成（例如进程重启导致中断），可展示为「已中断」。
- 补发基于**数据库轮次**而非事件缓冲：断线期间正在生成的那一轮不补逐帧内容，重连后由后续事件或 `message_completed` 覆盖收敛。

### 5.2 多用户与鉴权

身份在握手阶段确定，整个连接生命周期内不变。

| 配置 | 身份 | 说明 |
| --- | --- | --- |
| `users` 非空 | 查表得到的 userID | token 未登记或 userID 非法 → HTTP 401 |
| 只配 `auth_token` | 固定 `web` | 共享口令，本质仍是单用户 |
| 都不配 | 固定 `web` | 仅适合内网/本机 |

三层隔离：

1. **会话键**：`ws_<userID>_<chatID>`，同名对话天然不冲突。
2. **订阅**：Hub 按 sessionID 分发，连接只收到自己订阅的会话。
3. **命令校验**：`chat.send` / `chat.cancel` / `session.history` 都检查 `sessionID` 是否为 `ws_<当前 userID>_` 前缀，否则返回 `forbidden`。

userID 限制：只允许字母、数字、短横线，**不能含下划线**。归属判断是前缀匹配，`alice` 与 `alice_2` 会让 `ws_alice_` 产生歧义。`validateUsers` 在启动时校验，非法配置直接启动失败。

迁移：改造前的会话键是 `ws_web_*`，给对应 token 映射 `web` 即可继续访问旧会话；否则旧历史在新身份下不可见。`hello` 会返回当前 `userID`，前端可用于展示或排查。

---

## 6. 生命周期与并发

- 每连接两个 goroutine：`readPump`（读 + 处理 pong）与 `writePump`（唯一写者，gorilla 不允许并发写）。
- 心跳：服务端每 `heartbeat_interval` 发控制帧 ping；浏览器自动回 pong；`read_timeout` 内未收到 pong 则判定连接失效。
- 背压（`conn/client.go: enqueue`）：
  - 队列满时，`delta` 直接丢弃（前端靠 `completed` 收敛）；
  - 其余非增量帧（start/completed/tool/error/finished/approval）入队失败则主动断开，让前端重连。
- 关闭顺序：`Stop` → 取消 ctx → 关闭 bridge 订阅 → `http.Server.Shutdown` → 广播 `system/shutdown` → 逐个关闭连接。

---

## 7. 扩展指南

### 7.1 新增非 agent 消息（通知、定时任务等）

三步，不用改协议：

1. 在 `protocol/envelope.go` 的 `Kind` 加枚举（例如已有 `KindCron`、`KindNotice`）；
2. 在 `protocol/payload.go` 加对应 Data 结构；
3. 业务侧调用 `wsServer.Broadcast`：

```go
wsServer.Broadcast(protocol.KindCron, "task_done", &CronTaskData{
    TaskID: "t-1", Name: "日报", Status: "ok",
})
```

注意 `Broadcast` 是**广播给所有连接**，按会话定向推送当前未暴露；需要定向时扩展 `conn.Hub.Dispatch`。

`wsServer` 在 `main.go` 里是局部变量，需要推送的业务模块（如 cron）应通过构造函数或 setter 注入该引用，不要另起全局单例。

### 7.2 新增上行命令

```go
wsServer.HandleCommand("session.history", func(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
    var req struct {
        SessionID string `json:"sessionID"`
        Limit     int    `json:"limit"`
    }
    if err := env.DecodeData(&req); err != nil {
        c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败"})
        return
    }
    // 查询历史后回复，类型建议与上行命令同名
    c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, "session.history").
        WithReplyTo(env.ID).
        WithData(historyData))
})
```

处理器内部要自己保证：参数校验失败调 `c.ReplyError`，成功至少回一条带 `replyTo` 的响应。

### 7.3 新增配置项

`core/config/websocket.go` 加字段 + 在 `SetDefaults` 补默认值；`server.New` 里读取并落到 `conn.Config` 或 `bridge.Options`。配置统一通过 `Options` 传入，包内不要读取全局变量。

---

## 8. 修改注意事项（踩坑点）

1. **`message_completed` 必须是全量文本**。它来自 `codex/cdm/eventHandler.go` 的 `item.AgentMessageItemDetail.Text`。若上游改成只发增量，第 6 节的丢帧策略立刻不安全，前端会永久缺字。`bridge_test.go` 有断言守着这一点。
2. **`bridge.Run` 必须保证退出时注销订阅**。总线 fanout 是单 goroutine 顺序 push 且不带超时，残留订阅无人消费时，队列写满会阻塞整个 fanout，连带企业微信一起收不到消息。当前用 `defer b.Close()` + 两层 `recover` 兜住，改动时不要删。
3. **不要在 `bridge` 里做耗时操作**。它是总线上的消费者，慢一点就会反压到所有渠道。marshal 已放在 `Hub.deliver`（每条事件一次），大 base64 媒体是唯一潜在慢点。
4. **不要复用 `channel.StreamBuffer`**。那是给"覆盖式更新同一条消息 + 有 RTT/限流"的企业微信接口用的（accumulate + 合并）。ws 走增量 + 前端追加，带宽是 O(n)，套过来会变成 O(n²) 且增加延迟。
5. **`conn` 不要 import `server`**，用 `conn.Handler` 接口；`bridge` 不要依赖 `Hub` 具体类型，用 `Dispatcher` 接口。破坏这两条会形成循环依赖。
6. **`Client.Outbound()` 仅用于测试**，生产代码不要消费，否则消息会被偷走。
7. **`session_turns` 的关联字段靠 UPDATE 写入，且不会被覆盖**。正常轮次由 `cdm` 建 turn 时写入 `SessionID/ThreadID`；`manager` 完成时只补 `Answer`，因为 gorm 的 `Updates(struct)` 跳过零值字段。真正需要额外带 `SessionID` 的是**不走 cdm 建 turn 的路径**（`BeforeAgent` 拦截、`Steer`、`turn.Start` 失败），所以 `RequestSave` 改为在插入时就带上。新增写路径同样要带，否则该轮次在历史接口里查不到；也不要把 `TurnSave` 改成 `Save`，那会整行覆盖已有字段。
8. **userID 不能含下划线**。会话键与归属判断都依赖 `ws_<userID>_` 前缀，`alice_2` 会与 `alice` 混淆。`validateUsers` 已在启动时拦截，将来新增身份来源（如 JWT）时同样要校验。

---

## 9. 测试

```bash
go test -race ./core/db/ ./core/ws/...
```

| 包 | 覆盖 |
| --- | --- |
| `db` | 按会话/线程查询、limit 与 beforeSeq 翻页、since 增量、线程列表、RequestSave 写入 sessionID |
| `protocol` | 信封往返、6 类非法输入、Kind 校验、IsDelta |
| `conn` | 按会话分发、注销清理、广播、背压（delta 丢弃 / 关键帧断开） |
| `command` | chat.send 投递与 ack、空内容/未知 agent/未知命令/非 client 域拒绝、SessionKey 一致性、历史分组/翻页/补发/订阅、跨用户越权拒绝 |
| `bridge` | 渠道过滤、完成/错误/审批/增量转换、completed 的 delta=false 断言 |
| `server` | 真实握手 + hello、agent.list、ping/pong、401 鉴权、503 连接上限、多用户 token 映射、非法 userID 启动失败 |

---

## 10. 已知限制与后续

- **补发基于数据库轮次**：`session.history(since)` 只能补已落库的轮次；断线期间正在生成的那一轮不补逐帧内容，重连后由后续事件或 `message_completed` 覆盖收敛。
- **暂无会话列表接口**：前端需自行保存 chatID / sessionID；如需「历史会话列表」，可按 `Session` 表新增 `session.list`。
- **不支持审批回传**：`approval` 事件能下发，但客户端无法回应，审批闭环待 P2。
- **媒体原样透传**：`types.Media` 直接下发，若上游填了 `base64` 会原样传输；P2 计划改为只传 URL，避免大帧拖慢 bridge。
- **无用户管理**：身份来自静态 token 映射，没有增删改接口与轮换机制，也没有 per-user 连接数/速率限制（`max_clients` 是全局的）。
- **无定向推送**：`Broadcast` 面向所有连接。
- **未做 delta 合并**：弱网下会出现文字跳跃，如确有必要，在 `conn.writePump` 侧做 per-connection coalescing（16~33ms 窗口），不要动 bridge。

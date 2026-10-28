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
  "allow_origins": ["*"],
  "max_clients": 100,
  "send_buffer": 256,
  "stream_coalesce": true,
  "stream_coalesce_ms": 50,
  "stream_coalesce_max_bytes": 4096,
  "heartbeat_interval": 30,
  "read_timeout": 60,
  "write_timeout": 10,
  "forward_all_channels": false,
  "default_agent": "codex",
  "channel": "ws",
  "file_dir": "./tmp/ws",
  "max_upload_size": 52428800,
  "file_ttl": 86400
}
```

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `enabled` | false | 是否启用 |
| `host` / `port` | 0.0.0.0 / 8765 | 监听地址 |
| `path` | `/ws` | 升级路径 |
| `allow_origins` | 空 | 为空不限制；支持 `*`、完整 Origin、主机名 |
| `max_clients` | 100 | 超过返回 503 |
| `send_buffer` | 256 | 单连接发送队列长度，满时的策略见第 6 节 |
| `stream_coalesce` | true | 是否合并增量帧（`bridge/delta_coalescer.go`），关掉即逐帧直发 |
| `stream_coalesce_ms` | 50 | 合并窗口（毫秒）：窗口内同一片段的增量拼成一帧发出 |
| `stream_coalesce_max_bytes` | 4096 | 单片段攒到多少字节就不等窗口立刻落地 |
| `heartbeat_interval` | 30 | 控制帧 ping 间隔（秒） |
| `read_timeout` | 60 | 读超时（秒），需大于心跳间隔 |
| `write_timeout` | 10 | 单次写超时（秒） |
| `forward_all_channels` | false | 是否转发其他渠道（如 wxcom）的事件 |
| `default_agent` | 空 | 前端未指定 agent 时使用 |
| `channel` | `ws` | 入站消息的渠道标识。会话键为 `<channel>_<userID>_<chatID>`，同时决定 agent 的 `send_file` 回流到哪个通道，见 3.8 |
| `file_dir` | `./tmp/ws` | 上传文件落盘目录 |
| `max_upload_size` | 50MB | 单文件字节上限 |
| `file_ttl` | 86400 | 文件保留时长（秒） |
| `page_router` | 无 | 前置界面路由（决策模型替用户打开页面），见 3.7 |

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
wscat -c "ws://127.0.0.1:8765/ws?token=alice"
# token 即当前连接的身份（默认实现），会话键会是 ws_alice_<chatID>
# 连上后先收到 hello，里面带 userID 与 chatID，之后发消息不必再传对话信息
{"v":1,"id":"c1","kind":"client","type":"chat.send","data":{"content":"你好"}}
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
├── replay/          历史回放：turn_events → protocol.Envelope（与 bridge 产出同构）
├── filestore/       文件存储：落盘、索引、归属校验、过期回收
├── command/         上行命令：一个命令一个文件
│   ├── command.go        依赖、注册表、分发、会话键（只做注册与集成）
│   ├── ping.go           ping
│   ├── chat_send.go      chat.send（含 fileID 还原）
│   ├── chat_cancel.go    chat.cancel
│   ├── agent_list.go     agent.list
│   ├── session_list.go   session.list
│   ├── session_history.go session.history（含执行过程帧）
│   ├── turn_replay.go    session.turn.replay
│   └── helper.go         命令间共用的小工具
└── server/          HTTP 升级、鉴权、Origin、生命周期、组装
    ├── server.go         服务主体
    ├── files.go          文件端点 + send_file 回流
    └── ui.go / preflow.go 界面反向 RPC 与前置路由
```

依赖方向（单向，无环）：

```
server ──> command ──> replay ──> protocol
   │         │           │
   │         └──> conn ──┘
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
| `eventID` | string | 上游实体 ID（见下），同一实体的多次更新共享 |
| `replyTo` | string | 响应关联的上行 `id` |
| `seq` | int64 | 服务端全局单调序号，用于排序与排查 |
| `chunk` | int | 流内序号，对齐 `Event.ChunkIndex` |
| `ts` | int64 | 毫秒时间戳 |
| `data` | object | 载荷，按 `kind + type` 解释 |

`eventID` 是给前端做**归并**用的：同一条消息的 `message_start` / `message_delta` / `message_completed`，或者一个工具的 `tool_start` / `tool_completed`，共享同一个 `eventID`（即 codex 的 item ID）。前端按它挂载、覆盖同一条记录，就不必依赖「必须先收到 start」这种顺序假设——重连补发、乱序、历史回放都能落在正确的位置上。

例外：审批帧不填 `eventID`，标识在 `data.approval.id`（那是回传审批时要用的请求 ID）；`finished` 等轮次级事件也没有实体，保持为空。

`kind` 是扩展位：前端遇到未知 `kind` 应**忽略而不是报错**，这样服务端新增消息域不会打断老页面。

版本策略：新增 `kind` / `type` 属于向后兼容，不升 `v`；字段语义变更或删除才升版本。

### 3.2 下行 · system

| type | 时机 | data |
| --- | --- | --- |
| `hello` | 连接建立后第一帧 | `clientID`、`userID`、`chatID`、`protocol`、`agents`、`defaultAgent`、`heartbeat` |
| `ack` | 上行请求受理 | `ok`、`sessionID`、`chatID`、`reqID` |
| `pong` | 收到 `ping` | `clientTS`、`serverTS` |
| `error` | 任何失败 | `code`、`message`、`detail` |
| `agent.list` | 收到同名上行命令 | `agents`、`defaultAgent` |
| `session.list` | 收到同名上行命令 | `items[]`、`hasMore`、下一页游标（见 5.1） |
| `session.history` | 收到同名上行命令 | `sessionID`、`threads[]`、（可选）`frames[]`（见 5.2） |
| `session.turn.replay` | 收到同名上行命令 | `sessionID`、`turnID`、`frames[]`（见 5.2） |
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

data 结构见 `protocol.AgentData`：`event`、`agent`、`role`、`delta`、`content`、`reasoning`、`toolCalls`、`media`、`approval`、`error`、`meta`、`truncated`。

`truncated` 只在历史回放里可能出现，标记该条内容因为体积上限被截断；实时事件不会置位。

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

这条约定不是可选的：服务端在网络拥塞时**会丢弃 delta**（丢的是积压里最旧的那条，见 §6 背压），依靠 `message_completed` 的全量内容让界面收敛。如果前端把 `completed` 当增量拼接，丢帧后内容就永远修不回来了。

`message_delta` / `reasoning_delta` 会被合并（见 §6 增量帧合并）：**一帧的内容可能是好几个 token 拼起来的一段文本，但 `delta` 仍然是 `true`**，前端继续按追加处理，不需要区分是不是合并帧。合并只发生在同一片段（同 `eventID`）内部，不会把思考与正文混进一帧。

### 3.4 上行 · client

`kind` 必须是 `client`，否则服务端返回 `unsupported`。

| type | data | 说明 |
| --- | --- | --- |
| `chat.send` | `sessionID?`、`chatID?`、`agent?`、`content`、`media`、`fileIDs?`、`metadata` | 发送提问 |
| `chat.cancel` | `sessionID?`、`chatID?` | 中断当前会话 |
| `ping` | `ts` | 应用层心跳，测 RTT |
| `agent.list` | — | 拉取可用 agent |
| `session.list` | `limit?`、`beforeLastUpdate?`、`beforeThreadID?` | 拉取对话列表，见 5.1 |
| `session.history` | `sessionID?`、`chatID?`、`threadID?`、`beforeSeq?`、`since?`、`limit?`、`includeFrames?` | 拉取历史消息，见 5.2 |
| `session.turn.replay` | `sessionID?`、`chatID?`、`turnID`、`afterSeq?`、`limit?` | 回放单轮完整过程，见 5.2 |

`chat.send` 可以不传 `chatID` 与 `sessionID`：服务端会沿用连接当前的对话（即 `hello` 下发的 `chatID`），前端只需持有登录 token。传 `chatID` 表示切换到该对话，连接上下文随之更新；`ack` 会返回本次使用的 `chatID` 与 `sessionID`（`ws_<userID>_<chatID>`）。

### 3.5 错误码

| code | 场景 |
| --- | --- |
| `bad_request` | JSON 非法、缺字段、content / media / fileIDs 全为空、fileID 不可用 |
| `unauthorized` | 实际以 HTTP 401 返回，不进协议层 |
| `forbidden` | 访问不属于当前用户 `sessionID` 的会话，或他人的文件 |
| `unsupported` | 未知 `kind` 或 `type` |
| `agent_not_found` | 指定的 agent 不存在 |
| `too_many_clients` | 以 HTTP 503 返回 |
| `internal` | 投递失败等服务端异常 |
| `busy` | 预留 |

文件端点走 HTTP 状态码，响应体是 `{"code":"...","message":"..."}`：

| code | HTTP | 场景 |
| --- | --- | --- |
| `file_required` | 400 | 上传没带 `file` 字段 |
| `file_too_large` | 413 | 超过 `max_upload_size` |
| `file_not_found` | 404 | fileID 不存在、已过期或文件已被移走 |
| `forbidden` | 403 | 他人的文件 |
| `upload_failed` | 500 | 落盘失败 |

### 3.6 下行 · ui（反向请求）

上面几条都是「浏览器问、服务端答」。`ui` 域是反过来的：**服务端问、浏览器答**，
用于 agent 操作 Web 界面（工具见 `core/ui_tools.go`，实现见 `server/ui.go`）。

| 方向 | kind / type | 说明 |
| --- | --- | --- |
| 下行 | `ui` / `ui.request` | `data.op` 三选一：`observe` 取快照 / `act` 执行命令（带 `cmd`、`params`、`rev`）/ `navigate` 打开或切换页面（带 `view`，或只给 `query` 搜菜单） |
| 上行 | `client` / `ui.response` | `replyTo` 指回请求 ID；`data` 为 `{ok, error?, rev?, snapshot?, result?}` |

约定：

- `navigate` 不带 `rev`：打开另一个页面跟当前页面的版本无关。`rev` 按页面（view）分域维护。
- `navigate` 要等目标页 mount 并就绪（页面自报 ready）后才回执，回执里直接是新页面快照；
  不等就回执的话，agent 紧接着 observe 会拿到空列表，然后开始自我怀疑。
- 回执必须把请求帧的 `sessionID` 原样带回，服务端据此校验会话一致性（避免多标签页串号）。
- 请求按会话投递，页面必须先订阅该会话（发过 `chat.send` / `session.history` / `session.list`）。
  **没有订阅者时立即失败**（`ErrNoUIHost`），不会白等到超时。
- 服务端同步等待回执，默认超时 10s（`Options.UIRequestTimeout`）。超时、无宿主、连接断开
  都会作为工具失败告知 agent，而不是静默返回空快照。
- 载荷里的 `snapshot` / `result` / `params` 对服务端是**透传**的 `json.RawMessage`：
  服务端不理解业务结构，页面结构的变化只影响前端与工具描述。
- 同一会话有多个页面连接时，请求会投给全部连接，先回执者胜出；要精确到某个标签页
  需要引入 `targetID`（当前未实现）。

### 3.7 前置界面路由（决策模型）

「该打开哪个页面」「要不要先点开某个弹窗」这类判断是分类问题，不必花 agent 的 token：
`chat.send` 收到消息后先交给 [preflow](/Users/wangyingbin/Developer/go/src/bgAgent/buibuiCodex/core/ws/preflow/preflow.go)，
由百炼决策模型（`decision` 包）判完并把界面就位，消息再进 agent。协议与前端一行都不用改——
用的还是 3.6 那三条 `ui.request`。

```
chat.send ──> ack（先应答受理）──> 消息按会话入队，读循环立刻返回
                                    │
                  （每个会话一个 worker，串行）
                  ui.request observe ──> 决策「打开哪个页面」
                       └─> ui.request navigate ──> 决策「点开哪个入口」
                               └─> ui.request act ──> PublishInput ──> agent
```

`websocket.page_router` 配置（凭证留空回落到环境变量 `DASHSCOPE_API_KEY` / `DASHSCOPE_BASE_URL`）：

```json
"websocket": {
  "page_router": {
    "enabled": true,
    "api_key": "",
    "base_url": "https://<控制台给的完整地址>/compatible-mode/v1/systemone",
    "model": "decision-model-preview",
    "timeout": 8,
    "deep_action": true,
    "annotate": true,
    "max_pages": 60,
    "max_actions": 40
  }
}
```

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `enabled` | false | 关闭时 `chat.send` 与改造前完全一致 |
| `base_url` | 环境变量 | 决策模型接口地址，控制台给什么就填什么 |
| `timeout` | 8 | 整个前置流程的总预算（秒），超时后消息照常进 agent |
| `deep_action` | true | 打开页面后是否继续判断子入口 |
| `annotate` | true | 是否把已做完的界面动作作为一行说明随消息交给 agent |
| `max_pages` / `max_actions` | 60 / 40 | 送给模型的候选数量上限 |

要点：

- **不能在读循环里等回执**。`ui.response` 恰好要从同一条连接的读循环进来，同步等是自己等自己，
  必然超时。所以 `chat.send` 先回 `ack`，把消息交给会话队列，由 worker 做完界面动作再
  `PublishInput`；ack 里的 `reqID` 就是最终那条消息的 ID。
- **队列而不是每条消息一个 goroutine**。裸 goroutine 只解死锁，不解顺序与并发：用户连发两条时
  后一条可能先到 agent，两个 worker 还会交叉操作同一个页面。`command/preflow_queue.go` 保证
  一个会话最多一个 worker、严格先进先出；积压超过 32 条后新消息不再做界面判断，只排队照常投递。
  服务 `Stop` 时先 `commands.Shutdown()`：已排队的消息投完，只是不再发起新的界面动作。
- **不设置信度门槛**：选项里带了 `current` / `none` / `other` 三个空白项，模型选哪个就照做，
  不需要跳转时它自己会选空白项。52 个菜单里 top1 只有 0.36 是常态，用绝对置信度当闸门会把
  正确判断一起挡掉（本项目踩过：现象是「在主界面说要看预订单却没反应」）。
- **fail-open**：页面没开、决策失败、navigate 被拒、总预算超时——都只是不做什么，
  消息照常进 agent，不会挡住聊天。
- **声明了参数的入口不代办**：决策模型只出分类与概率，不生成文本，填不出 `orderId` 这类业务参数。
  这种入口会进 `pending`，说明文字里点名让 agent 去点，agent 该拿的参数照样能拿。
- **快照解析是宽松的**：`navigation`/`menus`/`pages`、`actions`/`commands`/`entries`、
  `title`/`name`/`label` 都能取到；解析不出可用信息就退化成「不做前置路由」。
- **候选菜单只要业务页面，不要分类节点**。分类节点（点开只是展开分组）带着 view，
  选中它会把用户带到一个分组页——实测「我要查看预订单」被开到「预订单」分类而不是「预订单管理」。
  `navigation[]` 里满足任一条即判为分类并剔除，日志用 `skippedGroups` 记账：
  1）显式标了不可选：`selectable` / `clickable` / `navigable` / `available` / `isLeaf` / `leaf` / `enabled` 为 false；
  2）`type` / `kind` / `nodeType` / `menuType` 命中 group、category、directory、submenu、folder、header 等；
  3）`children` / `items` / `submenus` / `options` 非空。
  **首选还是前端只下发可打开的叶子页面**：只有分类节点自己也带 view、又不带任何标记时，
  后端没有线索可判。叶子上的 `path` / `route` / `group` / `description` 会拼进选项描述，
  「预订单」和「预订单管理」这类相近标题就靠它们区分。
- **每一步都有日志**，「页面没反应」直接看这三条：

```
INFO preflow 界面请求   session=ws_u1_c1 op=navigate view=OrderList outcome=ok snapshotBytes=94 elapsed=0s
INFO preflow 决策       kind=page answer=p1 target=OrderList confidence=0.94 options=3
INFO preflow 界面前置路由 session=... message=帮我建个工单 applied=[...] decisions=[...] note=... reason=...
```

`reason` 就是没动界面的原因（`取页面快照失败：no ui host connected for session` 表示这条会话没有页面连接）。

### 3.8 文件收发（HTTP）

文件字节不走 WebSocket：单帧上限 1MB（`protocol.MaxMessageSize`），base64 再膨胀 4/3，
稍大的附件就撞墙；而且帧内传输没法断点续传，也没法用浏览器自身的下载与预览能力。
所以**字节走 HTTP，ws 只传 fileID**。

端点挂在升级路径下（`path=/ws` 时是 `/ws/files`），鉴权与 ws 共用同一个 token：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/ws/files` | `multipart/form-data`，字段名 `file`（可多个）。响应 `{"items":[FileInfo]}` |
| `PUT` | `/ws/files?name=xx.txt` | 请求体即文件本身，适合大文件与上传进度 |
| `GET` | `/ws/files/{id}` | 取文件内容。默认内联，`?download=1` 另存，`?meta=1` 只返回元信息 |
| `GET` | `/ws/files` | 列出当前用户仍在有效期内的文件 |
| `DELETE` | `/ws/files/{id}` | 删除自己的文件 |

```bash
# 上传
curl -F "file=@报表.xlsx" "http://127.0.0.1:8765/ws/files?token=alice"
# {"items":[{"id":"a1b2...","name":"报表.xlsx","size":20480,"url":"/ws/files/a1b2...","expiresAt":...}]}

# 让 agent 处理这个文件
# {"v":1,"id":"c2","kind":"client","type":"chat.send","data":{"content":"看看这个表格","fileIDs":["a1b2..."]}}

# 前端预览（图片/文本/PDF 可直接渲染）
# <img src="/ws/files/a1b2...?token=alice">
```

**上行链路**：

```
POST /ws/files        落盘到 file_dir，返回 fileID
  ↓
chat.send{fileIDs}    命令层取回本地路径，拼成 Media{URL: /abs/path, Metadata: {filename}}
  ↓
core.Manager          mapMediaToContainer 把文件移到该 agent 的 <WorkSpace>/tmp，
                      并把 URL 换成容器路径（/tmp/xxx）交给模型
```

路径转换刻意放在 `core.Manager` 而不是命令层：`chat.send` 执行的时刻还不知道
最终由哪个 agent 处理（前端可能指定，也可能落到默认 agent），而每个 agent 的
工作区映射是不同的。**不要在这条链路上提前把路径转成容器路径。**

注意文件是被**移动**而不是复制，同一个 fileID 只能用一次；移动后原路径消失，
`filestore.Get` 会做一次 `stat` 校验并把失效索引清掉。

**下行链路**：

```
agent 调 send_file 工具
  ↓ 工具层已做 ToHost（容器路径 → 宿主机路径）
bus.SyncMessage(send_file)  按 event.Channel 路由
  ↓
server.handleSyncRequest    登记文件 → 推一帧 kind=agent / type=tool_completed
  ↓                          data.media[0].url = /ws/files/<id>
浏览器收到附件卡片，可下载或内联预览
```

这条路由的键就是渠道标识 `channel`。**它必须是 `ws`**：与 wxcom 共用标识
（例如 `wxcom:work`）时，`send_file` 会被企业微信通道接管，文件真的发到
企业微信会话里，web 端收不到任何东西。改这个值等于关掉文件回传，
同时会话键前缀从 `ws_<userID>_<chatID>` 变成 `<channel>_<userID>_<chatID>`，
历史会话会对不上——确有需要时再改。

安全约定：

- 下载端点默认内联，但 HTML / SVG / XML 一律降级为 `attachment`，
  并统一带 `X-Content-Type-Options: nosniff`。可执行的类型内联在同源下
  等于把 XSS 开口子交给上传者。
- fileID 是 16 字节随机串，按连接身份校验归属：拿到别人的 fileID 也读不到内容。
- 文件按 `file_ttl` 过期，服务每小时回收一次；进程重启后索引丢失，
  磁盘目录靠 mtime + ttl 兜底清理。

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

### 5.1 对话列表

打开应用时先要「我有哪些对话」。`session.list` 返回当前用户的对话，**一条对应一个 thread**：

```json
{"kind":"client","type":"session.list","data":{"limit":50}}
```

响应：

```json
{
  "items": [
    {
      "threadID": "th-2", "sessionID": "ws_web_c2", "agent": "codex",
      "title": "分析昨天良率", "preview": "生产良率为 98.2%，主要瓶颈在二号线…",
      "turns": 6, "completed": true,
      "createdAt": 1761432000000, "lastUpdate": 1761435600000
    }
  ],
  "hasMore": true,
  "nextBeforeLastUpdate": 1761435600000,
  "nextBeforeThreadID": "th-2"
}
```

为什么按 thread 而不是 session 分组：

- 一个 **thread** 是一次完整对话，其下挂多轮 turn，`turns` 就是轮次数
- 一个 **session**（`ws_<userID>_<chatID>`）下可以有多个 thread——换过 agent 就会新建
- 所以同一会话里换过 agent，列表里会显示成两条对话，各自延续各自的上下文

字段约定：

- `threadID` + `sessionID` 拿去就能用：`session.history` 带 `threadID`，`chat.send` 带 `sessionID`
- `title` 取首轮提问（按 40 字符截断）；该轮只有媒体时为空，前端用「图片消息」之类的兜底文案
- `preview` 取末轮回答（80 字符）
- `turns` 是轮次数量，不需要再拉详情就能显示「共 N 轮」
- `completed=false` 表示末轮没有正常结束（被中断或仍在跑）
- 只返回当前用户自己的对话：归属过滤在 SQL 里通过 `sessions` 表关联完成，不能靠 `session_id` 前缀匹配——会话键里的下划线在 `LIKE` 里是通配符，`ws_alice_` 会误命中 `ws-alice_`

翻页：把 `nextBeforeLastUpdate` + `nextBeforeThreadID` 原样回传即可。用复合游标而非单一时间戳，是因为多个对话的 `last_update` 可能落在同一毫秒，只比时间会漏项。

`limit`：0 用默认 50，负数取全量（上限 200）。`hasMore` 是保守判断（本页条数触顶即为 true），以返回空列表为准。

### 5.2 历史消息

数据层级是 `session → thread → turn`：

- `session`：一个会话（`ws_<userID>_<chatID>`）
- `thread`：会话下某个 agent 的线程，换过 agent 就会新建，所以一个会话可能有多个 thread
- `turn`：一次问答，`Question` / `Answer` 落库，`Answer` 是完成时的全量文本

上行 `session.history` 的用法：

| 场景 | 参数 | 说明 |
| --- | --- | --- |
| 打开历史对话 | `{sessionID}` | 返回该会话下所有线程与轮次 |
| 带执行过程 | `{sessionID, includeFrames: true}` | 额外返回 `frames[]`，即思考与工具调用的完整流程 |
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

#### 执行过程（frames）

`question` / `answer` 只有问答本身，看不出中间发生了什么。带上 `includeFrames: true` 后，响应多出一个 `frames` 数组：

```json
{
  "sessionID": "ws_web_c1",
  "threads": [ ... ],
  "frames": [
    {"v":1,"kind":"agent","type":"reasoning_completed","sessionID":"ws_web_c1","reqID":"r1","eventID":"i1","seq":31,"ts":1761432001200,
     "data":{"event":"reasoning_completed","agent":"codex","role":"assistant","reasoning":"先确认表结构，再算良率"}},
    {"v":1,"kind":"agent","type":"tool_completed","sessionID":"ws_web_c1","reqID":"r1","eventID":"i2","seq":34,"ts":1761432004400,
     "data":{"event":"tool_completed","agent":"codex","role":"tool","delta":false,
             "toolCalls":[{"id":"i2","type":"command","name":"command","params":"duckdb -c 'select ...'",
                           "result":"98.2|1200","extra":{"cwd":"/work","status":"completed"}}]}},
    {"v":1,"kind":"agent","type":"message_completed","sessionID":"ws_web_c1","reqID":"r1","eventID":"i3","seq":41,"ts":1761432012000,
     "data":{"event":"message_completed","agent":"codex","role":"assistant","content":"生产良率为 98.2%"}}
  ],
  "framesTruncated": false
}
```

关键点：**`frames` 里每一项就是完整的信封，和实时推送长得一模一样**。前端可以直接把它们按 `seq` 顺序喂给同一个消息处理器，不需要为历史单独写一套渲染逻辑——`kind`、`type`、`delta`、`data` 的语义全部通用。

约定：

- `frames` 按时间升序，与实时流的时序一致；每帧的 `reqID` 指向所属轮次，前端据此把过程挂到对应问答下方。
- 每帧的 `eventID` 与实时流用的是同一套 ID（同一条消息、同一个工具，历史与实时取值相同）。所以先拉历史、再接实时，同一个实体不会重复建气泡；这也是"历史与实时共用一条渲染管线"能成立的前提。
- 帧一律 `delta=false`，是**全量快照**。开始帧（`tool_start`）与完成帧（`tool_completed`）共享同一个工具 ID，按 ID 覆盖即可得到最终状态；只有开始帧没有完成帧，说明该步被打断。
- 只回放关键帧（`item` 生命周期、审批、错误），**不重放逐 token 的 delta**。delta 在 `turn_events` 里仍然完整保留，需要时按 `method` 单独取。
- 出于体积考虑，历史里的 `frames` 是**有上限的**：每轮最多 200 帧，单个文本（思考、命令输出、工具结果）截断到 4KB，被截断的帧带 `truncated: true`，整体被截时 `framesTruncated: true`。想要完整内容用下面单轮接口。
- `framesTruncated` 为 true 只代表「不是 100% 完整」，不影响已返回帧的正确性。

#### 单轮完整回放

`includeFrames` 适合列表页一次性铺开。用户点开某一轮细看时，用 `session.turn.replay` 拉这一轮的完整过程（不限帧数、不截断内容）：

```json
{"kind":"client","type":"session.turn.replay","data":{"sessionID":"ws_web_c1","turnID":"tu-1"}}
```

响应 `data`：

| 字段 | 说明 |
| --- | --- |
| `sessionID` / `turnID` | 回显，便于并发请求时对号 |
| `frames` | 与上面同构的 agent 信封，按 `seq` 升序，完整不截断 |
| `hasMore` | 还有更晚的帧 |
| `nextAfterSeq` | 下一页游标，直接作为下次的 `afterSeq` |
| `truncated` | 是否有内容被截断（详情接口正常为 false） |

服务端会校验 `turnID` 是否属于该会话，跨会话访问返回 `forbidden`。

### 5.3 身份与鉴权

身份在握手阶段确定，整个连接生命周期内不变。解析逻辑是一个可替换的函数：

```go
// core/ws/server/server.go
type Authenticator func(token string) (userID string, chatID string, ok bool)
```

返回值语义：

- `ok=false` → 握手返回 401
- `chatID` 为空 → 服务端生成 uuid，通过 `hello` 下发
- `chatID` 非空 → 直接作为当前对话，适合「一个 token 固定一个对话」或从库里恢复最近会话

默认实现 `DefaultAuthenticator`：token 即 userID，`chatID` 留空，只校验 token 格式（字母、数字、短横线，长度 ≤ 64）。

接入真实鉴权时通过 `Options.Authenticator` 注入：

```go
wsServer := ws.New(ws.Options{
    // ...
    Authenticator: func(token string) (string, string, bool) {
        user, err := auth.Verify(token) // 查库 / 校验 JWT
        if err != nil {
            return "", "", false
        }
        return user.ID, user.CurrentChatID, true
    },
})
```

前端最简用法：只带 token，`hello` 里拿到 `chatID` 后直接 `chat.send`，不需要自己生成或保存任何对话标识；要切换对话时再显式传 `chatID`。

token 传递：`?token=xxx` 或 `Authorization: Bearer xxx`，Header 优先。

三层隔离：

1. **会话键**：`ws_<userID>_<chatID>`，同名对话天然不冲突。
2. **订阅**：Hub 按 sessionID 分发，连接只收到自己订阅的会话。
3. **命令校验**：`chat.send` / `chat.cancel` / `session.history` 检查 `sessionID` 是否为 `ws_<当前 userID>_` 前缀，否则返回 `forbidden`。

userID 限制：只允许字母、数字、短横线，**不能含下划线**。归属判断是前缀匹配，`alice` 与 `alice_2` 会让 `ws_alice_` 产生歧义。自定义 Authenticator 也必须返回合法值，否则该连接的会话命令都会因归属校验失败而被拒绝。

`hello` 会返回当前 `userID`，前端可用于展示或排查。

---

## 6. 生命周期与并发

- 每连接两个 goroutine：`readPump`（读 + 处理 pong）与 `writePump`（唯一写者，gorilla 不允许并发写）。
- 心跳：服务端每 `heartbeat_interval` 发控制帧 ping；浏览器自动回 pong；`read_timeout` 内未收到 pong 则判定连接失效。
- 背压（`conn/queue.go` + `conn/client.go: enqueue`）：
  - 队列满时**丢最旧的增量帧**（`delta`），不是丢新帧——早期丢新帧的写法在浏览器稍一卡顿时会把一整轮 delta 丢光，
    界面只能等 `message_completed` 的全量文本才一次性刷出来，现象就是「没有流式」；
  - 关键帧（start/completed/tool/error/finished/approval/系统帧）永不主动丢弃：队列满时先挤掉一条增量帧给它让位；
  - 只有队列里全是关键帧、腾不出位置时才断开连接，让前端重连 + 历史补发；
  - 每次丢弃都会累计计数并限流打一条 warn 日志（`dropped` / `queued` / `capacity`）——
    「界面不流式」时先看这条日志是否在涨。
- 增量帧合并（`bridge/delta_coalescer.go`，配置 `stream_coalesce*`）：
  - 目的在**降帧数**：上游逐 token 发 delta，一轮回答动辄上百帧，而前端每帧都要重渲染整段 Markdown；
    渲染慢过帧间隔，浏览器就不再从 socket 读，上面的队列填满，于是开始丢 delta，
    界面退化成「等 `message_completed` 才一次性刷出来」。帧数压一个量级，队列基本不再填满；
  - 合并窗口默认 50ms，按**片段**分组（sessionID + reqID + eventID + 类型），
    攒够 `stream_coalesce_max_bytes` 就不等窗口立刻落地；
  - 合并的是**增量**不是全文：一帧的内容仍是「要追加的那一段」，带宽保持 O(n)；
  - 关键帧（start / completed / tool / approval / error / finished）不参与合并，
    且发出前会先把该会话窗口里的积压落地，保证「增量在前、关键帧在后」；
    `*_completed` 之后顺手释掉该片段的状态；
  - 带 `toolCalls` / `media` / `approval` / `extra` 的 delta 帧原样直发，不拼进文本帧；
  - 排查：`stream_coalesce: false` 回到逐帧直发的旧行为；没 `Start` 起窗口时同样直发，
    所以单独调 `forward` 的测试不会看到帧被窗口吞掉。
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

**一个命令一个文件**。新建 `command/<命令>.go`，写一个 `func (r *Registry) xxx(ctx, c, env)`，再到 `command.go` 的 `NewRegistry` 里加一行注册即可——`command.go` 只放依赖、注册表与分发，不放具体命令。

```go
// command/session_pin.go
// 本文件实现 session.pin 命令：置顶某个对话。

package command

// sessionPin 置顶/取消置顶一个对话
func (r *Registry) sessionPin(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
    var data protocol.SessionPinData
    if err := env.DecodeData(&data); err != nil {
        c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败"})
        return
    }
    if !r.ownsSession(c.UserID(), data.SessionID) {
        c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeForbidden, Message: "无权访问该会话"})
        return
    }
    // 处理后回复，类型建议与上行命令同名
    c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypeSessionPin).
        WithReplyTo(env.ID).
        WithData(result))
}
```

```go
// command.go
r.Register(protocol.TypeSessionPin, r.sessionPin)
```

处理器内部要自己保证：参数校验失败调 `c.ReplyError`，成功至少回一条带 `replyTo` 的响应。涉及会话归属的一律先过 `r.ownsSession`。

### 7.3 新增配置项

`core/config/websocket.go` 加字段 + 在 `SetDefaults` 补默认值；`server.New` 里读取并落到 `conn.Config` 或 `bridge.Options`。配置统一通过 `Options` 传入，包内不要读取全局变量。

---

## 8. 修改注意事项（踩坑点）

1. **`message_completed` 必须是全量文本**。它来自 `codex/cdm/eventHandler.go` 的 `item.AgentMessageItemDetail.Text`。若上游改成只发增量，第 6 节的丢帧策略立刻不安全，前端会永久缺字。`bridge_test.go` 有断言守着这一点。
2. **`bridge.Run` 必须保证退出时注销订阅**。总线 fanout 是单 goroutine 顺序 push 且不带超时，残留订阅无人消费时，队列写满会阻塞整个 fanout，连带企业微信一起收不到消息。当前用 `defer b.Close()` + 两层 `recover` 兜住，改动时不要删。
3. **不要在 `bridge` 里做耗时操作**。它是总线上的消费者，慢一点就会反压到所有渠道。marshal 已放在 `Hub.deliver`（每条事件一次），大 base64 媒体是唯一潜在慢点。增量合并的出口 `bridge.send` 也在同一条链上（`deltaCoalescer` 持锁调用 emit），因此 `emit` 必须保持「入队即返回」，不要往里塞阻塞操作或 IO。
4. **不要复用 `channel.StreamBuffer`**，ws 的合并在 `bridge/delta_coalescer.go`。企业微信那份是「按 reqID 分组 + 累积全文覆盖同一条消息」：搬过来会把一轮里交错的多个 `eventID` 片段混成一个，而且每帧重发全文，带宽从 O(n) 变成 O(n²)，长回答下反而更贵。ws 这份的三条差别：按片段分组、**窗口内只拼接增量**、关键帧先落地积压再直发。
5. **`conn` 不要 import `server`**，用 `conn.Handler` 接口；`bridge` 不要依赖 `Hub` 具体类型，用 `Dispatcher` 接口。破坏这两条会形成循环依赖。
6. **`Client.Outbound()` 只是旁路副本，仅用于测试与调试**：它不参与投递，消费它不会偷走消息，
   但也不要拿它当业务通道；真实发送一律走 `outQueue` + `writePump`。
7. **`session_turns` 的关联字段靠 UPDATE 写入，且不会被覆盖**。正常轮次由 `cdm` 建 turn 时写入 `SessionID/ThreadID`；`manager` 完成时只补 `Answer`，因为 gorm 的 `Updates(struct)` 跳过零值字段。真正需要额外带 `SessionID` 的是**不走 cdm 建 turn 的路径**（`BeforeAgent` 拦截、`Steer`、`turn.Start` 失败），所以 `RequestSave` 改为在插入时就带上。新增写路径同样要带，否则该轮次在历史接口里查不到；也不要把 `TurnSave` 改成 `Save`，那会整行覆盖已有字段。
8. **userID 不能含下划线**。会话键与归属判断依赖 `ws_<userID>_` 前缀，`alice_2` 会与 `alice` 混淆。默认 Authenticator 会直接拒绝这类 token；自定义 Authenticator 也必须保证返回值合法。
9. **回放产物必须是 agent 信封，不能另起结构**。`replay.Build` 输出的是 `kind=agent` 的 `protocol.Envelope`，和 `bridge.convert` 逐字段对齐，前端才能共用一条渲染管线。新增事件映射时不要自定义新 type，优先复用 `types.EventType`；上游新增 item 类型时，在 `replay.itemStep` 的 switch 里补分支，否则会走 `default` 降级成原始 JSON（信息不丢但可读性差）。
10. **`replay` 与 `bridge` 的字段口径要一致**。两边都在填 `AgentData`，任何一边改了 `role` / `delta` / `toolCalls` 的语义，另一边都得同步，否则历史与实时渲染出来会长得不一样。这也是 `replay` 单独成包而不是塞进 `command` 的原因：它和 `bridge` 是一对镜像实现。
11. **新命令放独立文件，别塞回 `command.go`**。那个文件只做注册集成（接口、Deps、Registry、分发、会话键）。命令一旦混进去，找某条命令的行为就得翻整份文件；现在一个命令一个文件，出问题直接打开同名文件即可。
12. **凡是要等浏览器回执的逻辑，都不能在 `conn.readPump` 上同步等**。`ui.response` 就是从这条读循环进来的，
    在 `chat.send` 里直接 `Request` 会稳定超时（本项目实现时踩过：现象是「页面没反应」，日志只有
    `ui request timeout after 2s`）。正确做法是 `routeQueue.submit` 排队，见 3.7 与 `command/preflow_queue.go`。
    同理，agent 的界面工具走的是另一条链路（`core.UIBridge` → `Server.Request`），不在读循环上，才允许同步等。
13. **`channel` 不要随便改**。它同时是会话键前缀和 `bus.SyncMessage` 的路由键（见 3.8）。
    改成 `wxcom:work` 之类会立刻失去两点：`send_file` 的文件被对应通道接管、web 端收不到；
    以及历史会话因为前缀不一致而查不出来。要改就得同时迁移 `sessions` 里的旧键。
14. **文件路径不要在命令层转成容器路径**。`chat.send` 执行时还不知道最终哪个 agent 处理，
    各 agent 的工作区映射也不同。统一交给 `core.Manager.mapMediaToContainer`（投递前调用），
    它会移动文件并换成 `/tmp/...`。提前转换会得到一个容器里不存在、宿主机上也找不到的路径。
15. **上传文件的落盘名带 fileID 前缀**。附件随后被移到 agent 工作区的 `tmp`，那里按文件名铺平，
    不带前缀会在同名文件之间互相覆盖（用户连着传两个「报告.xlsx」就中招）。展示名用 `Item.Name`，别用路径末段。

---

## 9. 测试

```bash
go test -race ./core/db/ ./core/ws/...
```

| 包 | 覆盖 |
| --- | --- |
| `db` | 按会话/线程查询、limit 与 beforeSeq 翻页、since 增量、线程列表、RequestSave 写入 sessionID、事件按轮次/白名单/游标查询、线程按账号分页（含复合游标）、首末轮摘要 |
| `protocol` | 信封往返、6 类非法输入、Kind 校验、IsDelta |
| `conn` | 按会话分发、注销清理、广播、背压（满队列丢最旧增量帧、关键帧优先、全关键帧才断开）、发送队列顺序/丢弃计数/容量复用 |
| `command` | chat.send 投递与 ack、空内容/未知 agent/未知命令/非 client 域拒绝、SessionKey 一致性、对话列表分页与截断、历史分组/翻页/补发/订阅、历史帧还原、单轮回放、跨用户越权拒绝 |
| `command` · `preflow` | 会话队列：同会话串行且保序、积压超上限跳过界面判断仍投递、退出时投完积压、不同会话互不阻塞 |
| `preflow` | 快照→决策→navigate→决策→act 全链路、选空白项与对不上菜单都不动界面、navigate 失败不再点入口、需参数入口转 agent、页面不可用/超时/未启用时静默跳过、选项表重名与容错匹配、分类节点剔除（标记/类型/children 三种线索、全分类时不问模型）、快照宽松解析 |
| `server` · `preflow` | 端到端：chat.send 先 ack，浏览器依次收到 observe/navigate/act，进 agent 的消息带上「界面已就绪」；模型选空白项与未配置时消息原样投递 |
| `bridge` | 渠道过滤、完成/错误/审批/增量转换、completed 的 delta=false 断言 |
| `bridge` · `delta_coalescer` | 窗口内拼成一条增量、关键帧前先落地积压且顺序正确、多片段互不混用、累积全文切成真正增量与重复帧丢弃、攒够字节立刻落地、带媒体的帧直发、未 Start / 关掉时逐帧直发、定时器到点只发一次、Close 落地积压、不改写上游共享事件指针 |
| `replay` | 事件→agent 帧的完整映射、delta 与用量事件被跳过、思考拼接、命令/搜索/工具解析、未知类型降级、截断与 UTF-8 边界、审批与失败帧、帧数与 seq 回填 |
| `server` | 真实握手 + hello、agent.list、ping/pong、401 鉴权、503 连接上限、token 即身份、自定义 Authenticator、非法 userID 拒绝 |
| `filestore` | 落盘与取回、恰好等于上限、超限清理半成品、跨平台文件名清洗与超长截断、归属校验、Adopt 不动原文件、文件被移走后视为不存在、过期回收、按用户列表 |
| `server` · 文件 | multipart / PUT 上传、超限 413、缺字段 400、下载内容与 Content-Disposition、`download=1` 强制另存、`meta=1`、他人文件 403、无 token 401、删除 204、chat.send 的 fileIDs 还原为本地绝对路径、他人 fileID 拒绝、send_file 推帧并能从下载端点取回、同步处理器注册幂等 |
| `server` · `core` | 界面操作反向 RPC：`ui.request` 往返、无宿主立即失败、会话不一致拒绝、端到端（工具 → 桥 → ws → 浏览器 → 回执 → 工具文本）、工具入参校验与失败文本化 |

---

## 10. 已知限制与后续

- **界面操作没有 targetID**：同一会话多标签页时请求投给全部连接、先回执者胜出，无法指定操作哪一个标签页。
- **前置界面路由是尽力而为**：它只在 `chat.send` 且带文字内容时触发，斜杠命令与纯媒体消息直接进 agent；
  打开页面用的 view 取自快照的 `navigation`，页面入参（`navigate.params`）不代填；
  候选只取可打开的叶子页面，分类节点靠前端标注（`selectable:false` / `type:"group"` / 带 `children`）
  才能可靠剔除——两者都没有时后端无从判断；
  需要业务参数的入口只报告不代办。开启后 `ack` 早于 `PublishInput`，历史里的 question 会多出
  一行「[界面已就绪] …」说明（`annotate: false` 可关掉）。
- **界面快照的 rev 只是计数器**：服务端不关心，前端目前按「一次成功操作推进一格」实现，用户自己翻页/改筛选不会推进它；真实实现应由页面状态派生，否则并发操作拦不住。
- **界面工具只有 observe / act**：没有 select（给 GridTableApi 加按业务 ID 选中行后才能做）、没有审批闭环（有副作用的命令目前靠命令自身 disabled + 页面确认框兜底）。
- **补发基于数据库轮次**：`session.history(since)` 只能补已落库的轮次；断线期间正在生成的那一轮不补逐帧内容，重连后由后续事件或 `message_completed` 覆盖收敛。
- **历史帧不是逐帧录像**：`frames` 只还原关键帧（每轮上限 200 帧、单条文本 4KB），而且只覆盖 `turn_events` 里已有的记录。若某轮在落库前就崩溃，过程会有缺口；`turn_events` 也没有 `session_id`，只能靠 `turn_id` 反查。
- **回放不支持审批回传**：审批帧能展示，但历史里点「同意/拒绝」没有意义（那一轮已经结束），闭环仍待 P2。
- **对话列表只读 thread 表**：`session.list` 依赖 `sessions` 表关联出归属，若某会话在 `sessions` 里没有记录（历史数据缺失），对应 thread 不会出现在列表中；标题/预览取首末轮，中间轮次不参与。
- **不支持审批回传**：`approval` 事件能下发，但客户端无法回应，审批闭环待 P2。
- **媒体原样透传**：`AgentData.media` 里的 `types.Media` 直接下发；内联 `base64` 仍可传，但受 1MB 单帧限制，稍大的文件请走 3.8 的上传端点。
- **文件只在单机**：`filestore` 落在本地磁盘、索引在内存，多实例部署时上传与下载必须落到同一进程（或共享同一 `file_dir` 并接受索引不共享），也没有对象存储与 CDN 加速。
- **附件不进历史**：`send_file` 推的是实时帧，不落 `turn_events`；回放时看不到当时发过的附件（文件多半也过期了）。
- **默认不做真实鉴权**：`DefaultAuthenticator` 把任何 token 都折成同一个固定用户，既不校验有效性，也让文件归属校验形同虚设（TODO）。上线前务必替换成查库或 JWT 校验。
- **无用户管理**：没有用户增删改、token 轮换，也没有 per-user 连接数/速率限制（`max_clients` 是全局的）。
- **无定向推送**：`Broadcast` 面向所有连接。
- **delta 合并在 bridge，不在 `conn.writePump`**：到 `conn` 那一层载荷已经序列化好了，
  合并只能重新解析 JSON，或者让每个连接各自 marshal 一次，而片段身份（`eventID`）在这一层也不必再读。
  所以合并放在 `bridge/delta_coalescer.go`（序列化之前、按片段合并），`conn` 的丢帧策略退回成纯兜底。
  代价是节奏按会话统一：某个慢标签页不会让别的连接拿到更细的帧，真卡死的页面仍由丢弃策略 + 重连补发处理。
- **合并帧没有周期性全量快照**：被丢弃或跳过的中间文本仍要等 `message_completed` 才收敛（与原策略一致）。
  若出现「长回答里文字跳得明显」，再考虑加一条秒级的全量快照帧，注意那会重新引入 O(n²) 带宽。

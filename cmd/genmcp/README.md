# genmcp

根据 Swagger 2.0 operation 生成 MCP 工具。请求继续由 `mcp/edp` 执行，生成文件写在
`mcp/serv/generated/`。

## 基本用法

```bash
go run ./cmd/genmcp \
  --api /nc/material/query \
  --out mcp/serv/generated/nc_material_query.go \
  --allow-method post
```

`--swagger` 默认依次读取 `--swagger`、`APIDOCS_URL`、`http://boda.bj.company/apidocs.json`。
`--api` 支持 `/nc/material/query` 或完整 URL。

默认会把请求和响应转换成 Go 结构体，直接写进 `--out` 文件。生成结果类似：

```go
type NcMaterialQueryRequest struct {
	Code string `json:"code" description:"编码"`
}

type NcMaterialQueryResponseItem struct {
	Name string `json:"name" description:"物料名称"`
}

openapi.NewTypedRequestResponseOpenAPITool[
	NcMaterialQueryRequest,
	[]NcMaterialQueryResponseItem,
]
```

不需要再传 `--request-struct` 或 `--response-struct`。

## 复用已有 Go 结构体

```bash
go run ./cmd/genmcp \
  --api /nc/material/query \
  --out mcp/serv/generated/nc_material_query.go \
  --allow-method post \
  --request-struct 'mcp/serv/generated/material_types.go#MaterialQuery' \
  --response-struct 'mcp/serv/generated/material_types.go#[]Material'
```

只有在希望复用已有类型或覆盖错误 Swagger 时才需要这两个参数。结构体规则：

- 请求和响应结构体会生成泛型工具，运行时通过反射生成入参 schema，并通过 `edp.DoClient[T]` 解析泛型 `Response[T]`，生成文件不再保存 JSON Schema 字符串。
- `json` 标签决定 API 字段名和输出顺序。
- `description` 标签保留字段中文含义。
- 没有 `omitempty` 的字段默认必填；指针和带 `omitempty` 的字段默认选填。
- `required:"true"` 或 `required:"false"` 可以显式覆盖必填规则。
- `[]Type` 表示数组，嵌套结构体、指针和 `map[string]Type` 会递归处理。
- 被引用类型可以定义在同一目录的其他 `.go` 文件中。

请求参数也可以使用 `--request-struct ./types.go#MaterialQuery`。对于 POST/PUT/PATCH，
非路径字段默认进入 JSON body；对于 GET/DELETE，默认进入 query。

## JSON 覆盖文件

Swagger 无法自动修复时，使用 `--override override.json`：

```json
{
  "method": "POST",
  "path": "/nc/material/query",
  "request": {
    "struct": "./mcp/serv/generated/material_types.go#MaterialQuery",
    "path": {"code": "materialPK"},
    "body": {"name": "name", "page": "page"}
  },
  "response": {
    "struct": "./mcp/serv/generated/material_types.go#[]Material"
  }
}
```

`request.path`、`request.query`、`request.body` 的 key 是 MCP 参数名，value 是真实 API
字段名。`request.parameters` 可以直接提供完整 JSON Schema，但不能和 `request.struct`
同时设置。`response.schema` 可以直接提供 Swagger/JSON Schema，但不能和
`response.struct` 同时设置。

## 校验

生成文件入库后可用以下命令检查是否已被手动修改：

```bash
go run ./cmd/genmcp --api /nc/material/query --out mcp/serv/generated/nc_material_query.go \
  --allow-method post --check
```

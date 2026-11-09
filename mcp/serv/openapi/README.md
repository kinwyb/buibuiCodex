# OpenAPI 紧凑输出协议

`schema-data/v1` 用于把 typed API 响应转换成大模型易读、低 token 的位置数组。
字段名和描述只出现一次，数据行只保留值。

## 协议

```json
{
  "format": "schema-data/v1",
  "layout": {
    "field": ["name", "required", "type"],
    "type": ["kind", "description", "children?"]
  },
  "schema": ["array", "", [
    ["name", 1, ["string", "物料名称"]],
    ["code", 1, ["string", "物料编码"]],
    ["materialconvert", 0, ["array", "计量单位集合", ["object", "", [
      ["measdoc", 1, ["string", "单位名称"]],
      ["measrate", 1, ["string", "转换率"]]
    ]]]]
  ]],
  "data": [
    ["物料A", "M001", [["件", "1"]]]
  ]
}
```

- `field` 元组固定为字段名、必填标记、类型。
- `type` 元组固定为类型、中文描述、子类型。
- `required` 使用 `1/0`。
- object 的数据是位置数组，顺序与 schema 字段顺序一致。
- array 保持原顺序；map 保持 JSON 对象；raw 保留原始 JSON 值。
- 缺失字段和显式 `null` 都输出为 `null`。
- schema 不存在的额外字段会按名称排序追加到对象末尾，避免数据丢失。

## 5KB 阈值

完整协议小于 5KB 时全部内联。

达到或超过 5KB 时：

1. schema 始终内联。
2. 只返回前 3 条数据。
3. 增加 `meta`：

```json
{
  "meta": {
    "total": 300,
    "shown": 3,
    "file": "http://host/tmp_file?file=20261106xxx.json"
  }
}
```

4. 完整数据写入临时文件，文件内容仍是完整的 `schema-data/v1`，包含同一个 schema 和全部数据。
5. 文件写入失败时回退为完整内联结果，不会返回缺少数据的预览。

`page` 存在时同时保留在预览文件和完整文件中。

## 手动 Struct 的必填规则

`JSONSchemaFromType` 按以下顺序判断请求字段是否出现在 `required` 中：

1. `required:"true"`：强制必填。
2. `required:"false"`：强制选填。
3. 没有显式 `required` 标签时：
   - 带 `omitempty`：选填。
   - 指针类型：选填。
   - 普通类型且没有 `omitempty`：必填。

例如：

```go
type materialQueryRequest struct {
    Code       string `json:"code,omitempty" description:"物料编码"`
    Name       *string `json:"name" description:"物料名称"`
    VFree      string `json:"vfreeName,omitempty" enum:"vfree1,vfree4,vfree6,vfree7" description:"辅助物料属性"`
    VFreeValue string `json:"vfreeValue,omitempty" required:"false" description:"辅助属性值"`
}
```

上例所有字段都是选填。`omitempty` 只影响工具参数 schema，不会自动过滤模型显式传入的空字符串。

字符串枚举使用 `enum:"值1,值2,值3"` 标签。它与 `omitempty` 组合后会生成选填的字符串枚举：

```json
{
  "type": "string",
  "enum": ["vfree1", "vfree4", "vfree6", "vfree7"],
  "description": "辅助物料属性"
}
```

`至少传一个`、`A 有值时 B 必传` 这类跨字段规则不能通过单个字段标签表达，需要写入工具描述并在执行前校验。

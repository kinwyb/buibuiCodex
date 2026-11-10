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
- 位置格式中，object 的数据是位置数组，顺序与 schema 字段顺序一致。
- array 保持原顺序；map 保持 JSON 对象；raw 保留原始 JSON 值。
- 缺失字段和显式 `null` 都输出为 `null`。
- schema 不存在的额外字段会按名称排序追加到对象末尾，避免数据丢失。

## 根 data 是对象

根数据是对象时不输出 `format`、`layout` 和 `schema`，直接使用字段说明作为 key，字段顺序与 schema 一致：

```json
{
  "data": {
    "物料编码": "M001",
    "物料名称": "物料A",
    "计量单位集合": [
      {"单位名称": "件", "转换率": "1"}
    ]
  }
}
```

- 字段说明为空、同一层重复或与其他字段名冲突时，回退到 JSON 字段名。
- 缺失字段和显式 `null` 仍然输出 `null`。
- schema 不存在的额外字段会按名称排序追加到对象末尾。
- `page` 与 `data` 同级保留。

## 5KB 阈值

完整协议小于 5KB 时全部内联。

根 `data` 是数组，且完整协议达到或超过 5KB 时：

1. schema 始终内联。
2. 只返回前 3 条数据。
3. 增加 `meta`：

```json
{
  "meta": {
    "total": 300,
    "shown": 3,
    "file": "http://host/tmp_file?file=20261106xxx.json",
    "fileHint": "文件内容是上方 data 数组；按上方 schema 的字段顺序读取"
  }
}
```

4. 完整数据写入临时文件。文件内容只有当前响应的 `data` 字段，不再重复 `format`、`layout`、`schema` 和 `page`。
5. 文件写入失败时回退为完整内联结果，不会返回缺少数据的预览。

下载文件后应直接按上面的 `schema` 解释文件内容，不需要再读取 `.data` 字段。`page` 只保留在预览响应中。

根 `data` 是对象时，命名对象小于 5KB 直接内联。达到或超过 5KB 且包含对象数组或对象字典时，会再按位置格式编码一次，取字节数更短的结果；对象结果不会写入临时文件。

根 `data` 是基础类型或 `raw` 时仍使用原 schema 格式并完整内联，不生成文件。

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

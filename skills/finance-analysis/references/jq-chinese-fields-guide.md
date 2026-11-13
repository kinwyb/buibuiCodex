# jq 中文字段使用指南

财务返回数据的字段几乎全是中文（如 `订单号`、`所属工段`、`核算要素名称`、`金额`、`毛利率(小数)`）。使用 `jq` 时必须遵守以下写法，否则会报 `jq: error: ... is not defined`。

## 核心规则

`jq` 的 `.字段名` 简写只支持纯 ASCII 标识符（字母数字下划线），**中文、带括号、含特殊字符的字段一律使用 `.["字段名"]`**。

```text
正确：jq '.["订单号"]' file.json
错误：jq '.订单号' file.json              # 中文键简写，jq 编译失败

正确：jq '.["毛利率(小数)"]' file.json
错误：jq '.毛利率(小数)' file.json        # 带括号字段简写，jq 编译失败
```

**对象字面量的 key 也必须加双引号**（不能写裸中文）：

```text
正确：{ "要素": .["核算要素名称"], "金额": (.["金额"] // "0" | tonumber) }
错误：{ 要素: .["核算要素名称"] }         # jq 报 INVALID_CHARACTER
```

**数值字段**统一加 `// "0" | tonumber` 防空值或非数字字符串；空字符串 `""` 不能直接 `tonumber`。

**Shell 转义**：jq 表达式必须用**单引号**整体包裹，避免 `$`、反引号、空格被 Shell 解释。

```bash
# 正确
jq '[.["data"][] | .["金额"] // "0" | tonumber] | add' input-output.json

# 错误：双引号 + 点号简写
jq "[.data[].金额 | tonumber] | add" input-output.json
```

**中文输出**：提取原始字符串时加 `-r` / `--raw-output`：

```bash
jq -r '.["data"][0]["订单号"]' file.json
```

**先验证格式**：拿到文件后先执行 `head -1 file.json | jq .` 确认能解析，再用 `jq '.["columns"]' file.json` 核对字段名拼写。

## 错误对照表

| 场景 | 错误写法 | 正确写法 | 报错原因 |
|------|----------|----------|----------|
| 取中文字段 | `.订单号` | `.["订单号"]` | jq 不支持中文标识符 |
| 取带括号字段 | `.毛利率(小数)` | `.["毛利率(小数)"]` | 点号后是非法字符 |
| 数字字段汇总 | `tonumber` | `(."金额" // "0") \| tonumber` | 空字符串 tonumber 报错 |
| 条件比较数字 | `> 100` | `(.["金额"] \| tonumber) > 100` | 类型不匹配 |
| 中文输出 | 默认输出 | `jq -r` | 默认带 JSON 字符串引号 |
| Shell 传中文 | `"$(cat)"` + 双引号 | `'...'` 单引号 | 双引号内 $ 反引号被解释 |

## 投入产出常用查询

```bash
# 验证 JSON 与查看字段名
head -1 input-output.json | jq .
jq '.["columns"]' input-output.json
jq '.["data"] | length' input-output.json

# 汇总全部金额
jq '[.["data"][] | .["金额"] // "0" | tonumber] | add' input-output.json

# 按核算要素名称分组汇总
jq '
  [.["data"][] | { "要素": .["核算要素名称"], "金额": (.["金额"] // "0" | tonumber) }]
  | group_by(.["要素"])
  | map({ "要素": .[0]["要素"], "合计": ([.[] | .["金额"]] | add), "行数": length })
' input-output.json

# 按所属工段 + 核算要素两级分组
jq '
  [.["data"][] | { "工段": .["所属工段"], "要素": .["核算要素名称"], "金额": (.["金额"] // "0" | tonumber) }]
  | group_by(.["工段"] + "|" + .["要素"])
  | map({ "工段": .[0]["工段"], "要素": .[0]["要素"], "合计": ([.[] | .["金额"]] | add) })
' input-output.json

# 提取原材料工段
jq '[.["data"][] | select(.["所属工段"] == "原材料")]' input-output.json

# 原材料工段总金额
jq '[.["data"][] | select(.["所属工段"] == "原材料") | .["金额"] // "0" | tonumber] | add' input-output.json

# 单价 > 50 的高单价行
jq '[.["data"][] | select((.["单价"] // "0" | tonumber) > 50) | { "物料": .["材料名称"], "单价": (.["单价"] | tonumber), "金额": (.["金额"] | tonumber) }]' input-output.json

# 负金额/冲销记录
jq '[.["data"][] | select((.["金额"] // "0" | tonumber) < 0) | { "物料": .["材料名称"], "金额": (.["金额"] | tonumber), "数量": (.["数量"] | tonumber) }]' input-output.json

# 数量按物料编码 Top 10
jq '
  [.["data"][] | { "编码": .["材料编码"], "数量": (.["数量"] // "0" | tonumber) }]
  | group_by(.["编码"])
  | map({ "编码": .[0]["编码"], "数量": ([.[] | .["数量"]] | add) })
  | sort_by(-.["数量"])
  | .[0:10]
' input-output.json
```

## 存货明细常用查询

```bash
# 月度加权均价
jq '
  [.["data"][] | { "期间": .["会计期间"], "数量": (.["主数量"] // "0" | tonumber), "金额": (.["金额"] // "0" | tonumber) }]
  | group_by(.["期间"])
  | map({ "期间": .[0]["期间"], "数量": ([.[] | .["数量"]] | add), "金额": ([.[] | .["金额"]] | add), "加权均价": (([.[] | .["金额"]] | add) / ([.[] | .["数量"]] | add)) })
' inventory-ledger.json

# 按单据类型汇总
jq '
  [.["data"][] | { "类型": .["单据类型"], "金额": (.["金额"] // "0" | tonumber) }]
  | group_by(.["类型"])
  | map({ "类型": .[0]["类型"], "金额": ([.[] | .["金额"]] | add), "行数": length })
  | sort_by(-.["金额"])
' inventory-ledger.json

# 期间环比（按加权均价）
jq '
  [.["data"][] | { "期间": .["会计期间"], "数量": (.["主数量"] // "0" | tonumber), "金额": (.["金额"] // "0" | tonumber) }]
  | group_by(.["期间"])
  | map({ "期间": .[0]["期间"], "均价": (([.[] | .["金额"]] | add) / ([.[] | .["数量"]] | add)) })
  | sort_by(.["期间"])
  | . as $rows | range(1; length) as $i
  | { "上期": $rows[$i - 1]["期间"], "本期": $rows[$i]["期间"], "变化%": (($rows[$i]["均价"] - $rows[$i - 1]["均价"]) / $rows[$i - 1]["均价"] * 100) }
' inventory-ledger.json

# 单价 > 期间均价 50% 的异常高价单据
jq '
  [.["data"][] | { "期间": .["会计期间"], "单价": (.["单价"] // "0" | tonumber), "数量": (.["主数量"] // "0" | tonumber), "金额": (.["金额"] // "0" | tonumber) }]
  | group_by(.["期间"])
  | map({ "期间": .[0]["期间"], "均价": (([.[] | .["金额"]] | add) / ([.[] | .["数量"]] | add)), "单据": . })
  | .[] | .["单据"] as $rows | .["均价"] as $avg
  | $rows[] | select(.["单价"] > $avg * 1.5) | { "期间": .["期间"], "单价": .["单价"], "vs均价": ((.["单价"] - $avg) / $avg) }
' inventory-ledger.json
```

## 替代工具

复杂查询（跨文件 join、统计回归、CSV 透视）优先使用 Python：

```bash
# Python 单行调用
python3 -c "
import json
data = json.load(open('input-output.json'))['data']
rows = [r for r in data if r.get('所属工段') == '原材料']
total = sum(float(r.get('金额') or 0) for r in rows)
print(f'原材料金额: {total:,.2f} 元，行数 {len(rows)}')
"

# 数字验算
echo "scale=4; 16400 / 100" | bc                 # 单位成本
awk 'BEGIN{printf "%.4f\n", 3600/20000*100}'      # 毛利率百分比
```

数字验算优先 `bc -l` 保精度；简单对比用 `awk`；复杂分组、回归改用 Python。

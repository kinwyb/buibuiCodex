#!/usr/bin/env python3
"""
财务数据探查工具 —— 快速查看单位成本、投入产出、存货明细的数据概况。
用法: python3 inspect_finance_data.py <文件路径> [选项]

选项:
  --ucs          按单位成本格式解析（查看订单、物料、毛利率等）
  --io           按投入产出格式解析（查看工段分布、核算要素等）
  --ledger       按存货明细格式解析（查看物料分布、期间分布等）
  --material 编码  只看指定物料编码的数据
  --stage 工段     只看指定工段（投入产出）
  --limit N      最多显示N行样例（默认10）
"""
import json
import sys
from collections import Counter, defaultdict


def safe_float(v):
    try:
        return float(v) if v not in (None, "", "-") else 0.0
    except (ValueError, TypeError):
        return 0.0


def inspect_ucs(data, limit=10):
    records = data.get("data", [])
    fields = data.get("columns") or data.get("data_short", [])
    print(f"字段数: {len(fields)}")
    print(f"记录数: {len(records)}")
    if not records:
        return
    print(f"\n字段列表: {', '.join(fields[:20])}{'...' if len(fields) > 20 else ''}")
    print("\n=== 订单概览 ===")
    print(f"{'物料编码':<14} {'物料名称':<20} {'订单号':<14} {'产量':>8} {'完工成本':>14} {'单价':>12} {'毛利率':>8}")
    for r in records[:limit]:
        print(
            f"{r.get('物料编码',''):<14} {r.get('物料名称','')[:18]:<20} "
            f"{r.get('订单号',''):<14} {safe_float(r.get('完工产量')):>8.0f} "
            f"{safe_float(r.get('完工成本')):>14,.0f} {safe_float(r.get('销售单价(无税)')):>12,.2f} "
            f"{r.get('毛利率',''):>8}"
        )


def inspect_io(data, material=None, stage=None, limit=10):
    records = data.get("data", [])
    fields = data.get("columns") or data.get("data_short", [])
    print(f"字段数: {len(fields)}")
    print(f"记录数: {len(records)}")
    if not records:
        return

    rows = records
    if material:
        rows = [r for r in rows if r.get("物料编码") == material]
    if stage:
        rows = [r for r in rows if r.get("所属工段") == stage]

    print(f"\n过滤后记录数: {len(rows)}")
    print(f"字段列表: {', '.join(fields[:15])}{'...' if len(fields) > 15 else ''}")

    # 工段分布
    print("\n=== 所属工段分布 ===")
    c = Counter(r.get("所属工段", "未分类") for r in rows)
    for sec, cnt in c.most_common():
        amt = sum(safe_float(r.get("金额")) for r in rows if r.get("所属工段") == sec)
        print(f"  {sec}: {cnt:>5} 行 | 金额 {amt:>18,.2f}")

    # 物料分布
    print("\n=== 物料编码分布（Top 10）===")
    c = Counter((r.get("物料编码"), r.get("物料名称")) for r in rows)
    for (code, name), cnt in c.most_common(10):
        amt = sum(safe_float(r.get("金额")) for r in rows if r.get("物料编码") == code)
        print(f"  {code} ({name or 'N/A'}): {cnt:>5} 行 | 金额 {amt:>18,.2f}")

    # 核算要素分布
    print("\n=== 核算要素分布（Top 10）===")
    c = Counter(r.get("核算要素名称", "未分类") for r in rows)
    for elem, cnt in c.most_common(10):
        amt = sum(safe_float(r.get("金额")) for r in rows if r.get("核算要素名称") == elem)
        print(f"  {elem}: {cnt:>5} 行 | 金额 {amt:>18,.2f}")

    # 样例数据
    print(f"\n=== 前 {limit} 行样例 ===")
    for r in rows[:limit]:
        print(f"  工段={r.get('所属工段',''):<6} 要素={r.get('核算要素名称',''):<12} "
              f"物料={r.get('物料编码',''):<14} 材料={r.get('材料编码',''):<14} "
              f"金额={safe_float(r.get('金额')):>14,.2f} 数量={r.get('数量',''):>10} 单价={r.get('单价',''):>14}")


def inspect_ledger(data, material=None, limit=10):
    records = data.get("data", [])
    fields = data.get("columns") or data.get("data_short", [])
    print(f"字段数: {len(fields)}")
    print(f"记录数: {len(records)}")
    if not records:
        return

    rows = records
    if material:
        rows = [r for r in rows if r.get("物料编码") == material]

    print(f"\n过滤后记录数: {len(rows)}")

    # 物料分布
    print("\n=== 物料编码分布 ===")
    c = Counter((r.get("物料编码"), r.get("物料名称")) for r in rows)
    for (code, name), cnt in c.most_common():
        amt = sum(safe_float(r.get("金额")) for r in rows if r.get("物料编码") == code)
        qty = sum(safe_float(r.get("主数量")) for r in rows if r.get("物料编码") == code)
        print(f"  {code} ({name or 'N/A'}): {cnt:>6} 行 | 数量 {qty:>12,.2f} | 金额 {amt:>18,.2f}")

    # 期间分布
    print("\n=== 会计期间分布 ===")
    c = Counter(r.get("会计期间", "未分类") for r in rows)
    for period, cnt in c.most_common():
        amt = sum(safe_float(r.get("金额")) for r in rows if r.get("会计期间") == period)
        print(f"  {period}: {cnt:>6} 行 | 金额 {amt:>18,.2f}")

    # 单据类型分布
    print("\n=== 单据类型分布 ===")
    c = Counter(r.get("单据类型", "未分类") for r in rows)
    for doc_type, cnt in c.most_common():
        amt = sum(safe_float(r.get("金额")) for r in rows if r.get("单据类型") == doc_type)
        print(f"  {doc_type}: {cnt:>6} 行 | 金额 {amt:>18,.2f}")

    # 样例
    print(f"\n=== 前 {limit} 行样例 ===")
    for r in rows[:limit]:
        print(f"  期间={r.get('会计期间',''):<10} 类型={r.get('单据类型',''):<10} "
              f"物料={r.get('物料编码',''):<14} 数量={r.get('主数量',''):>10} "
              f"单价={r.get('单价',''):>14} 金额={safe_float(r.get('金额')):>14,.2f}")


def main():
    args = sys.argv[1:]
    if not args:
        print(__doc__)
        sys.exit(1)

    path = args[0]
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)

    mode = "auto"
    material = None
    stage = None
    limit = 10

    i = 1
    while i < len(args):
        a = args[i]
        if a == "--ucs":
            mode = "ucs"
        elif a == "--io":
            mode = "io"
        elif a == "--ledger":
            mode = "ledger"
        elif a == "--material" and i + 1 < len(args):
            material = args[i + 1]
            i += 1
        elif a == "--stage" and i + 1 < len(args):
            stage = args[i + 1]
            i += 1
        elif a == "--limit" and i + 1 < len(args):
            limit = int(args[i + 1])
            i += 1
        i += 1

    if mode == "auto":
        fields = data.get("columns") or data.get("data_short", [])
        if "所属工段" in fields:
            mode = "io"
        elif "会计期间" in fields:
            mode = "ledger"
        elif "完工成本" in fields or "单位成本" in fields:
            mode = "ucs"
        else:
            mode = "io"
        print(f"自动识别模式: {mode}\n")

    if mode == "ucs":
        inspect_ucs(data, limit=limit)
    elif mode == "io":
        inspect_io(data, material=material, stage=stage, limit=limit)
    elif mode == "ledger":
        inspect_ledger(data, material=material, limit=limit)


if __name__ == "__main__":
    main()

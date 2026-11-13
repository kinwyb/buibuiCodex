#!/usr/bin/env python3
"""
订单毛利率分析脚本
用法: python3 analyze_margin.py <json文件路径> [选项]

默认输出全部分析（等同 --all）。
支持从 stdin 读取 JSON: cat data.json | python3 analyze_margin.py -
"""

import json
import sys
import argparse
from collections import defaultdict

def load_data(path):
    if path == "-":
        return json.loads(sys.stdin.read())
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)

def safe_float(val):
    if val is None or val == "":
        return None
    return float(val)

def parse_records(raw):
    records = []
    for row in raw["data"]:
        margin_rate = safe_float(row.get("毛利率(小数)"))
        revenue = safe_float(row.get("销售无税金额"))
        cost = safe_float(row.get("成本金额"))
        gross_profit = safe_float(row.get("毛利"))
        quantity = safe_float(row.get("产量"))
        records.append({
            "period": row.get("会计期间", ""),
            "customer": row.get("客户", ""),
            "order_type": row.get("订单类型", ""),
            "channel": row.get("销售渠道类型", ""),
            "department": row.get("销售部门", ""),
            "order": row.get("订单号", ""),
            "material_code": row.get("物料编码", ""),
            "material_name": row.get("物料名称", ""),
            "quantity": quantity,
            "revenue": revenue,
            "cost": cost,
            "gross_profit": gross_profit,
            "margin_rate": margin_rate,
            "margin_rate_display": row.get("毛利率", ""),
        })
    return records

def tier_label(rate):
    if rate >= 0.30:
        return "优秀(≥30%)"
    elif rate >= 0.20:
        return "良好(20%-30%)"
    elif rate >= 0.10:
        return "一般(10%-20%)"
    elif rate >= 0.0:
        return "预警(0%-10%)"
    else:
        return "异常(<0%)"

def print_summary(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    invalid = [r for r in records if r["margin_rate"] is None]
    total_count = len(records)
    valid_count = len(valid)

    total_revenue = sum(r["revenue"] for r in valid if r["revenue"] is not None)
    total_cost = sum(r["cost"] for r in valid if r["cost"] is not None)
    total_profit = sum(r["gross_profit"] for r in valid if r["gross_profit"] is not None)
    weighted_margin = (total_revenue - total_cost) / total_revenue * 100 if total_revenue else 0
    avg_margin = sum(r["margin_rate"] for r in valid) / valid_count * 100 if valid_count else 0

    max_r = max(valid, key=lambda x: x["margin_rate"]) if valid else None
    min_r = min(valid, key=lambda x: x["margin_rate"]) if valid else None
    loss_count = len([r for r in valid if r["margin_rate"] < 0])
    loss_profit = sum(r["gross_profit"] for r in valid if r["margin_rate"] < 0 and r["gross_profit"] is not None)

    customers = set(r["customer"] for r in valid if r["customer"])
    departments = set(r["department"] for r in valid if r["department"])

    print("=" * 60)
    print("                    毛利率分析总览")
    print("=" * 60)
    print(f"订单记录总数:    {total_count}")
    print(f"有效记录数:      {valid_count}")
    print(f"无效记录数:      {len(invalid)} (毛利率为空)")
    print(f"涉及客户数:      {len(customers)}")
    print(f"涉及销售部门数:  {len(departments)}")
    print(f"销售无税金额合计: {total_revenue:,.2f}")
    print(f"成本金额合计:     {total_cost:,.2f}")
    print(f"毛利合计:         {total_profit:,.2f}")
    print(f"加权平均毛利率:   {weighted_margin:.2f}%")
    print(f"算术平均毛利率:   {avg_margin:.2f}%")
    print(f"亏损订单数:       {loss_count} (占比 {loss_count/valid_count*100:.1f}%)" if valid_count else "亏损订单数: 0")
    print(f"亏损订单毛利合计: {loss_profit:,.2f}" if loss_count else "")
    if max_r:
        print(f"最高毛利率:       {max_r['margin_rate']*100:.2f}% ({max_r['order']} / {max_r['material_name']} / 客户:{max_r['customer']})")
    if min_r:
        print(f"最低毛利率:       {min_r['margin_rate']*100:.2f}% ({min_r['order']} / {min_r['material_name']} / 客户:{min_r['customer']})")
    print()

def print_tier(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    tiers = defaultdict(lambda: {"count": 0, "revenue": 0.0, "profit": 0.0})
    for r in valid:
        label = tier_label(r["margin_rate"])
        tiers[label]["count"] += 1
        if r["revenue"] is not None:
            tiers[label]["revenue"] += r["revenue"]
        if r["gross_profit"] is not None:
            tiers[label]["profit"] += r["gross_profit"]

    order_labels = ["优秀(≥30%)", "良好(20%-30%)", "一般(10%-20%)", "预警(0%-10%)", "异常(<0%)"]
    total = len(valid)
    print("=" * 60)
    print("                    毛利率分布")
    print("=" * 60)
    print(f"{'毛利率范围':<16} {'订单数':>6} {'占比':>8} {'销售金额':>16} {'毛利':>16}")
    print("-" * 70)
    for label in order_labels:
        t = tiers[label]
        pct = t["count"] / total * 100 if total else 0
        print(f"{label:<16} {t['count']:>6} {pct:>7.1f}% {t['revenue']:>16,.2f} {t['profit']:>16,.2f}")
    print()

def print_loss(records):
    loss = [r for r in records if r["margin_rate"] is not None and r["margin_rate"] < 0]
    loss.sort(key=lambda x: x["margin_rate"])
    print("=" * 60)
    print(f"              亏损订单明细 (共 {len(loss)} 条)")
    print("=" * 60)
    print(f"{'订单号':<18} {'客户':<10} {'物料名称':<28} {'销售部门':<14} {'产量':>8} {'销售金额':>14} {'成本金额':>14} {'毛利':>14} {'毛利率':>8}")
    print("-" * 140)
    total_loss_profit = 0.0
    for r in loss:
        qty = f"{r['quantity']:,.0f}" if r["quantity"] is not None else "-"
        rev = f"{r['revenue']:,.2f}" if r["revenue"] is not None else "-"
        cost = f"{r['cost']:,.2f}" if r["cost"] is not None else "-"
        profit = f"{r['gross_profit']:,.2f}" if r["gross_profit"] is not None else "-"
        if r["gross_profit"] is not None:
            total_loss_profit += r["gross_profit"]
        print(f"{r['order']:<18} {r['customer']:<10} {r['material_name']:<28} {r['department']:<14} {qty:>8} {rev:>14} {cost:>14} {profit:>14} {r['margin_rate_display']:>8}")
    print("-" * 140)
    print(f"{'亏损合计':<72} {'':>8} {'':>14} {'':>14} {total_loss_profit:>14,.2f}")
    print()

def print_low(records):
    low = [r for r in records if r["margin_rate"] is not None and 0 <= r["margin_rate"] < 0.10]
    low.sort(key=lambda x: x["margin_rate"])
    print("=" * 60)
    print(f"           低毛利订单明细 (0%-10%, 共 {len(low)} 条)")
    print("=" * 60)
    print(f"{'订单号':<18} {'客户':<10} {'物料名称':<28} {'销售部门':<14} {'产量':>8} {'销售金额':>14} {'成本金额':>14} {'毛利':>14} {'毛利率':>8}")
    print("-" * 140)
    for r in low:
        qty = f"{r['quantity']:,.0f}" if r["quantity"] is not None else "-"
        rev = f"{r['revenue']:,.2f}" if r["revenue"] is not None else "-"
        cost = f"{r['cost']:,.2f}" if r["cost"] is not None else "-"
        profit = f"{r['gross_profit']:,.2f}" if r["gross_profit"] is not None else "-"
        print(f"{r['order']:<18} {r['customer']:<10} {r['material_name']:<28} {r['department']:<14} {qty:>8} {rev:>14} {cost:>14} {profit:>14} {r['margin_rate_display']:>8}")
    print()

def _print_group(title, groups, key_label, key_width):
    print("=" * 60)
    print(f"              {title}")
    print("=" * 60)
    print(f"{key_label:<{key_width}} {'记录数':>6} {'销售金额':>16} {'成本金额':>16} {'毛利':>16} {'加权毛利率':>10}")
    print("-" * (key_width + 70))
    sorted_groups = sorted(groups.items(), key=lambda x: (x[1]["profit"] / x[1]["revenue"] if x[1]["revenue"] else 0))
    for key, g in sorted_groups:
        margin = (g["revenue"] - g["cost"]) / g["revenue"] * 100 if g["revenue"] else 0
        print(f"{key:<{key_width}} {g['count']:>6} {g['revenue']:>16,.2f} {g['cost']:>16,.2f} {g['profit']:>16,.2f} {margin:>9.2f}%")
    print()

def print_group_by_order(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    groups = defaultdict(lambda: {"count": 0, "revenue": 0.0, "cost": 0.0, "profit": 0.0})
    for r in valid:
        g = groups[r["order"]]
        g["count"] += 1
        if r["revenue"]: g["revenue"] += r["revenue"]
        if r["cost"]: g["cost"] += r["cost"]
        if r["gross_profit"]: g["profit"] += r["gross_profit"]
    _print_group("按订单号汇总 (加权毛利率)", groups, "订单号", 20)

def print_group_by_material(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    groups = defaultdict(lambda: {"count": 0, "revenue": 0.0, "cost": 0.0, "profit": 0.0, "quantity": 0.0})
    for r in valid:
        g = groups[r["material_name"]]
        g["count"] += 1
        if r["revenue"]: g["revenue"] += r["revenue"]
        if r["cost"]: g["cost"] += r["cost"]
        if r["gross_profit"]: g["profit"] += r["gross_profit"]
        if r["quantity"]: g["quantity"] += r["quantity"]
    _print_group("按物料名称汇总 (加权毛利率)", groups, "物料名称", 36)

def print_group_by_customer(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    groups = defaultdict(lambda: {"count": 0, "revenue": 0.0, "cost": 0.0, "profit": 0.0})
    for r in valid:
        key = r["customer"] if r["customer"] else "(未知)"
        g = groups[key]
        g["count"] += 1
        if r["revenue"]: g["revenue"] += r["revenue"]
        if r["cost"]: g["cost"] += r["cost"]
        if r["gross_profit"]: g["profit"] += r["gross_profit"]
    _print_group("按客户汇总 (加权毛利率)", groups, "客户", 16)

def print_group_by_department(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    groups = defaultdict(lambda: {"count": 0, "revenue": 0.0, "cost": 0.0, "profit": 0.0})
    for r in valid:
        key = r["department"] if r["department"] else "(未知)"
        g = groups[key]
        g["count"] += 1
        if r["revenue"]: g["revenue"] += r["revenue"]
        if r["cost"]: g["cost"] += r["cost"]
        if r["gross_profit"]: g["profit"] += r["gross_profit"]
    _print_group("按销售部门汇总 (加权毛利率)", groups, "销售部门", 18)

def print_group_by_channel(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    groups = defaultdict(lambda: {"count": 0, "revenue": 0.0, "cost": 0.0, "profit": 0.0})
    for r in valid:
        key = r["channel"] if r["channel"] else "(未知)"
        g = groups[key]
        g["count"] += 1
        if r["revenue"]: g["revenue"] += r["revenue"]
        if r["cost"]: g["cost"] += r["cost"]
        if r["gross_profit"]: g["profit"] += r["gross_profit"]
    _print_group("按销售渠道类型汇总 (加权毛利率)", groups, "销售渠道", 16)

def print_group_by_order_type(records):
    valid = [r for r in records if r["margin_rate"] is not None]
    groups = defaultdict(lambda: {"count": 0, "revenue": 0.0, "cost": 0.0, "profit": 0.0})
    for r in valid:
        key = r["order_type"] if r["order_type"] else "(未知)"
        g = groups[key]
        g["count"] += 1
        if r["revenue"]: g["revenue"] += r["revenue"]
        if r["cost"]: g["cost"] += r["cost"]
        if r["gross_profit"]: g["profit"] += r["gross_profit"]
    _print_group("按订单类型汇总 (加权毛利率)", groups, "订单类型", 16)

def main():
    parser = argparse.ArgumentParser(description="订单毛利率分析")
    parser.add_argument("file", help="JSON 文件路径, 用 '-' 从 stdin 读取")
    parser.add_argument("--summary", action="store_true", help="输出总览统计")
    parser.add_argument("--tier", action="store_true", help="输出毛利率分布")
    parser.add_argument("--loss", action="store_true", help="输出亏损订单明细")
    parser.add_argument("--low", action="store_true", help="输出低毛利订单明细")
    parser.add_argument("--group-by-order", action="store_true", help="按订单号汇总")
    parser.add_argument("--group-by-material", action="store_true", help="按物料名称汇总")
    parser.add_argument("--group-by-customer", action="store_true", help="按客户汇总")
    parser.add_argument("--group-by-department", action="store_true", help="按销售部门汇总")
    parser.add_argument("--group-by-channel", action="store_true", help="按销售渠道类型汇总")
    parser.add_argument("--group-by-order-type", action="store_true", help="按订单类型汇总")
    parser.add_argument("--all", action="store_true", help="输出全部分析(默认)")
    args = parser.parse_args()

    group_flags = [
        args.summary, args.tier, args.loss, args.low,
        args.group_by_order, args.group_by_material,
        args.group_by_customer, args.group_by_department,
        args.group_by_channel, args.group_by_order_type,
    ]
    no_flag = not any(group_flags + [args.all])
    if no_flag:
        args.all = True

    raw = load_data(args.file)
    records = parse_records(raw)

    if args.all or args.summary:
        print_summary(records)
    if args.all or args.tier:
        print_tier(records)
    if args.all or args.loss:
        print_loss(records)
    if args.all or args.low:
        print_low(records)
    if args.all or args.group_by_order:
        print_group_by_order(records)
    if args.all or args.group_by_material:
        print_group_by_material(records)
    if args.all or args.group_by_customer:
        print_group_by_customer(records)
    if args.all or args.group_by_department:
        print_group_by_department(records)
    if args.all or args.group_by_channel:
        print_group_by_channel(records)
    if args.all or args.group_by_order_type:
        print_group_by_order_type(records)

if __name__ == "__main__":
    main()

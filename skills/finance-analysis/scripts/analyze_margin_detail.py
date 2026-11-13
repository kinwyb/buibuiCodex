#!/usr/bin/env python3
"""
毛利率明细分析脚本
用法:
  python3 analyze_margin_detail.py <单位成本.json> [投入产出.json] [存货明细.json]

默认输出全部分析（等同 --all）。
支持选项:
  --summary         成本结构总览（按订单）
  --io              投入产出分析（按核算要素名称分组）
  --price           存货价格分析（月度加权均价、异常单据）
  --anomaly         异常识别汇总（成本结构超阈值、数据一致性）
  --all             输出全部分析（默认）
"""

import json
import sys
import argparse
from collections import defaultdict

# 成本阈值（参考 references/gross-margin-detail.md）
THRESHOLDS = {
    "原材料": (0.30, 0.05),
    "辅料": (0.10, 0.05),
    "直接人工": (0.35, 0.05),
    "制造费用": (0.20, 0.05),
    "外协加工费": (0.05, 0.05),
}

# 阶段净成本规则：上游半成品/原料只作为追溯入口，不计入当前工段净成本。
# 例如包装投入里的"缝制成品"属于缝制阶段，不能算作包装阶段成本。
# 原料 / 纱线等已被独立为"原材料"工段，不再混入织造；保留少量防御性关键词
# 是为了兼容老数据中仍把"其他原料/原材料/纱线"留在织造工段的情况。
STAGE_CARRY_IN_RULES = [
    ("包装", ["缝制成品"]),
    ("缝制", ["毛坯"]),
    ("染色", ["白坯", "坯布"]),
    ("织造", ["白坯", "毛坯", "坯布", "其他原料", "纱线"]),
    ("原材料", []),
]


STAGE_ORDER = {"包装": 0, "缝制": 1, "染色": 2, "织造": 3, "原材料": 4, "未识别": 99}
EXPECTED_STAGES = ["包装", "缝制", "染色", "织造", "原材料"]
CHAIN_PARENT = {"缝制": "包装", "染色": "缝制", "织造": "染色", "原材料": "织造"}

# 投入产出口径：按"所属工段 + 核算要素名称"映射到标准五大成本项目。
# 包装层（包装辅料）和缝制层（缝制辅料）的辅料行通过关键词匹配归入辅料；
# 原料 / 纱线 / 其他原料 默认全部归入"原材料"成本项目，不再混入织造。
IO_COST_CATEGORIES = {
    "原材料": ["原料", "纱线", "其他原料"],
    "辅料": ["包装辅料", "缝制辅料", "公共缝制辅料", "公共包装辅料"],
    "直接人工": ["直接人工", "间接人工", "福利费", "五险一金", "劳务费"],
    "制造费用": ["折旧费", "水费", "电费", "水电费", "修理费", "低值易耗品领用摊销", "汇总其他"],
    "外协加工费": ["加工费", "委外加工费"],
}

# 阶段 carry 规则：上游半成品 / 原料只作为追溯入口，不计入当前工段净成本。
# 原材料工段没有下一层，但所有行直接归入"原材料"成本项目（"原材料"语义上即最终原料）。
IO_STAGE_CARRY = [
    ("包装", ["缝制成品"]),
    ("缝制", ["毛坯"]),
    ("染色", ["白坯", "坯布"]),
    ("织造", ["白坯", "毛坯", "坯布"]),
    ("原材料", []),
]


def load_json(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def safe_float(val):
    if val is None or val == "":
        return 0.0
    try:
        return float(val)
    except (ValueError, TypeError):
        return 0.0


def fmt_money(v, digits=2):
    if v is None:
        return "-"
    return f"{v:,.{digits}f}"


def fmt_pct(v, digits=2):
    if v is None:
        return "-"
    return f"{v * 100:.{digits}f}%"


def tier_label(rate):
    if rate >= 0.30:
        return "优秀(>=30%)"
    if rate >= 0.20:
        return "良好(20%-30%)"
    if rate >= 0.10:
        return "一般(10%-20%)"
    if rate >= 0.0:
        return "预警(0%-10%)"
    return "异常(<0%)"


def classify_io_element(element_name, stage=None):
    """把投入产出的核算要素名称映射到标准五大成本项目。"""
    name = element_name or ""
    if stage == "原材料":
        return "原材料"
    for cat, keywords in IO_COST_CATEGORIES.items():
        if any(k in name for k in keywords):
            return cat
    return "未分类"


def carry_keywords_for_stage(stage):
    for rule_stage, keywords in IO_STAGE_CARRY:
        if rule_stage == stage:
            return set(keywords)
    return set()


def _walk_chain(root_code, io_rows, rows_by_material):
    """沿 carry 链逐级展开，返回 [(stage, scale, stage_rows), ...]。"""
    result = []
    current_materials = {root_code}
    parent_materials = {root_code}
    last_scale = 1.0
    chain_broken = False

    for stage in EXPECTED_STAGES:
        if chain_broken:
            stage_rows = [r for r in io_rows if r.get("所属工段") == stage]
            scale = last_scale
            if stage_rows:
                result.append((stage, scale, stage_rows))
            continue

        stage_rows = []
        for code in current_materials:
            for r in rows_by_material.get(code, []):
                if r.get("所属工段") == stage:
                    stage_rows.append(r)

        if not stage_rows:
            chain_broken = True
            stage_rows = [r for r in io_rows if r.get("所属工段") == stage]
            scale = last_scale
            if stage_rows:
                result.append((stage, scale, stage_rows))
            continue

        if stage == "包装":
            scale = 1.0
            last_scale = 1.0
        else:
            parent = CHAIN_PARENT[stage]
            parent_carry = 0.0
            carry_kw = carry_keywords_for_stage(parent)
            for code in parent_materials:
                for r in rows_by_material.get(code, []):
                    if r.get("所属工段") == parent and r.get("核算要素名称") in carry_kw:
                        parent_carry += safe_float(r.get("金额"))
            stage_total = sum(safe_float(r.get("金额")) for r in stage_rows)
            scale = parent_carry / stage_total if stage_total > 0 else 0.0
            last_scale = scale

        result.append((stage, scale, stage_rows))
        parent_materials = current_materials

        carry_kw = carry_keywords_for_stage(stage)
        next_materials = set()
        for r in stage_rows:
            element = r.get("核算要素名称", "")
            if element in carry_kw and r.get("材料编码"):
                next_materials.add(r.get("材料编码"))

        if not next_materials and stage not in ("织造", "原材料"):
            chain_broken = True
        elif next_materials:
            current_materials = next_materials

    return result


def compute_cost_structure_from_io(ucs_records, io_rows):
    """按投入产出口径逐级展开计算各订单的成本项目标准占比。

    算法：
    1. 按 物料编码 索引 io 行（每个工段行的 物料编码 = 该工段的当前物料）。
    2. 从包装成品（根物料）出发，沿 carry 链向下游追踪：
       包装 缝制成品 → 缝制 毛坯 → 染色 白坯/坯布 → 织造 白坯/毛坯/坯布 → 原材料。
    3. 每个工段的折算系数 = 上级 carry 金额 / 本工段 stage_total。
    4. 链路断裂（某工段缺失）→ 该工段跳过，下游工段使用 last_scale 兜底。
    5. carry 行（下一层半成品）**不直接计入成本项目**，它们代表向下游转移的价值；
       下游工段展开后自然覆盖这部分。最终"未追溯半成品" = 完工成本 - 已分类合计，
       反映因数据缺失未能展开到成本项目的金额。

    返回 {物料编码: {成本项目: 折算后金额, ...}, ...}。
    """
    if not io_rows:
        return {}

    rows_by_material = {}
    for r in io_rows:
        code = r.get("物料编码", "")
        if code:
            rows_by_material.setdefault(code, []).append(r)

    io_root_codes = set(rows_by_material.keys())

    result = {}
    for rec in ucs_records:
        root_code = rec.get("物料编码", "")
        cost = safe_float(rec.get("完工成本"))
        if not root_code or root_code not in io_root_codes:
            continue

        chain = _walk_chain(root_code, io_rows, rows_by_material)
        cats = {c: 0.0 for c in IO_COST_CATEGORIES}
        cats["未分类"] = 0.0
        cats["未追溯半成品"] = 0.0

        for stage, scale, stage_rows in chain:
            carry_kw = carry_keywords_for_stage(stage)
            for r in stage_rows:
                amount = safe_float(r.get("金额")) * scale
                element = r.get("核算要素名称", "")
                if element in carry_kw and stage != "原材料":
                    continue
                cat = classify_io_element(element, stage=stage)
                cats[cat] += amount

        total_classified = sum(cats[c] for c in IO_COST_CATEGORIES) + cats.get("未分类", 0)
        untraced = cost - total_classified
        if untraced > 0.01:
            cats["未追溯半成品"] = untraced
        elif untraced < -0.01:
            cats["未追溯半成品"] = 0.0
            cats["数据异常"] = -untraced

        result[root_code] = cats

    return result


def get_chain_trace(root_code, io_rows):
    """返回链路追溯详情，用于可视化展示。

    返回 [{"stage": ..., "materials": [...], "scale": ..., "status": ..., "carry_to": [...], "amount": ...}, ...]
    """
    if not io_rows:
        return []

    rows_by_material = {}
    for r in io_rows:
        code = r.get("物料编码", "")
        if code:
            rows_by_material.setdefault(code, []).append(r)

    # 只处理在投入产出数据中确有对应记录的产品
    if root_code not in rows_by_material:
        return []

    present_stages = {r.get("所属工段", "") for r in io_rows if r.get("所属工段")}
    chain_status = stage_chain_status(present_stages)

    chain = _walk_chain(root_code, io_rows, rows_by_material)
    result = []

    for stage, scale, stage_rows in chain:
        materials = sorted({r.get("物料编码", "") for r in stage_rows if r.get("物料编码")})
        amount = sum(safe_float(r.get("金额")) for r in stage_rows)
        status = chain_status.get(stage, "未取到")
        if status == "已取到" and stage_affected_by_break(stage, chain_status):
            status = "已取到（last_scale 兜底）"

        carry_kw = carry_keywords_for_stage(stage)
        carry_to = sorted({r.get("材料编码", "") for r in stage_rows if r.get("核算要素名称") in carry_kw and r.get("材料编码")})

        result.append({
            "stage": stage,
            "materials": materials,
            "scale": scale,
            "status": status,
            "carry_to": carry_to,
            "amount": amount,
        })

    return result


def print_chain_trace(io_data, ucs_data=None):
    """打印链路追溯图，直观展示每个工段的状态和流向。"""
    if not io_data or not io_data.get("data"):
        return

    rows_by_material = {}
    for r in io_data["data"]:
        code = r.get("物料编码", "")
        if code:
            rows_by_material.setdefault(code, []).append(r)

    # 确定 root codes：如果有 ucs_data 则用其中的物料编码，否则用 io 中包装阶段的所有物料编码
    root_codes = set()
    if ucs_data and ucs_data.get("data"):
        root_codes = {r.get("物料编码", "") for r in ucs_data["data"] if r.get("物料编码")}
    else:
        for r in io_data["data"]:
            if r.get("所属工段") == "包装" and r.get("物料编码"):
                root_codes.add(r.get("物料编码"))

    if not root_codes:
        return

    print("\n  -- 投入产出链路追溯图 --")
    for root_code in sorted(root_codes):
        trace = get_chain_trace(root_code, io_data["data"])
        if not trace:
            continue

        # 取物料名称
        name_map = {}
        for r in io_data["data"]:
            code = r.get("物料编码", "")
            if code and code not in name_map:
                name_map[code] = r.get("物料名称", "")
            code = r.get("材料编码", "")
            if code and code not in name_map:
                name_map[code] = r.get("材料名称", "")

        print(f"\n  包装成品: {root_code} {name_map.get(root_code, '')}")
        for i, node in enumerate(trace):
            stage = node["stage"]
            materials = node["materials"]
            scale = node["scale"]
            status = node["status"]
            carry_to = node["carry_to"]
            amount = node["amount"]

            # 状态图标
            if status == "已取到":
                icon = "✓"
            elif "兜底" in status:
                icon = "⚠"
            elif status == "直接取值":
                icon = "⚠"
            else:
                icon = "✗"

            # 物料显示
            mat_str = ", ".join(f"{m} {name_map.get(m, '')}".strip() for m in materials[:3])
            if len(materials) > 3:
                mat_str += f" 等{len(materials)}个"

            # carry 显示
            carry_str = ""
            if carry_to:
                carry_str = " → " + ", ".join(f"{c} {name_map.get(c, '')}".strip() for c in carry_to[:2])
                if len(carry_to) > 2:
                    carry_str += f" 等{len(carry_to)}个"

            # scale 显示
            if stage == "包装":
                scale_str = ""
            elif "兜底" in status or status == "直接取值":
                scale_str = f" | 折算: {scale:.4f} ({status})"
            else:
                scale_str = f" | 折算: {scale:.4f}"

            indent = "    " + "  " * i
            print(f"{indent}{icon} {stage}: {mat_str}{carry_str} | 金额: {fmt_money(amount)}{scale_str}")

        # 链路断裂提示
        broken = [n for n in trace if n["status"] == "未取到"]
        if broken:
            for node in broken:
                print(f"    ⚠ 链路断裂: {node['stage']} 工段未取到投入产出明细，下游工段使用 last_scale 兜底")





def carry_names_for_stage(stage, io_rows):
    """根据生产阶段和核算要素识别应剔除的下一层成本。"""
    elements = {r.get("核算要素名称", "") for r in io_rows}
    for rule_stage, keywords in STAGE_CARRY_IN_RULES:
        if rule_stage == stage:
            return {e for e in elements if any(k in e for k in keywords)}
    return set()


def split_io_rows_by_stage(rows):
    missing_stage = [r for r in rows if not r.get("所属工段")]
    if missing_stage:
        raise SystemExit("投入产出数据缺少生产阶段字段，无法按最新口径分析")

    stage_values = sorted(
        {r.get("所属工段", "") for r in rows if r.get("所属工段")},
        key=lambda s: STAGE_ORDER.get(s, 50),
    )
    return [(stage, [r for r in rows if r.get("所属工段") == stage]) for stage in stage_values]


def stage_chain_status(present_stages):
    """根据已取到的工段判断每个工段的取数状态：已取到/直接取值/未取到。

    直接取值仅由"直接父级工段未取到"触发，不级联到更下游。
    染色未取到 → 织造 直接取值；织造未取到 → 原材料 直接取值。
    """
    statuses = {}
    for stage in EXPECTED_STAGES:
        if stage in present_stages:
            statuses[stage] = "已取到"
        else:
            statuses[stage] = "未取到"
    for stage, parent in CHAIN_PARENT.items():
        if statuses.get(stage) == "已取到" and statuses.get(parent) == "未取到":
            statuses[stage] = "直接取值"
    return statuses


def stage_affected_by_break(stage, chain_status):
    """本工段是否受链路断裂影响：状态为直接取值，或上游任一父级未取到。"""
    if chain_status.get(stage) == "直接取值":
        return True
    parent = CHAIN_PARENT.get(stage)
    while parent:
        if chain_status.get(parent) == "未取到":
            return True
        parent = CHAIN_PARENT.get(parent)
    return False


def print_stage_net_cost(rows, stage=None):
    carry_names = carry_names_for_stage(stage, rows)
    groups = defaultdict(float)
    for r in rows:
        groups[r.get("核算要素名称", "(未分类)")] += safe_float(r.get("金额"))

    total = sum(groups.values())
    carry_total = sum(amount for name, amount in groups.items() if name in carry_names)
    net_total = total - carry_total

    print("\n  -- 阶段净成本口径 --")
    print(f"  当前生产阶段: {stage}")
    print(f"  投入产出总额: {fmt_money(total)}")
    print(f"  剔除下一层半成品/原料: {fmt_money(carry_total)}")
    print(f"  本阶段净成本: {fmt_money(net_total)}")
    if total:
        print(f"  本阶段净成本占本层投入产出总额: {fmt_pct(net_total / total)}")

    if not groups:
        return
    for name, amount in sorted(groups.items(), key=lambda x: -x[1]):
        if name in carry_names:
            role = "剔除: 下一层成本/追溯入口"
        elif stage == "原材料":
            role = "计入: 最终原料成本（归入'原材料'成本项目）"
        else:
            role = "计入: 本阶段成本"
        print(f"    {name}: {fmt_money(amount)} | {role}")


def print_section(title):
    print()
    print("=" * 70)
    print(f"  {title}")
    print("=" * 70)


def print_order_summary(ucs_data, io_data=None):
    print_section("1. 订单成本结构总览")
    io_cats_by_code = compute_cost_structure_from_io(ucs_data["data"], io_data["data"]) if io_data else {}
    has_io = bool(io_cats_by_code)
    for r in ucs_data["data"]:
        order = r.get("订单号", "")
        material = r.get("物料编码", "")
        material_name = r.get("物料名称", "")
        qty = safe_float(r.get("完工产量"))
        cost = safe_float(r.get("完工成本"))
        revenue = safe_float(r.get("销售无税金额"))
        profit = safe_float(r.get("毛利"))
        margin = safe_float(r.get("毛利率(小数)"))
        unit_cost = safe_float(r.get("单位成本"))
        unit_price = safe_float(r.get("销售单价(无税)"))

        print(f"\n订单号: {order} | 物料: {material} {material_name}")
        print(f"  完工产量: {qty:,.0f} | 单位成本: {unit_cost:.4f} | 销售单价(无税): {unit_price:.4f}")
        print(f"  销售无税金额: {fmt_money(revenue)} | 完工成本: {fmt_money(cost)}")
        print(f"  毛利: {fmt_money(profit)} | 毛利率: {fmt_pct(margin)} [{tier_label(margin)}]")
        calc_profit = revenue - cost
        calc_margin = calc_profit / revenue if revenue else 0
        calc_unit = cost / qty if qty else 0
        print("  -- 校验 --")
        print(f"  计算毛利: {fmt_money(calc_profit)} (差异: {fmt_money(calc_profit - profit)})")
        print(f"  计算毛利率: {fmt_pct(calc_margin)} (差异: {(calc_margin - margin) * 100:.4f}pp)")
        print(f"  计算单位成本: {calc_unit:.4f} (差异: {calc_unit - unit_cost:.4f})")

        print("  -- 成本结构占比 --")
        if has_io and material in io_cats_by_code:
            cats = io_cats_by_code[material]
            total_io = sum(cats[c] for c in IO_COST_CATEGORIES) + cats.get("未追溯半成品", 0) + cats.get("未分类", 0)
            note = ""
            if total_io > cost * 1.5:
                note = "（含 1.0 兜底口径的高估部分，详见投入产出分析）"
            print(f"  [基于投入产出逐级展开]{note}")
            for cat in ["原材料", "辅料", "直接人工", "制造费用", "外协加工费"]:
                amt = cats.get(cat, 0.0)
                ratio = amt / cost if cost else 0
                std, tol = THRESHOLDS[cat]
                if ratio < std - tol:
                    status = "偏低"
                elif ratio > std + tol:
                    status = "偏高"
                else:
                    status = "正常"
                print(
                    f"    {cat}: {fmt_money(amt)} ({fmt_pct(ratio)}) | "
                    f"标准: {std*100:.0f}% | {status}"
                )
            for extra in ("未分类", "未追溯半成品"):
                amt = cats.get(extra, 0.0)
                if abs(amt) > 0.01:
                    print(f"    {extra}: {fmt_money(amt)} ({fmt_pct(amt/cost if cost else 0)})")
        else:
            print("  ⚠ 未提供完整的投入产出数据，无法计算成本结构占比。")
            print("  成本结构分析必须基于投入产出数据逐级展开，请提供完整的投入产出数据后重跑。")

    if has_io:
        print_chain_trace(io_data, ucs_data)


def print_io_summary(io_data):
    print_section("2. 投入产出分析（按生产阶段和核算要素名称分组）")
    stage_groups = split_io_rows_by_stage(io_data["data"])

    print("\n  -- 生产阶段取数状态 --")
    present_stages = {stage for stage, _rows in stage_groups}
    statuses = stage_chain_status(present_stages)
    broken_notes = []
    for stage in EXPECTED_STAGES:
        status = statuses.get(stage, "未取到")
        if status == "已取到":
            print(f"  {stage}: 已取到")
        elif status == "直接取值":
            parent = CHAIN_PARENT.get(stage, "")
            note = f"  {stage}: 直接取值（{parent}工段未取到，stage_total 直接归集）"
            print(note)
            broken_notes.append((stage, parent))
        else:
            print(f"  {stage}: 未取到 | 可能为委外生产或当前报表未返回该阶段")
    if broken_notes:
        missing_stages = {stage for stage, st in statuses.items() if st == "未取到"}
        missing_label = "、".join(
            sorted({parent for _stage, parent in broken_notes if parent in missing_stages},
                   key=lambda s: STAGE_ORDER.get(s, 50))
        ) or "-"
        broken_set = {stage for stage, _ in broken_notes}
        cascade = [s for s in EXPECTED_STAGES if s not in broken_set and stage_affected_by_break(s, statuses)]
        print(
            "  ⚠ 检测到链路断裂：{} 工段未取到，{} 改用直接取值口径（折算系数 1.0）{}。".format(
                missing_label,
                "、".join(s for s, _ in broken_notes),
                f"；下游 {'、'.join(cascade)} 因上游断裂按 1.0 兜底" if cascade else "",
            )
        )
        for stage, parent in broken_notes:
            print(
                f"    - {stage} 因 {parent} 工段未取到，改用直接取值（折算系数 1.0）；"
                f"如需恢复折算链路，请补充查询 {parent} 工段的实际投入产出明细。"
            )

    for stage, rows in stage_groups:
        carry_names = carry_names_for_stage(stage, rows)
        status = statuses.get(stage, "已取到")
        amount = sum(safe_float(r.get("金额")) for r in rows)
        products = {r.get("物料编码", "") for r in rows if r.get("物料编码")}
        if status == "直接取值":
            scale_note = " | 折算系数 1.0（直接取值）"
        elif stage_affected_by_break(stage, statuses):
            scale_note = " | 折算系数 1.0（上游断裂 1.0 兜底）"
        else:
            scale_note = ""
        print(
            f"  {stage}: {status} | 金额 {fmt_money(amount)} | 行数 {len(rows)} | "
            f"当前物料 {len(products)} 个{scale_note}"
        )

    for stage, rows in stage_groups:
        carry_names = carry_names_for_stage(stage, rows)
        print(f"\n  -- {stage}工段投入产出 --")
        products = sorted({r.get("物料编码", "") for r in rows if r.get("物料编码")})
        print(f"  当前物料: {', '.join(products) or '-'}")

        groups = defaultdict(lambda: {"qty": 0.0, "amount": 0.0, "rows": 0, "materials": set()})
        for r in rows:
            key = r.get("核算要素名称", "(未分类)")
            g = groups[key]
            g["qty"] += safe_float(r.get("数量"))
            g["amount"] += safe_float(r.get("金额"))
            g["rows"] += 1
            mat = r.get("材料编码")
            if mat:
                g["materials"].add(mat)

        for key in sorted(groups.keys(), key=lambda k: -groups[k]["amount"]):
            g = groups[key]
            print(
                f"    {key}: 金额 {fmt_money(g['amount'])} | 数量 {g['qty']:,.2f} "
                f"| 行数 {g['rows']} | 涉及材料 {len(g['materials'])} 个"
            )

        print_stage_net_cost(rows, stage=stage)

    sewing = [r for r in io_data["data"] if r.get("核算要素名称") == "缝制成品"]
    if sewing:
        print("\n  -- 缝制成品明细（用于下一层追溯） --")
        for r in sewing:
            print(
                f"    材料编码: {r.get('材料编码','')} | "
                f"材料名称: {r.get('材料名称','')} | "
                f"数量: {safe_float(r.get('数量')):,.0f} | "
                f"单价: {safe_float(r.get('单价')):.6f} | "
                f"金额: {fmt_money(safe_float(r.get('金额')))}"
            )

    print("\n  -- 投入产出异常 --")
    anomalies = []
    for stage, rows in stage_groups:
        carry_names = carry_names_for_stage(stage, rows)
        total = sum(safe_float(r.get("金额")) for r in rows)
        for r in rows:
            amt = safe_float(r.get("金额"))
            prefix = f"{stage}工段"
            if amt < 0:
                anomalies.append(f"    {prefix}负金额: {r.get('材料名称','')} ({r.get('材料编码','')}) = {fmt_money(amt)}")
            qty = safe_float(r.get("数量"))
            if qty < 0:
                anomalies.append(f"    {prefix}负数量: {r.get('材料名称','')} ({r.get('材料编码','')}) = {qty:,.2f}")
            if total and amt / total > 0.30 and r.get("核算要素名称") not in carry_names:
                if stage == "原材料":
                    anomalies.append(
                        f"    {prefix}高占比原料: {r.get('材料名称','')} ({r.get('材料编码','')}) "
                        f"占本工段总额 {fmt_pct(amt/total)}，建议核查单价与耗量"
                    )
                else:
                    anomalies.append(
                        f"    {prefix}高占比: {r.get('材料名称','')} ({r.get('材料编码','')}) "
                        f"占本工段总额 {fmt_pct(amt/total)}"
                    )
            elif total and amt / total > 0.30 and r.get("核算要素名称") in carry_names:
                anomalies.append(
                    f"    {prefix}追溯入口: {r.get('核算要素名称','')} {r.get('材料名称','')} "
                    f"({r.get('材料编码','')}) 占本工段总额 {fmt_pct(amt/total)}，不计入本阶段净成本"
                )
    if anomalies:
        for a in anomalies:
            print(a)
    else:
        print("    无异常")


def print_inventory_analysis(ledger_data):
    print_section("3. 存货价格分析")
    rows = ledger_data["data"]
    if not rows:
        print("  无存货明细数据")
        return

    by_period = defaultdict(lambda: {"qty": 0.0, "amount": 0.0, "rows": 0, "prices": []})
    by_type = defaultdict(lambda: {"qty": 0.0, "amount": 0.0, "rows": 0})
    for r in rows:
        period = r.get("会计期间", "(未知)")
        btype = r.get("单据类型", "(未知)")
        qty = safe_float(r.get("主数量"))
        price = safe_float(r.get("单价"))
        amt = safe_float(r.get("金额"))
        by_period[period]["qty"] += qty
        by_period[period]["amount"] += amt
        by_period[period]["rows"] += 1
        if price > 0:
            by_period[period]["prices"].append(price)
        by_type[btype]["qty"] += qty
        by_type[btype]["amount"] += amt
        by_type[btype]["rows"] += 1

    print("\n  -- 月度加权均价 --")
    print(f"    {'期间':<10} {'入库数量':>14} {'入库金额':>14} {'加权均价':>10} {'最高价':>10} {'最低价':>10} {'行数':>6}")
    monthly_avgs = {}
    for period in sorted(by_period.keys()):
        g = by_period[period]
        if g["qty"] > 0:
            avg = g["amount"] / g["qty"]
            mx = max(g["prices"]) if g["prices"] else 0
            mn = min(g["prices"]) if g["prices"] else 0
            monthly_avgs[period] = avg
            print(
                f"    {period:<10} {g['qty']:>14,.2f} {fmt_money(g['amount']):>14} "
                f"{avg:>10.4f} {mx:>10.4f} {mn:>10.4f} {g['rows']:>6}"
            )

    if len(monthly_avgs) >= 2:
        print("\n  -- 期间环比变化 --")
        periods = sorted(monthly_avgs.keys())
        for i in range(1, len(periods)):
            prev = monthly_avgs[periods[i - 1]]
            curr = monthly_avgs[periods[i]]
            chg = (curr - prev) / prev if prev else 0
            print(f"    {periods[i-1]} -> {periods[i]}: {prev:.4f} -> {curr:.4f} ({chg*100:+.2f}%)")

    print("\n  -- 单据类型分布 --")
    total_amt = sum(g["amount"] for g in by_type.values())
    for btype, g in sorted(by_type.items(), key=lambda x: -x[1]["amount"]):
        ratio = g["amount"] / total_amt if total_amt else 0
        print(
            f"    {btype}: 数量 {g['qty']:,.2f} | 金额 {fmt_money(g['amount'])} "
            f"({fmt_pct(ratio)}) | 行数 {g['rows']}"
        )

    print("\n  -- 异常单据（单价 > 期间均价 50%） --")
    anomalies = []
    for period, avg in monthly_avgs.items():
        threshold = avg * 1.5
        for r in rows:
            if r.get("会计期间") != period:
                continue
            price = safe_float(r.get("单价"))
            if price > threshold and price > 0:
                anomalies.append(
                    f"    {r.get('单据日期','')} {r.get('单据类型','')} "
                    f"数量 {safe_float(r.get('主数量')):,.2f} "
                    f"单价 {price:.4f} 金额 {fmt_money(safe_float(r.get('金额')))} "
                    f"(期间均价 {avg:.4f}, 高 {(price/avg-1)*100:.1f}%)"
                )
    if anomalies:
        for a in anomalies[:20]:
            print(a)
        if len(anomalies) > 20:
            print(f"    ... 共 {len(anomalies)} 条异常，仅显示前 20 条")
    else:
        print("    无异常")


def print_anomaly_check(ucs_data, io_data=None, ledger_data=None):
    print_section("4. 异常识别汇总")

    io_cats_by_code = compute_cost_structure_from_io(ucs_data["data"], io_data["data"]) if io_data else {}
    for r in ucs_data["data"]:
        order = r.get("订单号", "")
        material = r.get("物料编码", "")
        cost = safe_float(r.get("完工成本"))
        margin = safe_float(r.get("毛利率(小数)"))
        print(f"\n  订单 {order}:")
        if margin < 0:
            print(f"    [严重] 亏损订单，毛利率 {fmt_pct(margin)}")
        elif margin < 0.10:
            print(f"    [警告] 低毛利，毛利率 {fmt_pct(margin)}")

        if material in io_cats_by_code:
            cats = io_cats_by_code[material]
            for cat in IO_COST_CATEGORIES:
                amt = cats.get(cat, 0.0)
                ratio = amt / cost if cost else 0
                std, tol = THRESHOLDS[cat]
                if ratio < std - tol:
                    print(f"    [提示] {cat}占比 {fmt_pct(ratio)} 低于标准（标准 {std*100:.0f}% ±{tol*100:.0f}%）")
                elif ratio > std + tol:
                    print(f"    [警告] {cat}占比 {fmt_pct(ratio)} 超阈值（标准 {std*100:.0f}% ±{tol*100:.0f}%）")
        else:
            print("    [提示] 缺少投入产出数据，跳过成本结构异常检查")

        revenue = safe_float(r.get("销售无税金额"))
        qty = safe_float(r.get("完工产量"))
        unit_cost = safe_float(r.get("单位成本"))
        if revenue > 0 and qty > 0:
            calc_unit = cost / qty
            if abs(calc_unit - unit_cost) > 0.01:
                print(f"    [提示] 单位成本不一致: 返回 {unit_cost:.4f} vs 计算 {calc_unit:.4f}")
        if revenue > 0:
            profit = safe_float(r.get("毛利"))
            calc_profit = revenue - cost
            if abs(calc_profit - profit) > 0.01:
                print(f"    [提示] 毛利不一致: 返回 {fmt_money(profit)} vs 计算 {fmt_money(calc_profit)}")


def main():
    parser = argparse.ArgumentParser(description="毛利率明细分析")
    parser.add_argument("ucs", help="单位成本 JSON 文件路径")
    parser.add_argument("io", nargs="?", default=None, help="投入产出 JSON 文件路径（可选）")
    parser.add_argument("ledger", nargs="?", default=None, help="存货明细 JSON 文件路径（可选）")
    parser.add_argument("--summary", action="store_true", help="输出成本结构总览")
    parser.add_argument("--io", dest="show_io", action="store_true", help="输出投入产出分析")
    parser.add_argument("--price", action="store_true", help="输出存货价格分析")
    parser.add_argument("--anomaly", action="store_true", help="输出异常识别汇总")
    parser.add_argument("--all", action="store_true", help="输出全部分析（默认）")
    args = parser.parse_args()

    no_flag = not any([args.summary, args.show_io, args.price, args.anomaly, args.all])
    if no_flag:
        args.all = True

    ucs_data = load_json(args.ucs)

    io_data = None
    if args.io:
        io_data = load_json(args.io)

    ledger_data = None
    if args.ledger:
        ledger_data = load_json(args.ledger)

    print(f"分析数据: 单位成本={args.ucs}", end="")
    if args.io:
        print(f" + 投入产出={args.io}", end="")
    if args.ledger:
        print(f" + 存货明细={args.ledger}", end="")
    print()

    if args.all or args.summary:
        print_order_summary(ucs_data, io_data)
    if (args.all or args.show_io) and io_data:
        print_io_summary(io_data)
    if (args.all or args.price) and ledger_data:
        print_inventory_analysis(ledger_data)
    if args.all or args.anomaly:
        print_anomaly_check(ucs_data, io_data, ledger_data)


if __name__ == "__main__":
    main()

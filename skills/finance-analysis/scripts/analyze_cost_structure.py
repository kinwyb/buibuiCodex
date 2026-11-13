#!/usr/bin/env python3
"""
跨阶段成本项目标准占比分析脚本。

用途：
  按“阶段净成本”口径汇总包装、缝制、染色、织造等阶段的费用。
  每个阶段先剔除下一层产成品/半成品，再把本阶段费用累计到：
  原材料、辅料、直接人工、制造费用、外协加工费。

用法：
  python3 analyze_cost_structure.py 单位成本.json --io 投入产出.json
"""

import argparse
import json
from collections import defaultdict


THRESHOLDS = {
    "原材料": (0.30, 0.05),
    "辅料": (0.10, 0.05),
    "直接人工": (0.35, 0.05),
    "制造费用": (0.20, 0.05),
    "外协加工费": (0.05, 0.05),
}


# 阶段 carry 规则：上游半成品/原料只作为追溯入口，不计入当前工段净成本。
# "原材料"工段没有下一层（carry 列表留空），但仍按 raw 模式归集到"原材料"成本项目。
# 织造工段不再剔除原料/纱线（这些已迁到"原材料"工段），但保留少量防御性关键词。
STAGE_RULES = [
    ("包装", ["缝制成品"], "next"),
    ("缝制", ["毛坯"], "next"),
    ("染色", ["白坯", "坯布"], "next"),
    ("织造", ["白坯", "毛坯", "坯布", "其他原料", "纱线"], "next"),
    ("原材料", [], "raw"),
]


STAGE_ORDER = {"包装": 0, "缝制": 1, "染色": 2, "织造": 3, "原材料": 4}
EXPECTED_STAGES = ["包装", "缝制", "染色", "织造", "原材料"]
# 多链路计算中每个工段的父级（用于判断链路断裂）
CHAIN_PARENT = {"缝制": "包装", "染色": "缝制", "织造": "染色", "原材料": "织造"}


def load_json(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def safe_float(value):
    if value is None or value == "":
        return 0.0
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def fmt_money(value):
    return f"{value:,.2f}"


def fmt_pct(value):
    return f"{value * 100:.2f}%"


def sum_field(rows, field):
    return sum(safe_float(row.get(field)) for row in rows)


def weighted_unit_cost(rows):
    qty = sum_field(rows, "完工产量")
    cost = sum_field(rows, "完工成本")
    return cost / qty if qty else 0.0


def status_for(category, ratio):
    std, tol = THRESHOLDS[category]
    low = std - tol
    high = std + tol
    if ratio < low:
        return "偏低"
    if ratio > high:
        return "偏高"
    return "正常"


def classify_cost(element_name):
    """按核算要素名称归入标准成本项目。"""
    name = element_name or ""
    if any(k in name for k in ["原料", "原材料", "纱线"]):
        return "原材料"
    if "辅料" in name:
        return "辅料"
    if any(k in name for k in ["直接人工", "间接人工", "福利费", "五险一金", "劳务费"]):
        return "直接人工"
    if any(k in name for k in ["折旧", "水费", "电费", "水电费", "修理费", "低值易耗", "汇总其他"]):
        return "制造费用"
    if any(k in name for k in ["加工费", "委外加工费", "外协"]):
        return "外协加工费"
    return "未分类"


def carry_rule_for_stage(stage, rows):
    elements = {r.get("核算要素名称", "") for r in rows}
    for rule_stage, keywords, carry_mode in STAGE_RULES:
        if rule_stage == stage:
            carry_names = {e for e in elements if any(k in e for k in keywords)}
            return carry_names, carry_mode
    return set(), "none"


def summarize_io_rows(path, rows, stage=None):
    if not stage:
        raise SystemExit("投入产出数据缺少生产阶段字段，无法按最新口径分析")
    carry_names, carry_mode = carry_rule_for_stage(stage, rows)

    product_codes = {r.get("物料编码", "") for r in rows if r.get("物料编码")}
    product_names = {r.get("物料名称", "") for r in rows if r.get("物料名称")}

    stage_total = 0.0
    carry_total = 0.0
    category_totals = defaultdict(float)
    category_details = defaultdict(list)
    carry_rows = []
    carry_products = []
    unclassified_rows = []
    negative_rows = []

    for row in rows:
        element = row.get("核算要素名称", "")
        amount = safe_float(row.get("金额"))
        stage_total += amount

        if amount < 0:
            negative_rows.append(row)

        if element in carry_names:
            carry_total += amount
            carry_rows.append(row)
            carry_products.append(row)
            # 织造/原材料工段的原料/纱线是最终原料成本，剔除出工段净成本但纳入原材料汇总。
            if carry_mode == "raw":
                category_totals["原材料"] += amount
                category_details["原材料"].append((stage, element, row, amount))
            continue

        # 原材料工段没有下一层，所有行直接归入"原材料"成本项目，
        # 即使核算要素名称（如棉纱、化纤等）不被 classify_cost 默认规则匹配。
        if stage == "原材料":
            category_totals["原材料"] += amount
            category_details["原材料"].append((stage, element, row, amount))
            continue

        category = classify_cost(element)
        category_totals[category] += amount
        category_details[category].append((stage, element, row, amount))
        if category == "未分类" and abs(amount) > 0.01:
            unclassified_rows.append(row)

    stage_net = stage_total - carry_total
    return {
        "path": path,
        "stage": stage,
        "product_codes": sorted(product_codes),
        "product_names": sorted(product_names),
        "stage_total": stage_total,
        "carry_total": carry_total,
        "carry_mode": carry_mode,
        "stage_net": stage_net,
        "carry_rows": carry_rows,
        "carry_products": carry_products,
        "category_totals": category_totals,
        "category_details": category_details,
        "unclassified_rows": unclassified_rows,
        "negative_rows": negative_rows,
    }


def summarize_io(path, target_material=None):
    raw = load_json(path)
    rows = raw.get("data", [])
    if any(not row.get("所属工段") for row in rows):
        raise SystemExit("投入产出数据缺少生产阶段字段，无法按最新口径分析")

    # 始终按 (工段, 物料编码) 分组，避免多产品混算导致折算率稀释
    groups = defaultdict(list)
    for r in rows:
        key = (r.get("所属工段", ""), r.get("物料编码", ""))
        groups[key].append(r)

    summaries = []
    for (stage, code), stage_rows in sorted(groups.items(), key=lambda kv: (STAGE_ORDER.get(kv[0][0], 50), kv[0][1])):
        summaries.append(summarize_io_rows(path, stage_rows, stage=stage))
    return summaries, raw


def first_value(values):
    values = [v for v in values if v]
    return values[0] if values else ""


def summary_scale(summary, scale_by_code):
    scales = [scale_by_code[code] for code in summary["product_codes"] if code in scale_by_code]
    if not scales:
        return None
    return max(scales)


def carry_keywords_for_stage(stage):
    """返回指定阶段应剔除的下一层半成品关键词集合。"""
    for rule_stage, keywords, _ in STAGE_RULES:
        if rule_stage == stage:
            return set(keywords)
    return set()


# 从前一阶段 carry 到当前阶段的关键词映射（用于链路追踪）
PREV_STAGE_CARRY = {
    "缝制": {"缝制成品"},
    "染色": {"毛坯"},
    "织造": {"白坯", "坯布"},
    "原材料": set(),
}


def compute_last_scale_per_product(root_codes, scale_by_code, io_rows):
    """按每个包装成品沿 carry 链追踪，返回各自的 last_scale（链路断裂前的最后一个有效折算率）。"""
    rows_by_material = {}
    for r in io_rows:
        code = r.get("物料编码", "")
        if code:
            rows_by_material.setdefault(code, []).append(r)

    def _trace(root_code):
        current_materials = {root_code}
        last = 1.0
        for stage in EXPECTED_STAGES[1:]:
            prev_carry_kw = PREV_STAGE_CARRY.get(stage, set())
            next_materials = set()
            for code in current_materials:
                for r in rows_by_material.get(code, []):
                    if r.get("核算要素名称") in prev_carry_kw and r.get("材料编码"):
                        next_materials.add(r.get("材料编码"))

            for code in next_materials:
                if code in scale_by_code and scale_by_code[code] != 1.0:
                    last = scale_by_code[code]

            if not next_materials:
                break
            current_materials = next_materials
        return last

    return {code: _trace(code) for code in root_codes}


def build_material_to_root_map(root_codes, io_rows):
    """建立非包装成品物料编码到所属包装成品的映射（通过名称前缀匹配）。"""
    import re
    root_names = {}
    for r in io_rows:
        code = r.get("物料编码", "")
        if code in root_codes and code not in root_names:
            root_names[code] = r.get("物料名称", "")

    def _prefix(name):
        m = re.match(r"([A-Za-z0-9]+)", name or "")
        return m.group(1) if m else ""

    mat_to_root = {}
    for r in io_rows:
        code = r.get("物料编码", "")
        name = r.get("物料名称", "")
        if not code or code in root_codes or not name:
            continue
        prefix = _prefix(name)
        if not prefix:
            continue
        for root_code, root_name in root_names.items():
            if prefix == _prefix(root_name):
                mat_to_root[code] = root_code
                break
    return mat_to_root


def compute_chain_status(present_stages):
    """多链路取数状态：已取到 / 直接取值 / 未取到。

    直接取值仅由"直接父级工段未取到"触发，不级联到更下游。
    染色未取到 → 织造 直接取值；织造未取到 → 原材料 直接取值；其余保持"已取到"。

    链断裂的下游工段若自身数据存在但缺少父级折算依据，
    会在主流程里通过 effective_scale 兜底为 last_scale，
    这样不会在状态表里把所有下游工段都误标为"直接取值"。
    """
    statuses = {stage: ("已取到" if stage in present_stages else "未取到") for stage in EXPECTED_STAGES}
    for stage, parent in CHAIN_PARENT.items():
        if statuses.get(stage) == "已取到" and statuses.get(parent) == "未取到":
            statuses[stage] = "直接取值"
    return statuses


def effective_scale(summary, scale_by_code, chain_status, last_scale=1.0):
    """根据链路状态给出兜底折算系数：
    - 工段"已取到"且能算出父级折算 → 使用父级折算
    - 工段"已取到"但其直接父级"未取到"导致无父级折算 → last_scale（直接取值）
    - 工段"直接取值" → last_scale
    - 任何上游父级"未取到"导致级联无折算 → last_scale
    """
    raw = summary_scale(summary, scale_by_code)
    if raw is not None and raw > 0:
        return raw
    stage = summary["stage"]
    status = chain_status.get(stage, "已取到")
    if status == "直接取值":
        return last_scale
    # 级联兜底：上游任一父级未取到导致本工段拿不到父级折算，按 last_scale 计入
    parent = CHAIN_PARENT.get(stage)
    while parent:
        if chain_status.get(parent) == "未取到":
            return last_scale
        parent = CHAIN_PARENT.get(parent)
    return raw if raw is not None else 0.0


def filter_relevant_summaries(summaries, target_codes, scale_by_code, target_name=""):
    """过滤与 target 相关的 summary。

    优先通过编码匹配（target_codes 或 scale_by_code 中的编码）。
    对于链路断裂后的阶段（织造/原材料），额外通过物料名称前缀匹配。
    """
    prefix = ""
    if target_name:
        import re
        match = re.match(r"([A-Za-z0-9]+)", target_name)
        prefix = match.group(1) if match else ""

    relevant = []
    for summary in summaries:
        # 通过编码关联
        by_code = any(
            code in target_codes or code in scale_by_code
            for code in summary["product_codes"]
        )
        if by_code:
            relevant.append(summary)
            continue

        # 链路断裂后的阶段，尝试通过名称前缀匹配
        if prefix and summary["stage"] in ["织造", "原材料"]:
            for name in summary["product_names"]:
                if prefix in name:
                    relevant.append(summary)
                    break

    return relevant


def compute_scales(summaries, root_codes, target_codes=None):
    """按上层实际领用金额，把子阶段投入产出折算到订单口径。"""
    by_code = {}
    for summary in summaries:
        for code in summary["product_codes"]:
            by_code[code] = summary

    scale_by_code = {code: 1.0 for code in root_codes}
    carry_amount_by_code = defaultdict(float)

    # 只处理与 target 相关的链路
    relevant_codes = set(root_codes)
    if target_codes:
        for _ in range(len(summaries) + 2):
            new_codes = set()
            for code in relevant_codes:
                summary = by_code.get(code)
                if summary:
                    for row in summary["carry_rows"]:
                        material_code = row.get("材料编码", "")
                        if material_code:
                            new_codes.add(material_code)
            old_size = len(relevant_codes)
            relevant_codes.update(new_codes)
            if len(relevant_codes) == old_size:
                break

    for _ in range(len(summaries) + 2):
        changed = False
        carry_amount_by_code.clear()

        for summary in summaries:
            if target_codes:
                if not any(code in relevant_codes for code in summary["product_codes"]):
                    continue

            scale = summary_scale(summary, scale_by_code)
            if scale is None:
                continue
            for row in summary["carry_rows"]:
                material_code = row.get("材料编码", "")
                if material_code:
                    carry_amount_by_code[material_code] += safe_float(row.get("金额")) * scale

        for code, carry_amount in carry_amount_by_code.items():
            child = by_code.get(code)
            if not child or not child["stage_total"]:
                continue
            new_scale = carry_amount / child["stage_total"]
            if code not in scale_by_code or abs(scale_by_code[code] - new_scale) > 0.000001:
                scale_by_code[code] = new_scale
                changed = True

        if not changed:
            break

    return scale_by_code, carry_amount_by_code


def build_summary_by_code(summaries):
    by_code = {}
    for summary in summaries:
        for code in summary["product_codes"]:
            by_code[code] = summary
    return by_code


def product_label(codes, names):
    code = first_value(codes)
    name = first_value(names)
    if code and name:
        return f"{code} {name}"
    return code or name or "-"


def main():
    parser = argparse.ArgumentParser(description="跨阶段成本项目标准占比分析")
    parser.add_argument("ucs", help="单位成本 JSON 文件")
    parser.add_argument("--io", action="append", default=[], help="投入产出 JSON 文件，可重复传入")
    parser.add_argument("--material", help="只分析指定包装成品编码")
    args = parser.parse_args()

    ucs = load_json(args.ucs)
    if not ucs.get("data"):
        raise SystemExit("单位成本数据为空")
    orders = ucs["data"]
    if args.material:
        orders = [row for row in orders if row.get("物料编码") == args.material]
        if not orders:
            raise SystemExit(f"单位成本中未找到包装成品编码: {args.material}")
    order = orders[0]
    total_qty = sum_field(orders, "完工产量")
    total_cost = sum_field(orders, "完工成本")
    revenue = sum_field(orders, "销售无税金额")
    profit = sum_field(orders, "毛利")
    margin = profit / revenue if revenue else 0.0
    root_codes = {row.get("物料编码", "") for row in orders if row.get("物料编码")}

    print("## 成本项目标准占比分析")
    print()
    print(f"订单号: {order.get('订单号', '')}")
    print(f"包装成品数: {len(orders)} | 完工产量: {total_qty:,.2f}")
    print(f"完工成本: {fmt_money(total_cost)} | 销售无税金额: {fmt_money(revenue)} | 毛利: {fmt_money(profit)} | 毛利率: {fmt_pct(margin)}")
    print()

    print("### 单位成本明细")
    print()
    print("| 包装成品编码 | 包装成品名称 | 完工产量 | 单位成本 | 销售单价 | 销售无税金额 | 毛利率 |")
    print("|--------------|--------------|---------:|---------:|---------:|-------------:|-------:|")
    for row in orders:
        print(
            f"| {row.get('物料编码', '')} | {row.get('物料名称', '')} | "
            f"{safe_float(row.get('完工产量')):,.2f} | {safe_float(row.get('单位成本')):.4f} | "
            f"{safe_float(row.get('销售单价(无税)')):.4f} | {fmt_money(safe_float(row.get('销售无税金额')))} | "
            f"{row.get('毛利率', '')} |"
        )
    print(f"| 合计 | - | {total_qty:,.2f} | {weighted_unit_cost(orders):.4f} | - | {fmt_money(revenue)} | {fmt_pct(margin)} |")
    print()

    summaries = []
    io_raws = []
    for path in args.io:
        s, raw = summarize_io(path, target_material=args.material)
        summaries.extend(s)
        io_raws.append(raw)
    summaries.sort(key=lambda s: (STAGE_ORDER.get(s["stage"], 50), first_value(s["product_codes"])))
    summary_by_code = build_summary_by_code(summaries)
    target_codes = root_codes if args.material else None
    scale_by_code, _carry_amount_by_code = compute_scales(summaries, root_codes, target_codes=target_codes)
    target_name = order.get("物料名称", "") if args.material else ""
    if args.material:
        relevant_summaries = filter_relevant_summaries(summaries, root_codes, scale_by_code, target_name)
    else:
        relevant_summaries = summaries
    product_codes_with_io = set()
    for summary in relevant_summaries:
        product_codes_with_io.update(summary["product_codes"])

    # 多链路取数状态：仅直接父级为"未取到"的工段标"直接取值"，不级联。
    # 级联兜底由 effective_scale 处理：上游任一父级"未取到"导致折算不出时取 last_scale。
    present_stages = {summary["stage"] for summary in summaries}
    chain_status = compute_chain_status(present_stages)
    broken_stages = {stage for stage, st in chain_status.items() if st == "直接取值"}
    # 真正受链路断裂影响（需 last_scale 兜底）的工段集合：直接取值 + 级联下游
    affected_stages = set(broken_stages)
    for stage in EXPECTED_STAGES:
        if stage in affected_stages:
            continue
        if chain_status.get(stage) != "已取到":
            continue
        parent = CHAIN_PARENT.get(stage)
        while parent:
            if chain_status.get(parent) == "未取到":
                affected_stages.add(stage)
                break
            parent = CHAIN_PARENT.get(parent)

    # 按每个产品分别计算 last_scale（链路断裂前的最后一个有效折算率）
    all_io_rows = []
    for raw in io_raws:
        all_io_rows.extend(raw.get("data", []))
    last_scale_by_product = compute_last_scale_per_product(root_codes, scale_by_code, all_io_rows)
    mat_to_root = build_material_to_root_map(root_codes, all_io_rows)

    def get_last_scale_for_summary(summary):
        """为当前 summary 取对应产品的 last_scale（兜底用）。"""
        stage = summary["stage"]
        if stage not in affected_stages and stage not in broken_stages:
            return 1.0
        for code in summary["product_codes"]:
            # 本身就是包装成品
            if code in last_scale_by_product:
                return last_scale_by_product[code]
            # 通过名称前缀映射到包装成品
            root = mat_to_root.get(code)
            if root and root in last_scale_by_product:
                return last_scale_by_product[root]
        # 兜底：返回第一个可用值（通常不应走到这里）
        return next(iter(last_scale_by_product.values()), 1.0)

    print("### 生产阶段取数状态")
    print()
    print("| 生产阶段 | 查询结果 | 折算口径 | 说明 |")
    print("|----------|----------|----------|------|")
    for stage in EXPECTED_STAGES:
        status = chain_status.get(stage, "未取到")
        if status == "已取到":
            stage_rows = [summary for summary in relevant_summaries if summary["stage"] == stage]
            amount = 0.0
            for summary in stage_rows:
                ls = get_last_scale_for_summary(summary) if stage in affected_stages else 1.0
                amount += summary["stage_total"] * effective_scale(summary, scale_by_code, chain_status, ls)
            has_affected = any(summary["stage"] == stage and summary["stage"] in affected_stages for summary in relevant_summaries)
            if has_affected:
                scales_display = ", ".join(f"{last_scale_by_product.get(rc, 1.0):.4f}" for rc in sorted(root_codes))
                scale_kind = f"直接取值（上游缺失，各产品 last_scale 分别为 {scales_display}）"
            else:
                scale_kind = "父级折算"
            print(f"| {stage} | 已取到 | {scale_kind} | 折算后投入产出总额 {fmt_money(amount)} |")
        elif status == "直接取值":
            stage_rows = [summary for summary in relevant_summaries if summary["stage"] == stage]
            amount = 0.0
            for summary in stage_rows:
                amount += summary["stage_total"] * get_last_scale_for_summary(summary)
            parent = CHAIN_PARENT.get(stage, "")
            scales_display = ", ".join(f"{last_scale_by_product.get(rc, 1.0):.4f}" for rc in sorted(root_codes))
            print(
                f"| {stage} | 直接取值 | 各产品折算系数 {scales_display} | {parent}工段未取到，"
                f"本工段直接使用 stage_total × 各产品 last_scale "
                f"({fmt_money(amount)})；如需恢复折算链路，请补充查询 {parent} 工段实际投入产出明细 |"
            )
        else:
            print(f"| {stage} | 未取到 | - | 可能为委外生产或当前报表未返回该阶段，无法计算该阶段净成本 |")
    if broken_stages:
        missing_stages = {stage for stage, st in chain_status.items() if st == "未取到"}
        missing_label = "、".join(sorted(missing_stages, key=lambda s: STAGE_ORDER.get(s, 50))) or "-"
        broken_label = "、".join(sorted(broken_stages, key=lambda s: STAGE_ORDER.get(s, 50)))
        cascade_label = "、".join(sorted(affected_stages - broken_stages, key=lambda s: STAGE_ORDER.get(s, 50)))
        cascade_scales = ", ".join(f"{last_scale_by_product.get(rc, 1.0):.4f}" for rc in sorted(root_codes))
        cascade_msg = f"，下游 {cascade_label} 分别按 {cascade_scales} 兜底" if cascade_label else ""
        print()
        print(
            f"⚠ 检测到链路断裂：{missing_label} 工段未取到，导致 {broken_label} "
            f"改用直接取值口径（各产品折算系数分别为 {cascade_scales}）{cascade_msg}。"
            f"如需恢复折算链路，请补充查询 {missing_label} 工段的实际投入产出明细。"
        )
    print()

    category_totals = defaultdict(float)
    category_details = defaultdict(list)
    unresolved_carry = []
    negative_rows = []
    unclassified_rows = []

    print("### 生产阶段净成本")
    print()
    print("| 生产阶段 | 当前物料 | 折算系数 | 本层投入产出总额 | 剔除下一层 | 本阶段净成本 | 占完工成本 |")
    print("|------|----------|---------:|----------------:|------------:|------------:|----------:|")
    for summary in relevant_summaries:
        stage = summary["stage"]
        ls = get_last_scale_for_summary(summary)
        if stage in broken_stages:
            scale = ls
            scale_note = f"{ls:.4f}*"
        elif stage in affected_stages:
            scale = ls
            scale_note = f"{ls:.4f}†"
        else:
            scale = effective_scale(summary, scale_by_code, chain_status, ls)
            scale_note = f"{scale:.4f}"

        for category, amount in summary["category_totals"].items():
            category_totals[category] += amount * scale
        for category, details in summary["category_details"].items():
            for stage, element, row, amount in details:
                category_details[category].append((stage, element, row, amount * scale))
        for row in summary["negative_rows"]:
            negative_rows.append((row, scale))
        for row in summary["unclassified_rows"]:
            unclassified_rows.append((row, scale))

        current = ",".join(summary["product_codes"][:2])
        stage_total = summary["stage_total"] * scale
        carry_total = summary["carry_total"] * scale
        stage_net = summary["stage_net"] * scale
        if stage in broken_stages:
            scale_label = f"{scale_note} (直接取值，因父级未取到)"
        elif stage in affected_stages:
            scale_label = f"{scale_note} ({ls:.4f} 兜底，上游未取到)"
        else:
            scale_label = scale_note
        print(
            f"| {summary['stage']} | {current} | {scale_label} | {fmt_money(stage_total)} | "
            f"{fmt_money(carry_total)} | {fmt_money(stage_net)} | "
            f"{fmt_pct(stage_net / total_cost if total_cost else 0)} |"
        )

        for row in summary["carry_rows"]:
            material_code = row.get("材料编码", "")
            if summary.get("carry_mode") == "raw":
                continue
            if material_code and material_code not in product_codes_with_io:
                unresolved_carry.append((row, scale))

    print()
    print("### 生产阶段成品单价（含子工段）")
    print()
    print("| 生产阶段 | 上级产品 | 成品编码 | 成品名称 | 数量 | 上级领用单价 | 上级领用金额 | 本阶段单价 | 本阶段金额 | 差异金额 | 取价口径 |")
    print("|------|----------|----------|----------|-----:|-----------:|-----------:|-----------:|-----------:|---------:|----------|")
    for row in orders:
        print(
            f"| 包装 | - | {row.get('物料编码', '')} | {row.get('物料名称', '')} | "
            f"{safe_float(row.get('完工产量')):,.2f} | - | - | {safe_float(row.get('单位成本')):.4f} | "
            f"{fmt_money(safe_float(row.get('完工成本')))} | - | 单位成本 |"
        )
    for summary in relevant_summaries:
        parent = product_label(summary["product_codes"], summary["product_names"])
        for row in summary["carry_products"]:
            stage_name = row.get("核算要素名称", "")
            material_code = row.get("材料编码", "")
            qty = safe_float(row.get("数量"))
            carry_unit = safe_float(row.get("单价"))
            carry_amount = safe_float(row.get("金额"))
            child = summary_by_code.get(material_code)
            if child and qty:
                stage_amount = child["stage_total"]
                stage_unit = stage_amount / qty
                diff_amount = stage_amount - carry_amount
                pricing_basis = "本阶段明细"
            else:
                stage_amount = carry_amount
                stage_unit = carry_unit
                diff_amount = 0.0
                pricing_basis = "上级领用"
            print(
                f"| {stage_name} | {parent} | {material_code} | {row.get('材料名称', '')} | "
                f"{qty:,.2f} | {carry_unit:.4f} | {fmt_money(carry_amount)} | "
                f"{stage_unit:.4f} | {fmt_money(stage_amount)} | {fmt_money(diff_amount)} | {pricing_basis} |"
            )

    print()
    print("### 成本项目标准占比")
    print()
    print("| 成本项目 | 金额 | 占完工成本 | 标准占比 | 正常区间 | 判断 |")
    print("|----------|-----:|----------:|----------:|----------|------|")
    for category in ["原材料", "辅料", "直接人工", "制造费用", "外协加工费"]:
        amount = category_totals[category]
        ratio = amount / total_cost if total_cost else 0.0
        std, tol = THRESHOLDS[category]
        print(
            f"| {category} | {fmt_money(amount)} | {fmt_pct(ratio)} | {fmt_pct(std)} | "
            f"{fmt_pct(std - tol)} - {fmt_pct(std + tol)} | {status_for(category, ratio)} |"
        )

    unclassified = category_totals.get("未分类", 0.0)
    if abs(unclassified) > 0.01:
        ratio = unclassified / total_cost if total_cost else 0.0
        print(f"| 未分类 | {fmt_money(unclassified)} | {fmt_pct(ratio)} | - | - | 待确认 |")

    # 未追溯半成品 = 完工成本 - 已分类成本项目合计（确保占比闭合100%）
    resolved_total = sum(category_totals[c] for c in ["原材料", "辅料", "直接人工", "制造费用", "外协加工费"])
    unresolved_total = max(0.0, total_cost - resolved_total - unclassified)
    if unresolved_total > 0.01 or unresolved_carry:
        ratio = unresolved_total / total_cost if total_cost else 0.0
        print(f"| 未追溯半成品 | {fmt_money(unresolved_total)} | {fmt_pct(ratio)} | - | - | 缺失生产阶段 |")

    print()
    print("### 成本项目明细来源")
    for category in ["原材料", "辅料", "直接人工", "制造费用", "外协加工费", "未分类"]:
        details = category_details.get(category, [])
        if not details:
            continue
        print(f"\n#### {category}")
        grouped = defaultdict(float)
        for stage, element, _row, amount in details:
            grouped[(stage, element)] += amount
        for (stage, element), amount in sorted(grouped.items(), key=lambda item: -abs(item[1])):
            print(f"- {stage} / {element}: {fmt_money(amount)}")

    if unresolved_carry:
        print("\n### 未追溯半成品")
        for row, scale in unresolved_carry:
            print(
                f"- {row.get('核算要素名称', '')}: {row.get('材料编码', '')} "
                f"{row.get('材料名称', '')}，折算金额 {fmt_money(safe_float(row.get('金额')) * scale)}。"
                "该项未纳入五大成本项目，建议核查对应阶段是否委外生产或报表未返回该阶段。"
            )

    if negative_rows:
        print("\n### 负数/冲销记录")
        for row, scale in negative_rows[:20]:
            print(
                f"- {row.get('核算要素名称', '')}: {row.get('材料编码', '')} "
                f"{row.get('材料名称', '')}，折算金额 {fmt_money(safe_float(row.get('金额')) * scale)}，数量 {row.get('数量', '')}"
            )
        if len(negative_rows) > 20:
            print(f"- 其余 {len(negative_rows) - 20} 条未展示")


if __name__ == "__main__":
    main()

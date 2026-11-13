#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
生产效率柱状图生成脚本

生成两类柱状图：
1. 大部门对比图：X轴为大部门，每组柱子为不同日期
2. 部门对比图：X轴为具体部门，每组柱子为不同日期

这样可以直观对比：
- 同一部门在不同日期的效率变化
- 不同部门之间的效率差异

用法:
  python3 efficiency_chart.py --input <json文件> --output-dir <输出目录>
  cat data.json | python3 efficiency_chart.py --output-dir <输出目录>

参数:
  --input, -i       输入 JSON 文件路径（不指定则从 stdin 读取）
  --output-dir, -o  输出目录路径（必填）
  --prefix, -p      文件名前缀（默认: efficiency）
  --dpi             图片 DPI（默认: 150）
"""

import argparse
import json
import os
import re
import sys
import logging
from collections import defaultdict

# 设置 matplotlib 后端
import matplotlib
matplotlib.use('Agg')

# 配置中文字体
matplotlib.rcParams['font.sans-serif'] = ['Heiti SC', 'Hiragino Sans GB', 'Songti SC', 'Arial Unicode MS']
matplotlib.rcParams['axes.unicode_minus'] = False
# 忽略 Matplotlib 的 font 管理器弹出的警告
logging.getLogger('matplotlib.font_manager').setLevel(logging.ERROR)
# 或者忽略常规警告
# warnings.filterwarnings('ignore', category=UserWarning, module='matplotlib')

import matplotlib.pyplot as plt
import numpy as np

# ===================== 字体配置 =====================
# 1. 明确指定字体家族为无衬线字体 (sans-serif)
plt.rcParams['font.family'] = 'sans-serif'

# 2. 定义无衬线字体的优先级列表（把服务器上已有的 Noto Sans 放在前面）
plt.rcParams['font.sans-serif'] = [
    'Noto Sans CJK SC',       # Linux/服务器常见的中文字体（你当前系统已验证可用）
    'Source Han Sans CN',     # 思源黑体
    'SimHei',                 # Windows 常用的黑体
    'Heiti SC',               # Mac 常用的黑体
    'Arial Unicode MS'        # 通用兜底字体
]

# 3. 正常显示负号
plt.rcParams['axes.unicode_minus'] = False

def parse_efficiency(val):
    """解析效率值，支持 '110.00%' 或 '110.00' 格式"""
    if isinstance(val, (int, float)):
        return float(val)
    if isinstance(val, str):
        val = val.strip().rstrip('%')
        try:
            return float(val)
        except ValueError:
            return 0.0
    return 0.0


def parse_float(val):
    """解析浮点数"""
    if isinstance(val, (int, float)):
        return float(val)
    if isinstance(val, str):
        try:
            return float(val.strip())
        except ValueError:
            return 0.0
    return 0.0


def sort_dept_key(dept_name):
    """
    部门名称排序键：按中文拼音/笔画自然排序。
    提取数字部分用于排序，使 "缝制一组" < "缝制二组" < "缝制十组"。
    支持中文数字后带 "组" 等后缀的情况。
    """
    cn_num_map = {
        '一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
        '六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
        '十一': 11, '十二': 12, '十三': 13, '十四': 14, '十五': 15,
        '十六': 16, '十七': 17, '十八': 18, '十九': 19, '二十': 20,
    }

    # 尝试提取中文数字（允许后面带"组"等字）
    # 按长度从长到短匹配，避免 "十一" 被匹配成 "一"
    for cn, num in sorted(cn_num_map.items(), key=lambda x: -len(x[0])):
        # 查找中文数字在字符串中的任意位置
        idx = dept_name.find(cn)
        if idx >= 0:
            # 检查是否是更长数字的一部分（如"十一"后面的"一"）
            is_longer = False
            for other_cn in cn_num_map.keys():
                if len(other_cn) > len(cn) and other_cn.startswith(cn):
                    if (idx + len(other_cn) <= len(dept_name) and
                        dept_name[idx:idx+len(other_cn)] == other_cn):
                        is_longer = True
                        break
            if not is_longer:
                prefix = dept_name[:idx]
                return (prefix, num)

    # 尝试提取末尾阿拉伯数字
    m = re.search(r'(\d+)\s*$', dept_name)
    if m:
        prefix = dept_name[:m.start()]
        return (prefix, int(m.group(1)))

    return (dept_name, 0)


COLORS_BIG_DEPT = {
    '成衣一部': '#4E79A7',
    '成衣二部': '#F28E2B',
    '成衣三部': '#59A14F',
    '其他':   '#B07AA1',
}

COLORS_PALETTE = [
    '#4E79A7', '#F28E2B', '#59A14F', '#E15759', '#76B7B2',
    '#EDC948', '#B07AA1', '#FF9DA7', '#9C755F', '#BAB0AC',
    '#86BCB6', '#8CD17D', '#F1CE63', '#B6992D', '#499894',
]


def gen_big_dept_chart(data, output_dir, prefix, dpi):
    """
    生成大部门对比柱状图。
    X轴为大部门，每组柱子为不同日期，便于对比同一部门在不同日期的效率变化。
    返回 (图片路径, 描述文本) 或 None。
    """
    big_dept_rows = [r for r in data if r.get('汇总类型') == '大部门汇总']
    if not big_dept_rows:
        return None

    # 按日期分组
    dates = sorted(set(r.get('工作日期', '') for r in big_dept_rows))
    big_depts = sorted(
        set(r.get('大部门', '') for r in big_dept_rows if r.get('大部门')),
        key=sort_dept_key
    )

    if not big_depts or not dates:
        return None

    # 构建数据矩阵: dates x big_depts (每个日期对应所有大部门的数据)
    matrix = {}
    count_matrix = {}
    for d in dates:
        matrix[d] = []
        count_matrix[d] = []
        for bd in big_depts:
            row = next((r for r in big_dept_rows if r.get('大部门') == bd and r.get('工作日期') == d), None)
            if row:
                matrix[d].append(parse_efficiency(row.get('效率', '0')))
                count_matrix[d].append(row.get('人数', ''))
            else:
                matrix[d].append(0)
                count_matrix[d].append('')

    # 绘图
    n_dates = len(dates)
    n_depts = len(big_depts)
    fig_width = max(8, n_depts * n_dates * 0.8 + 3)
    fig, ax = plt.subplots(figsize=(fig_width, 6))

    x = np.arange(n_depts)
    bar_width = 0.8 / n_dates

    for i, d in enumerate(dates):
        offset = (i - n_dates / 2 + 0.5) * bar_width
        color = COLORS_PALETTE[i % len(COLORS_PALETTE)]
        date_label = d.replace('-', '/')
        bars = ax.bar(x + offset, matrix[d], bar_width, label=date_label, color=color, edgecolor='white', linewidth=0.5)

        # 在柱顶标注效率值和人数
        for bar, val, cnt in zip(bars, matrix[d], count_matrix[d]):
            label_text = f'{val:.1f}%'
            if cnt:
                label_text += f'\n({cnt}人)'
            ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                    label_text, ha='center', va='bottom', fontsize=8, fontweight='bold')

    # X轴为部门名称
    ax.set_xticks(x)
    ax.set_xticklabels(big_depts, fontsize=11)

    ax.axhline(y=100, color='gray', linestyle='--', linewidth=1, alpha=0.7, label='100%基准线')

    ax.set_ylabel('效率 (%)', fontsize=12)
    date_range = f'{dates[0]} ~ {dates[-1]}' if len(dates) > 1 else dates[0]
    ax.set_title(f'大部门生产效率对比（{date_range}）', fontsize=14, fontweight='bold', pad=15)
    ax.legend(loc='upper left', fontsize=10, title='日期')
    all_vals = [v for vs in matrix.values() for v in vs if v > 0]
    ax.set_ylim(0, max(max(all_vals) * 1.3, 120) if all_vals else 120)
    ax.grid(axis='y', alpha=0.3)
    ax.spines['top'].set_visible(False)
    ax.spines['right'].set_visible(False)

    plt.tight_layout()
    filepath = os.path.join(output_dir, f'{prefix}_big_dept_comparison.png')
    fig.savefig(filepath, dpi=dpi, bbox_inches='tight')
    plt.close(fig)

    # 生成描述
    desc_parts = [f'大部门生产效率对比柱状图，日期范围 {date_range}']
    for idx, bd in enumerate(big_depts):
        vals = [f'{matrix[d][idx]:.1f}%' for d in dates if matrix[d][idx] > 0]
        if vals:
            desc_parts.append(f'{bd} 效率: {", ".join(vals)}')

    return filepath, '。'.join(desc_parts)


def gen_dept_chart(data, output_dir, prefix, dpi, big_dept_name):
    """
    生成某大部门下各部门的对比柱状图。
    X轴为具体部门，每组柱子为不同日期，便于对比同一部门在不同日期的效率变化。
    返回 (图片路径, 描述文本) 或 None。
    """
    dept_rows = [
        r for r in data
        if r.get('汇总类型') == '部门汇总' and r.get('大部门') == big_dept_name
    ]
    if not dept_rows:
        return None

    dates = sorted(set(r.get('工作日期', '') for r in dept_rows))
    depts = sorted(
        set(r.get('部门', '') for r in dept_rows if r.get('部门')),
        key=sort_dept_key
    )

    if not depts or not dates:
        return None

    # 构建数据矩阵: dates x depts (每个日期对应所有部门的数据)
    matrix = {}
    count_matrix = {}
    for d in dates:
        matrix[d] = []
        count_matrix[d] = []
        for dept in depts:
            row = next((r for r in dept_rows if r.get('部门') == dept and r.get('工作日期') == d), None)
            if row:
                matrix[d].append(parse_efficiency(row.get('效率', '0')))
                count_matrix[d].append(row.get('人数', ''))
            else:
                matrix[d].append(0)
                count_matrix[d].append('')

    # 绘图
    n_dates = len(dates)
    n_depts = len(depts)
    fig_width = max(10, n_depts * n_dates * 0.7 + 4)
    fig_height = max(6, n_depts * 0.4 + 3)
    fig, ax = plt.subplots(figsize=(fig_width, fig_height))

    x = np.arange(n_depts)
    bar_width = 0.8 / n_dates

    for i, d in enumerate(dates):
        offset = (i - n_dates / 2 + 0.5) * bar_width
        color = COLORS_PALETTE[i % len(COLORS_PALETTE)]
        date_label = d.replace('-', '/')
        bars = ax.bar(x + offset, matrix[d], bar_width, label=date_label, color=color, edgecolor='white', linewidth=0.5)

        # 在柱顶标注效率值
        for bar, val in zip(bars, matrix[d]):
            if val > 0:
                ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                        f'{val:.1f}%', ha='center', va='bottom', fontsize=7)

    # X轴为部门名称
    ax.set_xticks(x)
    # 部门名称较长时旋转显示
    if n_depts > 5 or any(len(d) > 6 for d in depts):
        ax.set_xticklabels(depts, rotation=30, ha='right', fontsize=9)
    else:
        ax.set_xticklabels(depts, fontsize=10)

    ax.axhline(y=100, color='gray', linestyle='--', linewidth=1, alpha=0.7)

    ax.set_ylabel('效率 (%)', fontsize=12)
    date_range = f'{dates[0]} ~ {dates[-1]}' if len(dates) > 1 else dates[0]
    ax.set_title(f'{big_dept_name} 各部门生产效率对比（{date_range}）', fontsize=14, fontweight='bold', pad=15)

    # 图例为日期，移到左上角（与大部门统计图一致）
    ncol = min(n_dates, 3)
    ax.legend(loc='upper left', fontsize=9, ncol=ncol, title='日期')

    all_vals = [v for vs in matrix.values() for v in vs if v > 0]
    ax.set_ylim(0, max(max(all_vals) * 1.3, 120) if all_vals else 120)
    ax.grid(axis='y', alpha=0.3)
    ax.spines['top'].set_visible(False)
    ax.spines['right'].set_visible(False)

    plt.tight_layout()
    # 文件名中大部门名称转拼音简写
    dept_slug = big_dept_name.replace(' ', '_')
    filepath = os.path.join(output_dir, f'{prefix}_dept_{dept_slug}.png')
    fig.savefig(filepath, dpi=dpi, bbox_inches='tight')
    plt.close(fig)

    # 生成描述
    desc_parts = [f'{big_dept_name}各部门生产效率对比柱状图，日期范围 {date_range}']
    desc_parts.append(f'共 {n_depts} 个部门')
    # 找最高和最低（按平均效率）
    avg_effs = {}
    for idx, dept in enumerate(depts):
        vals = [matrix[d][idx] for d in dates if matrix[d][idx] > 0]
        if vals:
            avg_effs[dept] = sum(vals) / len(vals)
    if avg_effs:
        best = max(avg_effs, key=avg_effs.get)
        worst = min(avg_effs, key=avg_effs.get)
        desc_parts.append(f'平均效率最高: {best}({avg_effs[best]:.1f}%)')
        desc_parts.append(f'平均效率最低: {worst}({avg_effs[worst]:.1f}%)')

    return filepath, '，'.join(desc_parts)


def draw_grouped_bar_chart(ax, categories, dates, matrix, count_matrix, title, show_count=False):
    n_dates = len(dates)
    x = np.arange(len(categories))
    bar_width = 0.8 / n_dates

    for i, d in enumerate(dates):
        offset = (i - n_dates / 2 + 0.5) * bar_width
        color = COLORS_PALETTE[i % len(COLORS_PALETTE)]
        date_label = d.replace('-', '/')
        bars = ax.bar(x + offset, matrix[d], bar_width, label=date_label, color=color, edgecolor='white', linewidth=0.5)

        for idx, (bar, val) in enumerate(zip(bars, matrix[d])):
            if val <= 0:
                continue
            label_text = f'{val:.1f}%'
            if show_count and count_matrix and count_matrix[d][idx]:
                label_text += f'\n({count_matrix[d][idx]}人)'
            ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                    label_text, ha='center', va='bottom', fontsize=7)

    ax.set_xticks(x)
    if len(categories) > 5 or any(len(c) > 6 for c in categories):
        ax.set_xticklabels(categories, rotation=30, ha='right', fontsize=9)
    else:
        ax.set_xticklabels(categories, fontsize=10)

    ax.axhline(y=100, color='gray', linestyle='--', linewidth=1, alpha=0.7)
    ax.set_ylabel('效率 (%)', fontsize=11)
    ax.set_title(title, fontsize=13, fontweight='bold', pad=12)
    ax.legend(loc='upper left', fontsize=8, ncol=min(n_dates, 3), title='日期')
    all_vals = [v for vs in matrix.values() for v in vs if v > 0]
    ax.set_ylim(0, max(max(all_vals) * 1.3, 120) if all_vals else 120)
    ax.grid(axis='y', alpha=0.3)
    ax.spines['top'].set_visible(False)
    ax.spines['right'].set_visible(False)


def build_big_dept_section(data):
    rows = [r for r in data if r.get('汇总类型') == '大部门汇总']
    if not rows:
        return None

    dates = sorted(set(r.get('工作日期', '') for r in rows))
    categories = sorted(
        set(r.get('大部门', '') for r in rows if r.get('大部门')),
        key=sort_dept_key
    )
    if not categories or not dates:
        return None

    matrix = {}
    count_matrix = {}
    for d in dates:
        matrix[d] = []
        count_matrix[d] = []
        for category in categories:
            row = next((r for r in rows if r.get('大部门') == category and r.get('工作日期') == d), None)
            matrix[d].append(parse_efficiency(row.get('效率', '0')) if row else 0)
            count_matrix[d].append(row.get('人数', '') if row else '')

    date_range = f'{dates[0]} ~ {dates[-1]}' if len(dates) > 1 else dates[0]
    return {
        'title': f'大部门生产效率对比（{date_range}）',
        'categories': categories,
        'dates': dates,
        'matrix': matrix,
        'count_matrix': count_matrix,
        'show_count': True,
        'height': 4,
    }


def build_dept_section(data, big_dept_name):
    rows = [
        r for r in data
        if r.get('汇总类型') == '部门汇总' and r.get('大部门') == big_dept_name
    ]
    if not rows:
        return None

    dates = sorted(set(r.get('工作日期', '') for r in rows))
    categories = sorted(
        set(r.get('部门', '') for r in rows if r.get('部门')),
        key=sort_dept_key
    )
    if not categories or not dates:
        return None

    matrix = {}
    count_matrix = {}
    for d in dates:
        matrix[d] = []
        count_matrix[d] = []
        for category in categories:
            row = next((r for r in rows if r.get('部门') == category and r.get('工作日期') == d), None)
            matrix[d].append(parse_efficiency(row.get('效率', '0')) if row else 0)
            count_matrix[d].append(row.get('人数', '') if row else '')

    date_range = f'{dates[0]} ~ {dates[-1]}' if len(dates) > 1 else dates[0]
    return {
        'title': f'{big_dept_name} 各部门生产效率对比（{date_range}）',
        'categories': categories,
        'dates': dates,
        'matrix': matrix,
        'count_matrix': count_matrix,
        'show_count': False,
        'height': max(4, len(categories) * 0.35 + 2),
    }


def gen_combined_chart(data, output_dir, prefix, dpi):
    sections = []
    big_section = build_big_dept_section(data)
    if big_section:
        sections.append(big_section)

    for big_dept_name in ['成衣一部', '成衣二部', '成衣三部']:
        section = build_dept_section(data, big_dept_name)
        if section:
            sections.append(section)

    if not sections:
        return None

    max_categories = max(len(section['categories']) for section in sections)
    max_dates = max(len(section['dates']) for section in sections)
    fig_width = max(12, max_categories * max_dates * 0.55 + 4)
    heights = [section['height'] for section in sections]
    fig, axes = plt.subplots(
        len(sections),
        1,
        figsize=(fig_width, sum(heights)),
        gridspec_kw={'height_ratios': heights},
        squeeze=False
    )

    for ax, section in zip(axes.flatten(), sections):
        draw_grouped_bar_chart(
            ax,
            section['categories'],
            section['dates'],
            section['matrix'],
            section['count_matrix'],
            section['title'],
            show_count=section['show_count']
        )

    plt.tight_layout(h_pad=2.5)
    filepath = os.path.join(output_dir, f'{prefix}_combined.png')
    fig.savefig(filepath, dpi=dpi, bbox_inches='tight')
    plt.close(fig)

    description = '生产效率组合柱状图，依次包含大部门、成衣一部、成衣二部、成衣三部'
    return filepath, description


def main():
    parser = argparse.ArgumentParser(description='生产效率柱状图生成脚本')
    parser.add_argument('--input', '-i', help='输入 JSON 文件路径（不指定则从 stdin 读取）')
    parser.add_argument('--output-dir', '-o', required=True, help='输出目录路径')
    parser.add_argument('--prefix', '-p', default='efficiency', help='文件名前缀（默认: efficiency）')
    parser.add_argument('--dpi', type=int, default=150, help='图片 DPI（默认: 150）')
    args = parser.parse_args()

    # 确保输出目录存在
    os.makedirs(args.output_dir, exist_ok=True)

    # 读取数据
    if args.input:
        with open(args.input, 'r', encoding='utf-8') as f:
            raw = json.load(f)
    else:
        raw = json.load(sys.stdin)

    # 兼容直接传入 data 数组或带 data 字段的对象
    if isinstance(raw, list):
        data = raw
    elif isinstance(raw, dict) and 'data' in raw:
        data = raw['data']
    else:
        print(json.dumps({'error': '无法识别的 JSON 结构，需要 data 数组或直接的数组'}, ensure_ascii=False))
        sys.exit(1)

    if not data:
        print(json.dumps({'error': '数据为空，无法生成图表'}, ensure_ascii=False))
        sys.exit(1)

    results = []

    result = gen_combined_chart(data, args.output_dir, args.prefix, args.dpi)
    if result:
        results.append({
            'chart_type': '组合对比',
            'file_path': result[0],
            'description': result[1],
        })

    # 输出结果
    output = {
        'generated_charts': len(results),
        'charts': results,
    }
    print(json.dumps(output, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()

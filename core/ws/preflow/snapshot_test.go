package preflow

import (
	"encoding/json"
	"testing"
)

func TestParseSnapshotVariants(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantNil bool
		check   func(*testing.T, *Page)
	}{
		{
			name: "标准结构",
			raw:  `{"view":"OrderList","title":"生产工单","rev":5,"navigation":[{"view":"Yield","title":"良率报表"}],"actions":[{"id":"refresh","label":"刷新"}]}`,
			check: func(t *testing.T, p *Page) {
				if p.View != "OrderList" || p.Title != "生产工单" || p.Rev != 5 {
					t.Errorf("页面信息 = %+v", p)
				}
				if len(p.Nav) != 1 || p.Nav[0].View != "Yield" || p.Nav[0].Title != "良率报表" {
					t.Errorf("菜单 = %+v", p.Nav)
				}
				if len(p.Actions) != 1 || p.Actions[0].ID != "refresh" || p.Actions[0].RequiresParams {
					t.Errorf("入口 = %+v", p.Actions)
				}
			},
		},
		{
			name: "嵌套与别名键",
			raw:  `{"page":{"view":"Home","name":"工作台","menus":[{"key":"V1","name":"工单"}],"commands":[{"cmd":"open","name":"打开"}]}}`,
			check: func(t *testing.T, p *Page) {
				if p.Title != "工作台" || len(p.Nav) != 1 || p.Nav[0].View != "V1" || p.Nav[0].Title != "工单" {
					t.Errorf("宽松取值失败: %+v", p)
				}
				if len(p.Actions) != 1 || p.Actions[0].ID != "open" || p.Actions[0].Label != "打开" {
					t.Errorf("入口 = %+v", p.Actions)
				}
			},
		},
		{
			name: "参数声明形态",
			raw:  `{"view":"Order","actions":[{"id":"a","label":"甲","params":["orderId"]},{"id":"b","label":"乙","params":[]},{"id":"c","label":"丙"}]}`,
			check: func(t *testing.T, p *Page) {
				wants := map[string]bool{"a": true, "b": false, "c": false}
				if len(p.Actions) != len(wants) {
					t.Fatalf("入口数 = %d", len(p.Actions))
				}
				for _, item := range p.Actions {
					if item.RequiresParams != wants[item.ID] {
						t.Errorf("入口 %s RequiresParams = %v, want %v", item.ID, item.RequiresParams, wants[item.ID])
					}
				}
			},
		},
		{
			name: "去重与脏数据",
			raw:  `{"view":"A","navigation":[{"view":"B","title":"乙"},{"view":"B","title":"乙重复"},{"view":"","title":"没有 view"},{"view":"C"},{"404":"不是对象"}],"actions":[{"id":"x"},{"id":"x"},{"id":""}]}`,
			check: func(t *testing.T, p *Page) {
				if len(p.Nav) != 2 || p.Nav[1].View != "C" {
					t.Errorf("菜单应去重且跳过空 view: %+v", p.Nav)
				}
				if len(p.Actions) != 1 {
					t.Errorf("入口应去重: %+v", p.Actions)
				}
			},
		},
		{
			name:    "只有筛选与数据",
			raw:     `{"filters":{"status":"open"},"data":{"rows":[]}}`,
			wantNil: true,
		},
		{
			name:    "数组形态的快照",
			raw:     `[{"id":"a"}]`,
			wantNil: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page := parseSnapshot(json.RawMessage(tc.raw), 0)
			if tc.wantNil {
				if page != nil {
					t.Fatalf("应判定为不可用，实际 %+v", page)
				}
				return
			}
			if page == nil {
				t.Fatal("解析失败")
			}
			tc.check(t, page)
		})
	}
}

func TestParseSnapshotRev(t *testing.T) {
	// 回执里的 rev 优先，快照里的兜底
	if p := parseSnapshot(json.RawMessage(`{"view":"Order","title":"工单"}`), 9); p.Rev != 9 {
		t.Errorf("rev = %d, want 9", p.Rev)
	}
	if p := parseSnapshot(json.RawMessage(`{"view":"Order","rev":3}`), 0); p.Rev != 3 {
		t.Errorf("rev = %d, want 3", p.Rev)
	}
}

func TestParseSnapshotInvalid(t *testing.T) {
	for _, raw := range []string{"", "null", "[]", `"只有一段文本"`, "{bad json"} {
		if p := parseSnapshot(json.RawMessage(raw), 1); p != nil {
			t.Errorf("%s 应判定为不可用，实际 %+v", raw, p)
		}
	}
}

func TestPageLookups(t *testing.T) {
	page := parseSnapshot(json.RawMessage(`{"view":"A","navigation":[{"view":"B","title":"乙"}],"actions":[{"id":"c","label":"丙"}]}`), 0)
	if item, ok := page.NavOf("B"); !ok || item.Title != "乙" {
		t.Errorf("NavOf = %+v %v", item, ok)
	}
	if _, ok := page.NavOf("nope"); ok {
		t.Error("NavOf 不存在时不应命中")
	}
	if item, ok := page.Action("c"); !ok || item.Label != "丙" {
		t.Errorf("Action = %+v %v", item, ok)
	}
	if _, ok := page.Action("nope"); ok {
		t.Error("Action 不存在时不应命中")
	}
	if page.Name() != "A" {
		t.Errorf("没有 title 时应退回 view: %q", page.Name())
	}

	// 空页面对象要安全：Route 的降级路径上会碰到
	var nilPage *Page
	if nilPage.Name() != "" {
		t.Error("空页面不应有名字")
	}
	if _, ok := nilPage.Action("x"); ok {
		t.Error("空页面不应命中入口")
	}
	if _, ok := nilPage.NavOf("x"); ok {
		t.Error("空页面不应命中菜单")
	}
}

func TestClip(t *testing.T) {
	if got := clip("  中文   摘要  ", 0); got != "中文 摘要" {
		t.Errorf("应压缩空白: %q", got)
	}
	if got := clip("一二三四五", 3); got != "一二三…" {
		t.Errorf("应按 rune 截断: %q", got)
	}
	if got := clip("short", 10); got != "short" {
		t.Errorf("未超限时不该改动: %q", got)
	}
}

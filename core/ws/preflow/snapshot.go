// 本文件解析浏览器回传的界面快照。
//
// 快照结构由前端生产，服务端一直是透传的；这里只做「够决策」的宽松提取：
// 键名允许多种写法，取不到的字段留空，解析不出可用信息就判定为「不做前置路由」，
// 让消息原样进 agent，绝不因为界面结构变化而卡住主流程。

package preflow

import (
	"encoding/json"
	"strings"
)

// Page 一次界面快照里与路由相关的信息
type Page struct {
	// View 页面标识（navigate 的目标）
	View string
	// Title 页面名称
	Title string
	// Rev 页面版本号，act 需要带上
	Rev int64
	// Actions 当前页面上的入口（按钮、弹窗、表单、抽屉）
	Actions []Action
	// Nav 可打开的业务页面（菜单里的叶子节点，分类节点已剔除）
	Nav []NavItem
	// Groups 被判定为分类/分组而没有独立界面的菜单节点，只用于日志，不送决策模型
	Groups []NavItem
}

// NavItem 一个可打开的页面
type NavItem struct {
	View  string
	Title string
	// Desc 菜单里的补充说明（分组、路径、描述、别名）。只用来把几十个相近菜单的差别
	// 写进选项描述，不参与匹配。
	Desc string
	// Path 节点自带的路由；分类节点通常没有
	Path string
	// Kind 节点类型（如 group / category / menu），缺失时靠 children 判断
	Kind string
	// Children 下级菜单数量：大于 0 说明它是分类节点，点开的是分组页而不是业务界面
	Children int
	// Selectable 前端显式标注「能不能打开」；nil 表示没这个字段，按其他线索判断
	Selectable *bool
}

// Action 页面上的一个入口
type Action struct {
	ID string
	// Label 入口名称
	Label string
	// Desc 入口的补充说明（描述、所属分组等）
	Desc string
	// RequiresParams 声明了参数：决策模型不生成文本，参数只能由 agent 从业务里取，
	// 所以这类入口不代办。
	RequiresParams bool
}

// 快照字段的候选键名：前端命名不统一时按顺序取第一个非空值。
var (
	nestedKeys = []string{"page", "snapshot", "state"}
	viewKeys   = []string{"view", "key", "route", "path"}
	titleKeys  = []string{"title", "name", "label", "text"}
	pathKeys   = []string{"path", "route", "url", "href"}
	kindKeys   = []string{"type", "kind", "nodeType", "menuType", "itemType"}
	leafKeys   = []string{"selectable", "clickable", "navigable", "available", "isLeaf", "leaf", "enabled"}
	descKeys   = []string{"description", "desc", "group", "parent", "keywords", "alias", "tip", "hint"}
	navKeys    = []string{"navigation", "menus", "menu", "pages", "views", "routes"}
	childKeys  = []string{"children", "items", "submenus", "submenu", "options", "nodes"}
	actionKeys = []string{"actions", "commands", "entries", "buttons"}
	actionIDs  = []string{"id", "cmd", "key", "name"}
	paramKeys  = []string{"params", "arguments", "args", "required"}
)

// parseSnapshot 宽松解析界面快照；拿不到任何可用信息时返回 nil。
func parseSnapshot(raw json.RawMessage, rev int64) *Page {
	if len(raw) == 0 {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		return nil
	}
	// 快照内容可能整体挂在 page/snapshot 下面
	if len(pick(root, navKeys)) == 0 && len(pick(root, actionKeys)) == 0 {
		for _, key := range nestedKeys {
			if nested, ok := root[key].(map[string]any); ok {
				root = nested
				break
			}
		}
	}

	page := &Page{
		View:  pickString(root, viewKeys),
		Title: pickString(root, titleKeys),
		Rev:   rev,
	}
	if page.Rev == 0 {
		if n, ok := root["rev"].(float64); ok {
			page.Rev = int64(n)
		}
	}
	// 菜单里只留「能直接打开的业务页面」：分类节点也带 view，选中它只会把用户带到
	// 一个分组页（实测「我要看预订单」被开到「预订单」分类而不是「预订单管理」）。
	for _, item := range pick(root, navKeys) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		view := pickString(m, viewKeys)
		if view == "" {
			continue
		}
		nav := NavItem{
			View:       view,
			Title:      pickString(m, titleKeys),
			Desc:       pickString(m, descKeys),
			Path:       pickString(m, pathKeys),
			Kind:       pickString(m, kindKeys),
			Children:   len(pickArray(m, childKeys)),
			Selectable: boolField(m, leafKeys),
		}
		if nav.isCategory() {
			page.Groups = append(page.Groups, nav)
			continue
		}
		page.Nav = append(page.Nav, nav)
	}
	for _, item := range pick(root, actionKeys) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := pickString(m, actionIDs)
		if id == "" {
			continue
		}
		page.Actions = append(page.Actions, Action{
			ID:             id,
			Label:          pickString(m, titleKeys),
			Desc:           pickString(m, descKeys),
			RequiresParams: hasParams(m),
		})
	}
	if page.View == "" && len(page.Nav) == 0 && len(page.Actions) == 0 {
		return nil
	}
	return dedupe(page)
}

// dedupe 去掉重复项：页面 view 或入口 id 重复会让选项撞车、act 无法定位
func dedupe(page *Page) *Page {
	seen := map[string]bool{}
	nav := page.Nav[:0]
	for _, item := range page.Nav {
		if seen["v:"+item.View] {
			continue
		}
		seen["v:"+item.View] = true
		nav = append(nav, item)
	}
	seen = map[string]bool{}
	actions := page.Actions[:0]
	for _, item := range page.Actions {
		if seen["a:"+item.ID] {
			continue
		}
		seen["a:"+item.ID] = true
		actions = append(actions, item)
	}
	page.Nav, page.Actions = nav, actions
	return page
}

// pick 返回第一个非空数组字段
func pick(m map[string]any, keys []string) []any {
	for _, key := range keys {
		if arr, ok := m[key].([]any); ok && len(arr) > 0 {
			return arr
		}
	}
	return nil
}

// pickArray 按候选键名取任意数组字段（长度即子节点数量）
func pickArray(m map[string]any, keys []string) []any {
	for _, key := range keys {
		if arr, ok := m[key].([]any); ok {
			return arr
		}
	}
	return nil
}

// boolField 取布尔标记：只有字段真的存在才返回非 nil，缺字段不等于 false
func boolField(m map[string]any, keys []string) *bool {
	for _, key := range keys {
		v, ok := m[key]
		if !ok {
			continue
		}
		switch b := v.(type) {
		case bool:
			return &b
		case string:
			parsed := b == "true" || b == "1" || b == "yes"
			return &parsed
		case float64:
			parsed := b != 0
			return &parsed
		}
	}
	return nil
}

// categoryKinds 明确表示「这是个分组/目录，不是业务界面」的节点类型
var categoryKinds = map[string]bool{
	"group": true, "category": true, "directory": true, "dir": true,
	"parent": true, "submenu": true, "sub_menu": true, "catalog": true,
	"folder": true, "header": true, "divider": true, "title": true, "space": true,
}

// isCategory 判断菜单节点是不是只能展开、没有独立界面的分类：
// 前端标了不可选、类型是分组目录、或带着下级菜单，都不该进候选列表。
func (n NavItem) isCategory() bool {
	if n.Selectable != nil && !*n.Selectable {
		return true
	}
	if categoryKinds[strings.ToLower(strings.TrimSpace(n.Kind))] {
		return true
	}
	return n.Children > 0
}

// pickString 返回第一个非空字符串字段
func pickString(m map[string]any, keys []string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// hasParams 判断入口是否声明了参数（对象或数组非空都算）
func hasParams(m map[string]any) bool {
	for _, key := range paramKeys {
		switch v := m[key].(type) {
		case map[string]any:
			if len(v) > 0 {
				return true
			}
		case []any:
			if len(v) > 0 {
				return true
			}
		}
	}
	return false
}

// Action 按 id 取入口
func (p *Page) Action(id string) (Action, bool) {
	if p == nil {
		return Action{}, false
	}
	for _, item := range p.Actions {
		if item.ID == id {
			return item, true
		}
	}
	return Action{}, false
}

// NavOf 按 view 取页面
func (p *Page) NavOf(view string) (NavItem, bool) {
	if p == nil {
		return NavItem{}, false
	}
	for _, item := range p.Nav {
		if item.View == view {
			return item, true
		}
	}
	return NavItem{}, false
}

// Name 页面的人可读名字
func (p *Page) Name() string {
	if p == nil {
		return ""
	}
	if p.Title != "" {
		return p.Title
	}
	return p.View
}

// clip 按 rune 截断，避免切坏中文
func clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

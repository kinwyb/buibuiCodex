package db

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestSessionStorage(t *testing.T) *sessionStorage {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	s := newSessionStorage(gdb)
	s.Init()
	return s
}

func mustCreateTurn(t *testing.T, s *sessionStorage, turn *SessionTurn) {
	t.Helper()
	if err := s.db.Create(turn).Error; err != nil {
		t.Fatalf("插入轮次失败: %v", err)
	}
}

func seedTurns(t *testing.T, s *sessionStorage) time.Time {
	t.Helper()
	base := time.Now().Add(-3 * time.Hour)
	mustCreateTurn(t, s, &SessionTurn{SessionID: "s1", ReqID: "r1", ThreadID: "t1", TurnID: "u1", Agent: "codex", Question: "q1", Answer: "a1", IsCompleted: true, CreatedAt: base})
	mustCreateTurn(t, s, &SessionTurn{SessionID: "s1", ReqID: "r2", ThreadID: "t1", TurnID: "u2", Agent: "codex", Question: "q2", Answer: "a2", IsCompleted: true, CreatedAt: base.Add(time.Hour)})
	mustCreateTurn(t, s, &SessionTurn{SessionID: "s1", ReqID: "r3", ThreadID: "t2", Agent: "coder", Question: "q3", CreatedAt: base.Add(2 * time.Hour)})
	// 其他会话的数据不应被查出
	mustCreateTurn(t, s, &SessionTurn{SessionID: "s2", ReqID: "r4", ThreadID: "t3", CreatedAt: base})

	for _, th := range []SessionThread{
		{ThreadID: "t1", SessionID: "s1", Agent: "codex", CreateTime: base, LastUpdate: base.Add(time.Hour)},
		{ThreadID: "t2", SessionID: "s1", Agent: "coder", CreateTime: base.Add(2 * time.Hour), LastUpdate: base.Add(2 * time.Hour)},
	} {
		thread := th
		if err := s.db.Create(&thread).Error; err != nil {
			t.Fatalf("插入线程失败: %v", err)
		}
	}
	return base
}

func reqIDs(list []SessionTurn) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, item.ReqID)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTurnListBySession(t *testing.T) {
	s := newTestSessionStorage(t)
	seedTurns(t, s)

	list, err := s.TurnList(context.Background(), TurnQuery{SessionID: "s1", Limit: -1})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got := reqIDs(list); !equalStrings(got, []string{"r1", "r2", "r3"}) {
		t.Fatalf("应按 seq 升序返回本会话全部轮次，实际 %v", got)
	}
}

func TestTurnListByThread(t *testing.T) {
	s := newTestSessionStorage(t)
	seedTurns(t, s)

	list, err := s.TurnList(context.Background(), TurnQuery{SessionID: "s1", ThreadID: "t1"})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got := reqIDs(list); !equalStrings(got, []string{"r1", "r2"}) {
		t.Fatalf("应只返回 t1 的轮次，实际 %v", got)
	}
}

func TestTurnListLimitAndBeforeSeq(t *testing.T) {
	s := newTestSessionStorage(t)
	seedTurns(t, s)

	// limit 取最近 N 条，但返回仍是升序
	list, err := s.TurnList(context.Background(), TurnQuery{SessionID: "s1", Limit: 2})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got := reqIDs(list); !equalStrings(got, []string{"r2", "r3"}) {
		t.Fatalf("limit 应取最近两条并升序返回，实际 %v", got)
	}

	// 用本页最新一条的 seq 作为游标，继续向前翻页
	before := list[len(list)-1].Seq
	prev, err := s.TurnList(context.Background(), TurnQuery{SessionID: "s1", BeforeSeq: before})
	if err != nil {
		t.Fatalf("翻页失败: %v", err)
	}
	if got := reqIDs(prev); !equalStrings(got, []string{"r1", "r2"}) {
		t.Fatalf("翻页应返回更早的 r1、r2，实际 %v", got)
	}
}

func TestTurnListSince(t *testing.T) {
	s := newTestSessionStorage(t)
	base := seedTurns(t, s)

	list, err := s.TurnList(context.Background(), TurnQuery{SessionID: "s1", Since: base.Add(90 * time.Minute)})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got := reqIDs(list); !equalStrings(got, []string{"r3"}) {
		t.Fatalf("since 应只返回之后的轮次，实际 %v", got)
	}
}

func TestTurnListRequiresCondition(t *testing.T) {
	s := newTestSessionStorage(t)
	if _, err := s.TurnList(context.Background(), TurnQuery{}); err == nil {
		t.Fatal("缺少 sessionID 与 threadID 时应返回错误")
	}
}

func TestThreadList(t *testing.T) {
	s := newTestSessionStorage(t)
	seedTurns(t, s)

	threads, err := s.ThreadList(context.Background(), "s1")
	if err != nil {
		t.Fatalf("查询线程失败: %v", err)
	}
	if len(threads) != 2 || threads[0].ThreadID != "t1" || threads[1].ThreadID != "t2" {
		t.Fatalf("应按创建时间升序返回两个线程，实际 %+v", threads)
	}
}

func TestRequestSaveKeepsSessionID(t *testing.T) {
	s := newTestSessionStorage(t)
	if err := s.RequestSave(context.Background(), "s9", "r9", "问题", "codex"); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	list, err := s.TurnList(context.Background(), TurnQuery{SessionID: "s9"})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(list) != 1 || list[0].SessionID != "s9" || list[0].Question != "问题" {
		t.Fatalf("请求保存后应能按 sessionID 查到: %+v", list)
	}
}

// TestTurnSaveKeepsExistingSessionID 锁定语义：
// manager 完成时只补 Answer，不应把 cdm 建 turn 时写入的 SessionID/ThreadID 清空。
// 实现依赖 gorm 的 Updates(struct) 跳过零值字段；若改成 Save 会整行覆盖，此用例会失败。
func TestTurnSaveKeepsExistingSessionID(t *testing.T) {
	s := newTestSessionStorage(t)
	ctx := context.Background()

	if err := s.RequestSave(ctx, "s1", "r1", "问题", "codex"); err != nil {
		t.Fatalf("保存请求失败: %v", err)
	}
	// 模拟 cdm 建 turn：写入 thread/session 关联
	if err := s.TurnSave(ctx, &SessionTurn{ReqID: "r1", TurnID: "u1", ThreadID: "t1", SessionID: "s1"}); err != nil {
		t.Fatalf("写入 turn 关联失败: %v", err)
	}
	// 模拟 manager 完成：不带 SessionID/ThreadID
	if err := s.TurnSave(ctx, &SessionTurn{ReqID: "r1", Answer: "回答", IsCompleted: true}); err != nil {
		t.Fatalf("补写回答失败: %v", err)
	}

	list, err := s.TurnList(ctx, TurnQuery{SessionID: "s1"})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("应查到 1 条轮次，实际 %d", len(list))
	}
	got := list[0]
	if got.SessionID != "s1" || got.ThreadID != "t1" || got.TurnID != "u1" {
		t.Fatalf("补写回答不应清空关联字段: %+v", got)
	}
	if got.Answer != "回答" || !got.IsCompleted {
		t.Fatalf("回答未正确补写: %+v", got)
	}
}

func TestNormalizeLimit(t *testing.T) {
	cases := []struct {
		limit int
		want  int
	}{
		{0, DefaultTurnLimit},
		{-1, MaxTurnLimit},
		{10, 10},
		{MaxTurnLimit + 1, MaxTurnLimit},
	}
	for _, c := range cases {
		if got := (TurnQuery{Limit: c.limit}).NormalizeLimit(); got != c.want {
			t.Fatalf("limit=%d 应归一化为 %d，实际 %d", c.limit, c.want, got)
		}
	}
}

func TestTurnEventListFiltersAndOrders(t *testing.T) {
	s := newTestSessionStorage(t)
	ctx := context.Background()
	base := time.Now().Add(-time.Hour)

	// 同一轮里既有逐 token 的 delta，也有关键帧；回放只应拿到关键帧
	events := []TurnEvent{
		{ID: "e1", ThreadID: "t1", TurnID: "u1", Type: "notification", Method: "item/started", RawData: `{"item":{"id":"i1"}}`, CreateTime: base},
		{ID: "e2", ThreadID: "t1", TurnID: "u1", Type: "notification", Method: "item/reasoning/textDelta", RawData: `{"delta":"a"}`, CreateTime: base},
		{ID: "e3", ThreadID: "t1", TurnID: "u1", Type: "notification", Method: "item/completed", RawData: `{"item":{"id":"i1"}}`, CreateTime: base},
		{ID: "e4", ThreadID: "t1", TurnID: "u2", Type: "notification", Method: "item/completed", RawData: `{"item":{"id":"i2"}}`, CreateTime: base},
		{ID: "e5", ThreadID: "t2", TurnID: "u3", Type: "notification", Method: "item/completed", RawData: `{"item":{"id":"i3"}}`, CreateTime: base},
	}
	for i := range events {
		if err := s.TurnEventSave(ctx, &events[i]); err != nil {
			t.Fatalf("写入事件失败: %v", err)
		}
	}

	list, err := s.TurnEventList(ctx, TurnEventQuery{
		TurnIDs: []string{"u1", "u2"},
		Methods: []string{"item/started", "item/completed"},
	})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("应只拿到 2 轮的关键帧共 3 条，实际 %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Seq >= list[i].Seq {
			t.Fatalf("事件未按 seq 升序返回: %d >= %d", list[i-1].Seq, list[i].Seq)
		}
	}

	// 只取指定轮次，且不越界到其他线程
	for _, ev := range list {
		if ev.TurnID != "u1" && ev.TurnID != "u2" {
			t.Fatalf("查出了其他轮次的事件: %+v", ev)
		}
		if ev.Method == "item/reasoning/textDelta" {
			t.Fatalf("delta 应被 method 白名单过滤掉: %+v", ev)
		}
	}
}

func TestTurnEventListAfterSeq(t *testing.T) {
	s := newTestSessionStorage(t)
	ctx := context.Background()
	base := time.Now()
	for i := 0; i < 3; i++ {
		ev := &TurnEvent{ID: "e" + string(rune('a'+i)), ThreadID: "t1", TurnID: "u1", Method: "item/completed", CreateTime: base}
		if err := s.TurnEventSave(ctx, ev); err != nil {
			t.Fatalf("写入事件失败: %v", err)
		}
	}

	all, err := s.TurnEventList(ctx, TurnEventQuery{TurnIDs: []string{"u1"}})
	if err != nil {
		t.Fatalf("查询事件失败: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("应查到 3 条事件，实际 %d", len(all))
	}

	rest, err := s.TurnEventList(ctx, TurnEventQuery{TurnIDs: []string{"u1"}, AfterSeq: all[0].Seq})
	if err != nil {
		t.Fatalf("按游标查询失败: %v", err)
	}
	if len(rest) != 2 || rest[0].Seq <= all[0].Seq {
		t.Fatalf("afterSeq 应跳过已读事件: %+v", rest)
	}
}

func TestTurnEventListRequiresScope(t *testing.T) {
	s := newTestSessionStorage(t)
	if _, err := s.TurnEventList(context.Background(), TurnEventQuery{}); err == nil {
		t.Fatal("缺少 turnID / threadID 时应报错，避免全表扫描")
	}
}

func TestTurnEventLimitNormalize(t *testing.T) {
	cases := []struct {
		limit int
		want  int
	}{
		{0, DefaultTurnEventLimit},
		{-1, MaxTurnEventLimit},
		{10, 10},
		{MaxTurnEventLimit + 1, MaxTurnEventLimit},
	}
	for _, c := range cases {
		if got := (TurnEventQuery{Limit: c.limit}).NormalizeLimit(); got != c.want {
			t.Fatalf("limit=%d 应归一化为 %d，实际 %d", c.limit, c.want, got)
		}
	}
}

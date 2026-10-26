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

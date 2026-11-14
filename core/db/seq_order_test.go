package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestListBySeqOrderTakesDescPageButReturnsAscending(t *testing.T) {
	storage, err := NewSQLiteStorage(filepath.Join(t.TempDir(), "seq-order.db"))
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	data := NewData(storage)
	defer data.Close()
	s := data.Session()
	ctx := context.Background()
	for _, reqID := range []string{"r1", "r2", "r3", "r4", "r5"} {
		if err := s.RequestSave(ctx, "s1", reqID, "问题", "codex"); err != nil {
			t.Fatalf("保存请求失败: %v", err)
		}
	}

	tx := storage.GetDB().Model(&SessionTurn{}).Where("session_id = ?", "s1")
	list, err := listBySeqOrder[SessionTurn](ctx, storage.GetDB(), tx, "seq desc", 3)
	if err != nil {
		t.Fatalf("排序查询失败: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("应取最近 3 条，实际 %d", len(list))
	}
	if list[0].ReqID != "r3" || list[1].ReqID != "r4" || list[2].ReqID != "r5" {
		t.Fatalf("应取最近 3 条并按 seq 升序返回: %+v", list)
	}
}

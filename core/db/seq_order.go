package db

import (
	"context"
	"sort"

	"gorm.io/gorm"
)

type seqModel interface {
	seq() uint
}

func (t SessionTurn) seq() uint { return t.Seq }
func (t TurnEvent) seq() uint   { return t.Seq }

// listBySeqOrder 先只排序主键 seq，再按主键取整行，避免 MySQL filesort
// 携带 raw_data/question/answer 大字段。order 决定取哪一端的主键，
// 结果统一按 seq 升序返回，保持回放与历史时序一致。
func listBySeqOrder[T seqModel](ctx context.Context, db *gorm.DB, tx *gorm.DB, order string, limit int) ([]T, error) {
	var seqs []uint
	if err := tx.Order(order).Limit(limit).Pluck("seq", &seqs).Error; err != nil {
		return nil, err
	}
	if len(seqs) == 0 {
		return nil, nil
	}
	var list []T
	if err := db.WithContext(ctx).Where("seq IN ?", seqs).Find(&list).Error; err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].seq() < list[j].seq() })
	return list, nil
}

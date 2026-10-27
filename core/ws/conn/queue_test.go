package conn

import (
	"strconv"
	"testing"
)

// ts 造一条带可识别内容的帧
func ts(label string, critical bool) outbound {
	return outbound{raw: []byte(label), critical: critical}
}

// labels 依次取出队列里所有帧的内容，顺带断言顺序
func labels(t *testing.T, q *outQueue) []string {
	t.Helper()
	var out []string
	for {
		item, ok := q.pop()
		if !ok {
			return out
		}
		out = append(out, string(item.raw))
	}
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// 队列满时丢「最旧的那条增量帧」，新帧一定进得去，其余帧的相对顺序不变
func TestOutQueueDropsOldestDelta(t *testing.T) {
	q := newOutQueue(3)
	for _, label := range []string{"d1", "d2", "d3"} {
		if ok, dropped := q.push(ts(label, false)); !ok || dropped != 0 {
			t.Fatalf("未满时入队应成功且不丢弃：%s ok=%v dropped=%d", label, ok, dropped)
		}
	}

	ok, dropped := q.push(ts("d4", false))
	if !ok || dropped != 1 {
		t.Fatalf("满队列来新增量帧应挤掉最旧的一条：ok=%v dropped=%d", ok, dropped)
	}

	if got := labels(t, q); !equal(got, []string{"d2", "d3", "d4"}) {
		t.Fatalf("丢的应是最旧的一条且顺序不变，实际 %v", got)
	}
}

// 关键帧不会被挤掉：满队列时先丢增量帧给它让位
func TestOutQueueKeepsCritical(t *testing.T) {
	q := newOutQueue(3)
	q.push(ts("start", true))
	q.push(ts("d1", false))
	q.push(ts("d2", false))

	ok, dropped := q.push(ts("completed", true))
	if !ok || dropped != 1 {
		t.Fatalf("关键帧应挤掉增量帧让位：ok=%v dropped=%d", ok, dropped)
	}
	if got := labels(t, q); !equal(got, []string{"start", "d2", "completed"}) {
		t.Fatalf("关键帧不能被丢，实际 %v", got)
	}
}

// 队列里全是关键帧、腾不出位置：拒绝入队，由调用方决定断开
func TestOutQueueRejectsWhenAllCritical(t *testing.T) {
	q := newOutQueue(2)
	q.push(ts("a", true))
	q.push(ts("b", true))

	ok, dropped := q.push(ts("c", true))
	if ok || dropped != 0 {
		t.Fatalf("全是关键帧时应拒绝入队：ok=%v dropped=%d", ok, dropped)
	}
	if got := labels(t, q); !equal(got, []string{"a", "b"}) {
		t.Fatalf("拒绝入队不应破坏已有帧，实际 %v", got)
	}

	// 增量帧在同样的情况下只丢自己，不算「丢弃计数」（队列没变）
	q2 := newOutQueue(1)
	q2.push(ts("only", true))
	if ok, dropped := q2.push(ts("late-delta", false)); ok || dropped != 0 {
		t.Fatalf("没有可让位的增量帧时应拒绝新增量帧：ok=%v dropped=%d", ok, dropped)
	}
	if queued, dropped := q2.stats(); queued != 1 || dropped != 0 {
		t.Fatalf("排队与丢弃计数应保持不变：queued=%d dropped=%d", queued, dropped)
	}
}

// 反复 pop 之后容量要能自愈，不能一直缩小导致频繁重新分配
func TestOutQueueReusesCapacity(t *testing.T) {
	q := newOutQueue(8)
	for round := 0; round < 100; round++ {
		for i := 0; i < 5; i++ {
			q.push(ts(strconv.Itoa(i), false))
		}
		for i := 0; i < 5; i++ {
			if _, ok := q.pop(); !ok {
				t.Fatalf("第 %d 轮第 %d 次 pop 应有数据", round, i)
			}
		}
	}
	if _, ok := q.pop(); ok {
		t.Fatal("取空后不应再有数据")
	}
	if queued, dropped := q.stats(); queued != 0 || dropped != 0 {
		t.Fatalf("容量 8 装 5 条不该触发丢弃：queued=%d dropped=%d", queued, dropped)
	}
}

// 入队要唤醒消费者，且重复通知合并成一次
func TestOutQueueSignalsOnce(t *testing.T) {
	q := newOutQueue(4)
	q.push(ts("a", false))
	q.push(ts("b", false))

	select {
	case <-q.wake():
	default:
		t.Fatal("入队后应唤醒消费者")
	}
	select {
	case <-q.wake():
		t.Fatal("重复通知应被合并，不残留第二次信号")
	default:
	}
}

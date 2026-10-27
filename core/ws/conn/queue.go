// 本文件实现单连接的发送队列。
//
// 为什么不用带缓冲的 channel：channel 满时只能「丢掉新帧」，而新帧恰恰是用户
// 此刻最该看到的那一帧。一轮回答里 delta 被连续丢弃时，界面会整段空白，直到
// message_completed 的全量文本到达才一次性刷出来 —— 现象就是「没有流式」。
//
// 这里的策略改成「丢最旧的那条 delta」：
//   - 关键帧（start / completed / tool / error / finished / 审批 / 系统帧）永不主动丢弃；
//   - 队列满时先腾掉最旧的一条 delta，给新帧让位，顺序保持不变；
//   - 只有当队列里全是关键帧、腾不出位置时，才由调用方按原策略断开连接。
//
// 少几个字由 completed 的全量文本收敛，比整段空白好：前者是瑕疵，后者是故障。

package conn

import "sync"

// outbound 一条待发送的已序列化消息
type outbound struct {
	raw []byte
	// critical 关键帧：丢了会让界面永久缺内容（start/completed/tool/error/finished 等），必须送达
	critical bool
}

// outQueue 有界、可丢弃最旧增量帧的发送队列，并发安全
type outQueue struct {
	mu       sync.Mutex
	items    []outbound
	capacity int
	// dropped 累计丢掉的增量帧数量。排查「界面不流式」时先看这个数：
	// 只要它在一轮里持续增长，就说明浏览器那头消费不过来。
	dropped int64
	// signal 有新数据可读的通知，容量 1：消费者的 select 靠它醒来，重复通知会被合并
	signal chan struct{}
}

func newOutQueue(capacity int) *outQueue {
	if capacity <= 0 {
		capacity = 256
	}
	return &outQueue{
		capacity: capacity,
		signal:   make(chan struct{}, 1),
	}
}

// push 入队。
//
// ok=false 表示队列里全是关键帧、腾不出位置（是否断开由调用方决定）；
// dropped 是本次为腾位置丢掉的增量帧数量（0 或 1），供调用方记日志。
func (q *outQueue) push(item outbound) (ok bool, dropped int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) >= q.capacity {
		if !q.dropOldestDeltaLocked() {
			return false, 0
		}
		dropped = 1
	}
	q.items = append(q.items, item)
	q.notifyLocked()
	return true, dropped
}

// pop 取队头一条；队列空返回 false
func (q *outQueue) pop() (outbound, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return outbound{}, false
	}
	item := q.items[0]
	// 清引用再前移：否则被丢弃的消息会一直被底层数组钉住
	q.items[0] = outbound{}
	q.items = q.items[1:]
	switch {
	case len(q.items) == 0:
		q.items = nil
	case cap(q.items) < q.capacity/2:
		// 反复 pop 会把容量一路切小，这里顺手搬一次，保证之后不再频繁重新分配
		q.items = append(make([]outbound, 0, q.capacity), q.items...)
	}
	return item, true
}

// wake 消费者等待「有新数据」用的通道
func (q *outQueue) wake() <-chan struct{} {
	return q.signal
}

// stats 当前排队长度与累计丢弃数，日志与测试用
func (q *outQueue) stats() (queued int, dropped int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items), q.dropped
}

// dropOldestDeltaLocked 丢掉队列里最旧的一条增量帧，返回是否丢掉了。
// 关键帧跳过：删中间的一条不会打乱其余帧的相对顺序。
func (q *outQueue) dropOldestDeltaLocked() bool {
	for i := range q.items {
		if q.items[i].critical {
			continue
		}
		copy(q.items[i:], q.items[i+1:])
		q.items[len(q.items)-1] = outbound{}
		q.items = q.items[:len(q.items)-1]
		q.dropped++
		return true
	}
	return false
}

// notifyLocked 通知消费者；已有未消费的通知就合并，不阻塞
func (q *outQueue) notifyLocked() {
	select {
	case q.signal <- struct{}{}:
	default:
	}
}

// 本文件实现 ping 命令：应用层心跳。

package command

import (
	"context"
	"time"

	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// ping 应用层心跳，前端可借此测量往返延迟
func (r *Registry) ping(_ context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.PingData
	_ = env.DecodeData(&data)
	c.SendEnvelope(protocol.NewEnvelope(protocol.KindSystem, protocol.TypePong).
		WithReplyTo(env.ID).
		WithData(&protocol.PongData{ClientTS: data.TS, ServerTS: time.Now().UnixMilli()}))
}

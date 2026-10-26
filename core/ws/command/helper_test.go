// 本文件存放各命令测试共用的桩与构造器。

package command

import (
	"context"

	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/db"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
)

type fakeAgents []string

func (f fakeAgents) ListAgents() []string { return []string(f) }

func newTestRegistry(msgBus *bus.MessageBus, agents AgentSource) (*Registry, *conn.Hub) {
	return newTestRegistryWithTurns(msgBus, agents, nil)
}

func newTestRegistryWithTurns(msgBus *bus.MessageBus, agents AgentSource, turns TurnReader) (*Registry, *conn.Hub) {
	hub := conn.NewHub()
	return NewRegistry(Deps{
		Bus:          msgBus,
		Hub:          hub,
		Agents:       agents,
		Turns:        turns,
		Channel:      "ws",
		DefaultAgent: "codex",
	}), hub
}

func newTestClient(hub *conn.Hub) *conn.Client {
	return newTestClientAs(hub, "web")
}

func newTestClientAs(hub *conn.Hub, userID string) *conn.Client {
	c := conn.NewClient(conn.Config{SendBuffer: 8}, nil, hub, nil, conn.Identity{UserID: userID})
	hub.Add(c)
	return c
}

type fakeTurnReader struct {
	threads      []db.SessionThread
	turns        map[string][]db.SessionTurn
	queries      []db.TurnQuery
	turnEvents   map[string][]db.TurnEvent
	eventQueries []db.TurnEventQuery
	pageThreads  []db.SessionThread
	pageQueries  []db.ThreadQuery
	summaries    map[string]db.ThreadSummary
}

func (f *fakeTurnReader) ThreadPage(_ context.Context, q db.ThreadQuery) ([]db.SessionThread, error) {
	f.pageQueries = append(f.pageQueries, q)
	return f.pageThreads, nil
}

func (f *fakeTurnReader) ThreadSummaries(_ context.Context, _ []string) (map[string]db.ThreadSummary, error) {
	return f.summaries, nil
}

func (f *fakeTurnReader) ThreadList(_ context.Context, _ string) ([]db.SessionThread, error) {
	return f.threads, nil
}

func (f *fakeTurnReader) TurnList(_ context.Context, q db.TurnQuery) ([]db.SessionTurn, error) {
	f.queries = append(f.queries, q)
	return f.turns[q.ThreadID], nil
}

func (f *fakeTurnReader) TurnQueryByID(_ context.Context, turnID string) *db.SessionTurn {
	for _, list := range f.turns {
		for i := range list {
			if list[i].TurnID == turnID {
				return &list[i]
			}
		}
	}
	return nil
}

func (f *fakeTurnReader) TurnEventList(_ context.Context, q db.TurnEventQuery) ([]db.TurnEvent, error) {
	f.eventQueries = append(f.eventQueries, q)
	var out []db.TurnEvent
	for _, id := range q.TurnIDs {
		out = append(out, f.turnEvents[id]...)
	}
	return out, nil
}

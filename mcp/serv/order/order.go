package order

import "github.com/kinwyb/buibuiCodex/mcp/serv"

func InitTool() {
	serv.RegisterTool(&orderProcess{})
	serv.RegisterTool(&orderEfficiency{})
	serv.RegisterTool(&orderMu{})
}

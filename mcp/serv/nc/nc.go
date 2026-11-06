package nc

import "github.com/kinwyb/buibuiCodex/mcp/serv"

func InitTool() {
	serv.RegisterTool(materialQuery())
}

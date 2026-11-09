package nc

import "github.com/kinwyb/buibuiCodex/mcp/serv"

func InitTool() {
	serv.RegisterTool(materialQuery())
	serv.RegisterTool(countryZoneQuery())
	serv.RegisterTool(currTypeQuery())
	serv.RegisterTool(currrateQuery())
	serv.RegisterTool(customerQuery())
	serv.RegisterTool(custClass())
}

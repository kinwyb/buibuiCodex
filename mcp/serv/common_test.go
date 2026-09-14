package serv

import (
	"encoding/json/v2"
	"testing"
)

func Test_bigDataHandler(t *testing.T) {
	TmpDir = "/Users/wangyingbin/Downloads/kanflux/buibuiCodex/mcp/tmp"
	TmpUrl = "http://127.0.0.1:8000/tmp_file"
	data := dataJson{
		Columns: []string{"订单号", "客户", "数量"},
		Data: []map[string]string{
			{
				"订单号": "AT-001",
				"客户":  "AT",
				"数量":  "102",
			}, {
				"订单号": "AT-002",
				"客户":  "AT",
				"数量":  "1021",
			}, {
				"订单号": "AT-003",
				"客户":  "AT",
				"数量":  "10244",
			}, {
				"订单号": "HG-1231",
				"客户":  "HG",
				"数量":  "2501",
			}, {
				"订单号": "HG-3221",
				"客户":  "HG",
				"数量":  "5938",
			}, {
				"订单号": "HG-7712",
				"客户":  "HG",
				"数量":  "8371",
			},
		},
	}
	dataVal, _ := json.Marshal(data)
	result := bigDataHandler(string(dataVal))
	t.Log(result)
}

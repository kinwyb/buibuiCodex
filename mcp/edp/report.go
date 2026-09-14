package edp

import (
	"net/http"

	"github.com/kinwyb/buibuiCodex/mcp/models"
)

// QueryBI 查询报表。
// id: 报表ID
// args: 报表参数列表，可为空
func QueryBI(token, id string, args ...models.BIArg) (*models.TableData, error) {
	req := NewRequest(http.MethodPost, "/bi/query",
		WithToken(token),
		WithBody(models.BIQueryReq{
			ID:          id,
			Args:        args,
			ExportExecl: false,
		}),
	)

	result, err := DoClient[models.TableData](req)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

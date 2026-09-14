package edp

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/kinwyb/buibuiCodex/mcp/models"
)

// SQLBoda sql查询博大数据库
func SQLBoda(token string, sql string, args ...string) (*models.TableData, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil, errors.New("sql is empty")
	}
	parseSQL := strings.ToLower(sql)
	if strings.Contains(parseSQL, "delete ") ||
		strings.Contains(parseSQL, "update ") ||
		strings.Contains(parseSQL, "insert ") ||
		strings.Contains(parseSQL, ";") ||
		strings.Contains(parseSQL, " set ") ||
		!strings.HasPrefix(parseSQL, "select") {
		return nil, errors.New("非法SQL:仅允许SELECT语句")
	}
	ip := "" //服务端没验证暂时不用
	sb := strings.Builder{}
	sb.WriteString(token)
	sb.WriteString(sql)
	sb.WriteString(ip)
	for _, arg := range args {
		sb.WriteString(arg)
	}
	sb.WriteString(token)
	request := models.SQLQueryRequest{
		Args:       args,
		DataSource: 1,
		SQL:        sql,
		Secret:     md5Encode(sb.String()),
	}

	req := NewRequest(http.MethodPost, "/sqlAdapter/query",
		WithToken(token),
		WithBody(request),
	)

	result, err := DoClient[models.TableData](req)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func md5Encode(content string) string {
	m := md5.New()
	m.Write([]byte(content))
	data := m.Sum(nil)
	return hex.EncodeToString(data)
}

// SQLProcedureBoda 执行博大存储过程
func SQLProcedureBoda(token string, procedureName string, args map[string]string) (*models.TableData, error) {
	procedureName = strings.TrimSpace(procedureName)
	if procedureName == "" {
		return nil, errors.New("procedureName is empty")
	}
	parseSQL := strings.ToLower(procedureName)
	if strings.Contains(parseSQL, "delete ") ||
		strings.Contains(parseSQL, "update ") ||
		strings.Contains(parseSQL, "insert ") ||
		strings.Contains(parseSQL, ";") ||
		strings.Contains(parseSQL, " set ") ||
		!strings.HasPrefix(parseSQL, "select") {
		return nil, errors.New("非法SQL:仅允许SELECT语句")
	}
	var argList []string
	for k, v := range args {
		argList = append(argList, fmt.Sprintf("%s:%s", k, v))
	}
	ip := "" //服务端没验证暂时不用
	sb := strings.Builder{}
	sb.WriteString(token)
	sb.WriteString(procedureName)
	sb.WriteString(ip)
	for _, arg := range argList {
		sb.WriteString(arg)
	}
	sb.WriteString(token)
	request := models.SQLProcedureRequest{
		Args:          argList,
		DataSource:    1,
		ProcedureName: procedureName,
		Secret:        md5Encode(sb.String()),
	}

	req := NewRequest(http.MethodPost, "/sqlAdapter/procedure",
		WithToken(token),
		WithBody(request),
	)

	result, err := DoClient[models.TableData](req)
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}

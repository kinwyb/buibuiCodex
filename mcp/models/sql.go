package models

// SQLQueryRequest SQL查询请求
type SQLQueryRequest struct {
	Args       []string `json:"Args" binding:"required"`       // 参数
	DataSource int64    `json:"DataSource" binding:"required"` // 数据源
	SQL        string   `json:"SQL" binding:"required"`        // SQL语句
	Secret     string   `json:"Secret" binding:"required"`     // 校验验证
}

// SQLProcedureRequest 存储过程执行请求
type SQLProcedureRequest struct {
	Args          []string `json:"Args" binding:"required"`          // 参数
	DataSource    int64    `json:"DataSource" binding:"required"`    // 数据源
	ProcedureName string   `json:"ProcedureName" binding:"required"` // 存储过程名称
	Secret        string   `json:"Secret" binding:"required"`        // 校验验证
}

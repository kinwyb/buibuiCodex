package models

// UserInfo 用户基本信息
type UserInfo struct {
	EmpID             string   `json:"EmpID" binding:"required"`             // 员工工号
	EmpName           string   `json:"EmpName" binding:"required"`           // 员工名称
	IP                string   `json:"IP" binding:"required"`                // 用户IP
	Language          string   `json:"Language" binding:"required"`          // 语言
	OrgID             string   `json:"OrgID" binding:"required"`             // 组织ID
	Orgs              []string `json:"Orgs" binding:"required"`              // 所属组织ID列表
	RequestSucc       bool     `json:"RequestSucc" binding:"required"`       // 请求是否成功
	RequestUniqID     string   `json:"RequestUniqID" binding:"required"`     // 请求唯一ID
	RoleDepartmentIDs []string `json:"RoleDepartmentIDs" binding:"required"` // 权限部门
	RoleDepartmentNos []string `json:"RoleDepartmentNos" binding:"required"` // 权限部门编号
	RoleIDs           []string `json:"RoleIDs" binding:"required"`           // 权限组ID
	RoomCode          string   `json:"RoomCode" binding:"required"`          // 工厂
	SQLTraceID        string   `json:"SQLTraceID" binding:"required"`
	ServerMode        string   `json:"ServerMode" binding:"required"` // 服务模式
	Token             string   `json:"Token" binding:"required"`      // token标记
	Tracing           bool     `json:"Tracing" binding:"required"`    // 开启追踪
	UserID            string   `json:"UserID" binding:"required"`     // 用户ID
}

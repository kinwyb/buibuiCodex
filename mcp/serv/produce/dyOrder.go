package produce

import (
	"errors"

	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type DyOrderList struct {
	StartDate string `json:"StartDate" description:"开始日期(yyyy-mm-dd)"`
	EndDate   string `json:"EndDate" description:"结束日期(yyyy-mm-dd)"`
	DyUser    string `json:"DyUser,omitempty" description:"打样师傅名称"`
	OrderCode string `json:"OrderCode,omitempty" description:"订单号"`
	StyleCode string `json:"StyleCode,omitempty" description:"款号(格式是数字和字母)"`
	Ywy       string `json:"Ywy,omitempty" description:"业务员名称"`
}

type DyOrderListResponse struct {
	AuditTime         string `json:"AuditTime" description:"审核时间"`
	AuditUser         string `json:"AuditUser" description:"审核人"`
	BillDate          string `json:"BillDate" description:"单据日期"`
	CheckMemo         string `json:"CheckMemo" description:"核版描述"`
	CheckStatus       int    `json:"CheckStatus" description:"核版状态(0=未核版,1=未通过,2=通过)"`
	CreateTime        string `json:"CreateTime" description:"创建时间"`
	CreateUser        string `json:"CreateUser" description:"创建人"`
	CustomCheckStatus int    `json:"CustomCheckStatus" description:"客户确认(0=未核版,1=未通过,2=通过)"`
	Customer          string `json:"Customer" description:"客户"`
	CustomerMemo      string `json:"CustomerMemo" description:"客户要求"`
	DiffcultyName     string `json:"DiffcultyName" description:"难度系数名称"`
	DyBillCode        string `json:"DyBillCode" description:"打样单据号"`
	DyOrderID         string `json:"DyOrderID" description:"打样单ID(获取详情时用这个ID)"`
	DyUser            string `json:"DyUser" description:"打样师傅"`
	DyeMemo           string `json:"DyeMemo" description:"染色要求"`
	EndDate           string `json:"EndDate" description:"样品交期(2001-01-01)"`
	ExtendFile        string `json:"ExtendFile" description:"附件"`
	ExtendFileName    string `json:"ExtendFileName" description:"附件名称"`
	ExtendFileTime    string `json:"ExtendFileTime" description:"附件时间"`
	HasPrice          bool   `json:"HasPrice" description:"是否已报价"`
	Image             string `json:"Image" description:"图片"`
	IsAudit           bool   `json:"IsAudit" description:"是否审核"`
	IsDispatch        bool   `json:"IsDispatch" description:"是否派工"`
	IsLast            string `json:"IsLast" description:"是否最新"`
	IsSign            bool   `json:"IsSign" description:"是否签名"`
	JJ                string `json:"JJ" description:"是否紧急"`
	MLName            string `json:"MLName" description:"面料类型"`
	OrderCode         string `json:"OrderCode" description:"订单号"`
	PlanEndDate       string `json:"PlanEndDate" description:"研发计划交期"`
	ProductName       string `json:"ProductName" description:"款式名称"`
	ProductType       string `json:"ProductType" description:"产品类型"`
	Qty               int    `json:"Qty" description:"数量"`
	RowNo             int    `json:"RowNo" description:"数据ID"`
	ScanProcess       string `json:"ScanProcess" description:"最后扫描工序"`
	SignTime          string `json:"SignTime" description:"签名时间"`
	SignUser          string `json:"SignUser" description:"签名人"`
	SimpleTypeName    string `json:"SimpleTypeName" description:"打样类别名称"`
	Status            int    `json:"Status" description:"状态(0=草稿,1=已发布)"`
	StyleCode         string `json:"StyleCode" description:"款号"`
	StyleLevel        string `json:"StyleLevel" description:"款式难度"`
	SubmitTime        string `json:"SubmitTime" description:"发布时间"`
	TDCheckStatus     int    `json:"TDCheckStatus" description:"TD确认(0=未核版,1=未通过,2=通过)"`
	TDUser            string `json:"TDUser" description:"TD名称"`
	UpdateTime        string `json:"UpdateTime" description:"更新时间"`
	UpdateUser        string `json:"UpdateUser" description:"更新人"`
	WorkPlace         string `json:"WorkPlace" description:"产地"`
	YPLevel           int    `json:"YPLevel" description:"样品等级"`
	Ywy               string `json:"Ywy" description:"业务员"`
	YwyDepartment     string `json:"YwyDepartment" description:"业务员部门"`
	YwyDepartmentID   string `json:"YwyDepartmentID" description:"业务员部门ID"`
	YwyNo             string `json:"YwyNo" description:"业务员编号"`
}

// 打样单列表
func dyOrderList() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[DyOrderList, []DyOrderListResponse](openapi.OpenAPIToolSpec{
		Name:        "produce_dyOrder_list",
		Description: "打样单列表数据,必须传递开始日期(StartDate)和结束日期(EndDate)。如果要获取具体某一个打样单的详情请用返回结果数据中的DyOrderID去调用produce_dyOrder_info工具获取",
		Method:      "POST",
		Path:        "/produce/dy/order/list",
		PrepareParams: func(info *serv.UserInfo, params serv.Param) (serv.Param, error) {
			startDate, _ := params.Get[string]("StartDate")
			endDate, _ := params.Get[string]("EndDate")
			if startDate == "" || endDate == "" {
				return nil, errors.New("开始日期(StartDate)和结束日期(EndDate)必须传递")
			}
			return params, nil
		},
		Binding: openapi.OpenAPIRequestBinding{
			Path:      map[string]string(nil),
			Query:     map[string]string(nil),
			Body:      map[string]string(nil),
			BodyAll:   true,
			BodyParam: "",
		},
	})
}

type DyOrderInfoRequest struct {
	DyOrderID string `json:"dyOrderID" description:"打样单ID"`
}

type DyOrderInfoColorSize struct {
	Color     string `json:"Color" description:"颜色"`
	KnitColor string `json:"KnitColor" description:"提字色"`
	Qty       int    `json:"Qty" description:"数量"`
	Size      string `json:"Size" description:"尺码"`
	SortNo    int    `json:"SortNo" description:"排序"`
}

type DyOrderInfoCustomerMemoLogs struct {
	CreateTime   string `json:"CreateTime" description:"创建时间"`
	CreateUser   string `json:"CreateUser" description:"创建人"`
	CustomerMemo string `json:"CustomerMemo" description:"客户要求"`
}

type DyOrderInfoDyeCheckData struct {
	IsSucc     int    `json:"IsSucc" description:"是否正确(0=否，1=是)"`
	LevelMemo  string `json:"LevelMemo" description:"等级描述"`
	Question   string `json:"Question" description:"问题"`
	SortNo     int    `json:"SortNo" description:"序号"`
	UpdateTime string `json:"UpdateTime" description:"更新时间"`
	UpdateUser string `json:"UpdateUser" description:"更新人"`
}

type DyOrderInfoProducePacks struct {
	Color      string `json:"Color" description:"颜色"`
	DispatchNo string `json:"DispatchNo" description:"派工单号"`
	PackQrCode string `json:"PackQrCode" description:"小包号"`
}

type DyOrderInfoProduceWork struct {
	Employee   string  `json:"Employee" description:"扫描人"`
	Rate       float64 `json:"Rate" description:"倍率"`
	ScanTime   string  `json:"ScanTime" description:"扫描时间"`
	UpdateTime string  `json:"UpdateTime" description:"最后更新时间"`
	UpdateUser string  `json:"UpdateUser" description:"更新人"`
	WorkMmy    float64 `json:"WorkMmy" description:"工价"`
	WorkName   string  `json:"WorkName" description:"工序名称"`
	WorkSerial string  `json:"WorkSerial" description:"工序序号"`
	WorkTime   float64 `json:"WorkTime" description:"工时"`
}

type DyOrderInfoSewItems struct {
	ExtendVal1 string `json:"ExtendVal1" description:"扩展字段1"`
	ExtendVal2 string `json:"ExtendVal2" description:"扩展字段2"`
	ExtendVal3 string `json:"ExtendVal3" description:"扩展字段3"`
	ExtendVal4 string `json:"ExtendVal4" description:"扩展字段4"`
	ExtendVal5 string `json:"ExtendVal5" description:"扩展字段5"`
	ItemColor  string `json:"ItemColor" description:"辅料颜色"`
	ItemType   string `json:"ItemType" description:"辅料类型"`
	ItemVendor string `json:"ItemVendor" description:"辅料供应商"`
}

type DyOrderInfo struct {
	AuditTime        string                        `json:"AuditTime" description:"审核时间"`
	AuditUser        string                        `json:"AuditUser" description:"审核人"`
	BillDate         string                        `json:"BillDate" description:"单据日期"`
	ColorSize        []DyOrderInfoColorSize        `json:"ColorSize" description:"颜色尺码"`
	CreateTime       string                        `json:"CreateTime" description:"创建时间"`
	CreateUser       string                        `json:"CreateUser" description:"创建人"`
	Customer         string                        `json:"Customer" description:"客户"`
	CustomerMemo     string                        `json:"CustomerMemo" description:"客户要求"`
	CustomerMemoLogs []DyOrderInfoCustomerMemoLogs `json:"CustomerMemoLogs" description:"客户要求记录"`
	DiffcultyName    string                        `json:"DiffcultyName" description:"难度系数名称"`
	DyBillCode       string                        `json:"DyBillCode" description:"打样单据号"`
	DyOrderID        string                        `json:"DyOrderID" description:"打样单ID"`
	DyUser           string                        `json:"DyUser" description:"打样师傅"`
	DyeCheckData     []DyOrderInfoDyeCheckData     `json:"DyeCheckData" description:"染色师傅意见"`
	DyeMemo          string                        `json:"DyeMemo" description:"染色要求"`
	EndDate          string                        `json:"EndDate" description:"样品交期(2001-01-01)"`
	ExtendFile       string                        `json:"ExtendFile" description:"附件"`
	ExtendFileName   string                        `json:"ExtendFileName" description:"附件名称"`
	ExtendFileTime   string                        `json:"ExtendFileTime" description:"附件时间"`
	Image            string                        `json:"Image" description:"图片"`
	IsAudit          bool                          `json:"IsAudit" description:"是否审核"`
	IsDispatch       bool                          `json:"IsDispatch" description:"是否派工"`
	IsLast           string                        `json:"IsLast" description:"是否最新"`
	IsSign           bool                          `json:"IsSign" description:"是否签名"`
	MLName           string                        `json:"MLName" description:"面料类型"`
	Memo             string                        `json:"Memo" description:"备注"`
	OrderCode        string                        `json:"OrderCode" description:"订单号"`
	ProducePacks     []DyOrderInfoProducePacks     `json:"ProducePacks" description:"小包信息"`
	ProduceWork      []DyOrderInfoProduceWork      `json:"ProduceWork" description:"工序列表"`
	ProductName      string                        `json:"ProductName" description:"款式名称"`
	ProductType      string                        `json:"ProductType" description:"款式类型"`
	Qty              int                           `json:"Qty" description:"数量"`
	ScanProcess      string                        `json:"ScanProcess" description:"最后扫描工序"`
	SewItems         []DyOrderInfoSewItems         `json:"SewItems" description:"缝制辅料"`
	SewMemo          string                        `json:"SewMemo" description:"缝制工艺"`
	SignTime         string                        `json:"SignTime" description:"签名时间"`
	SignUser         string                        `json:"SignUser" description:"签名人"`
	SimpleTypeName   string                        `json:"SimpleTypeName" description:"打样类别名称"`
	StyleCode        string                        `json:"StyleCode" description:"款号"`
	SubmitTime       string                        `json:"SubmitTime" description:"发布时间"`
	UpdateTime       string                        `json:"UpdateTime" description:"更新时间"`
	UpdateUser       string                        `json:"UpdateUser" description:"更新人"`
	WorkPlace        string                        `json:"WorkPlace" description:"生产基地"`
	YPLevel          int                           `json:"YPLevel" description:"样品等级"`
	Ywy              string                        `json:"Ywy" description:"业务员"`
	YwyDepartment    string                        `json:"YwyDepartment" description:"业务员部门"`
	YwyNo            string                        `json:"YwyNo" description:"业务员编号"`
}

// dyOrderInfo 打样单详情
func dyOrderInfo() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[DyOrderInfoRequest, DyOrderInfo](openapi.OpenAPIToolSpec{
		Name:        "produce_dyOrder_info",
		Description: "打样单详情数据,查询时需要传递打样单ID可以从produce_dyOrder_list工具返回结果中获取",
		Method:      "GET",
		Path:        "/produce/dy/order/info",
		Binding: openapi.OpenAPIRequestBinding{
			Path:      map[string]string(nil),
			Query:     map[string]string{"dyOrderID": "dyOrderID"},
			Body:      map[string]string(nil),
			BodyAll:   false,
			BodyParam: "",
		},
	})
}

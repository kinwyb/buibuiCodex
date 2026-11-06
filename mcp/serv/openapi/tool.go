package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

// OpenAPIToolSpec 描述一个由 Swagger 2.0 生成的 MCP 工具。
type OpenAPIToolSpec struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Parameters     json.RawMessage
	ResponseSchema json.RawMessage
	Binding        OpenAPIRequestBinding
}

// OpenAPIRequestBinding 将 MCP 参数映射到 HTTP path、query 和 body。
type OpenAPIRequestBinding struct {
	Path      map[string]string
	Query     map[string]string
	Body      map[string]string
	BodyAll   bool
	BodyParam string
}

// OpenAPITool 使用统一运行时执行 Swagger operation。
type OpenAPITool struct {
	spec        OpenAPIToolSpec
	compact     CompactSchema
	responseErr error
}

// NewOpenAPITool 创建 Swagger 生成工具。
func NewOpenAPITool(spec OpenAPIToolSpec) *OpenAPITool {
	tool := &OpenAPITool{spec: spec}
	if len(spec.ResponseSchema) == 0 {
		tool.responseErr = errors.New("响应 schema 为空")
		return tool
	}
	if err := json.Unmarshal(spec.ResponseSchema, &tool.compact); err != nil {
		tool.responseErr = fmt.Errorf("解析响应 schema 失败: %w", err)
	}
	return tool
}

// Name 工具名称。
func (t *OpenAPITool) Name() string {
	return t.spec.Name
}

// Description 工具描述。
func (t *OpenAPITool) Description() string {
	return t.spec.Description
}

// Parameters 使用原始 schema，此方法仅用于满足 IMcpTool 接口。
func (t *OpenAPITool) Parameters() *types.ToolParams {
	return nil
}

// RawParameters 返回 Swagger operation 的原始参数 schema。
func (t *OpenAPITool) RawParameters() json.RawMessage {
	return t.spec.Parameters
}

// Execute 调用 edp 并转换业务数据。
func (t *OpenAPITool) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	if info == nil || info.Token == "" {
		return "", errors.New("用户 token 为空")
	}
	if t.responseErr != nil {
		return "", t.responseErr
	}

	req, err := t.spec.newRequest(info, params)
	if err != nil {
		return "", err
	}
	response, err := edp.DoClient[json.RawMessage](req)
	if err != nil {
		return "", err
	}
	return EncodeSchemaData(t.compact, response.Data, response.Page)
}

// TypedOpenAPITool 使用 Go 结构体作为响应类型，并通过通用 edp.DoClient 执行请求。
type TypedOpenAPITool[T any] struct {
	spec    OpenAPIToolSpec
	compact CompactSchema
}

// NewTypedOpenAPITool 创建强类型 Swagger 工具。
func NewTypedOpenAPITool[T any](spec OpenAPIToolSpec) *TypedOpenAPITool[T] {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	return &TypedOpenAPITool[T]{
		spec:    spec,
		compact: CompactSchemaFromType(typ),
	}
}

// NewTypedRequestOpenAPITool 使用 Go 结构体生成入参 schema，响应仍由 schema 字段转换。
func NewTypedRequestOpenAPITool[TRequest any](spec OpenAPIToolSpec) *OpenAPITool {
	spec.Parameters = JSONSchemaFromType(reflect.TypeOf((*TRequest)(nil)).Elem())
	return NewOpenAPITool(spec)
}

// NewTypedRequestResponseOpenAPITool 同时使用 Go 结构体生成入参和响应类型。
func NewTypedRequestResponseOpenAPITool[TRequest, TResponse any](spec OpenAPIToolSpec) *TypedOpenAPITool[TResponse] {
	spec.Parameters = JSONSchemaFromType(reflect.TypeOf((*TRequest)(nil)).Elem())
	return NewTypedOpenAPITool[TResponse](spec)
}

// Name 工具名称。
func (t *TypedOpenAPITool[T]) Name() string {
	return t.spec.Name
}

// Description 工具描述。
func (t *TypedOpenAPITool[T]) Description() string {
	return t.spec.Description
}

// Parameters 使用原始 schema，此方法仅用于满足 IMcpTool 接口。
func (t *TypedOpenAPITool[T]) Parameters() *types.ToolParams {
	return nil
}

// RawParameters 返回 Swagger operation 的原始参数 schema。
func (t *TypedOpenAPITool[T]) RawParameters() json.RawMessage {
	return t.spec.Parameters
}

// Execute 调用 edp 并按 Go 结构体转换响应。
func (t *TypedOpenAPITool[T]) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	if info == nil || info.Token == "" {
		return "", errors.New("用户 token 为空")
	}
	req, err := t.spec.newRequest(info, params)
	if err != nil {
		return "", err
	}
	response, err := edp.DoClient[T](req)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(response.Data)
	if err != nil {
		return "", fmt.Errorf("序列化响应结构体失败: %w", err)
	}
	return EncodeSchemaData(t.compact, data, response.Page)
}

func (s OpenAPIToolSpec) newRequest(info *serv.UserInfo, params serv.Param) (*edp.Request, error) {
	path, err := s.buildPath(params)
	if err != nil {
		return nil, err
	}
	query, err := s.buildQuery(params)
	if err != nil {
		return nil, err
	}
	body, err := s.buildBody(params)
	if err != nil {
		return nil, err
	}
	options := []edp.RequestOption{edp.WithToken(info.Token)}
	if len(query) > 0 {
		options = append(options, edp.WithQueryParams(query))
	}
	if body != nil {
		options = append(options, edp.WithBody(body))
	}
	return edp.NewRequest(strings.ToUpper(s.Method), path, options...), nil
}

func (s OpenAPIToolSpec) buildPath(params serv.Param) (string, error) {
	result := s.Path
	if s.Binding.Path != nil {
		for _, argName := range sortedKeys(s.Binding.Path) {
			apiName := s.Binding.Path[argName]
			value, ok := params[argName]
			if !ok || value == nil {
				return "", fmt.Errorf("缺少路径参数: %s", argName)
			}
			text, err := scalarString(value)
			if err != nil {
				return "", fmt.Errorf("路径参数 %s 转换失败: %w", argName, err)
			}
			result = strings.ReplaceAll(result, "{"+apiName+"}", url.PathEscape(text))
		}
		return validateBuiltPath(result)
	}
	return validateBuiltPath(result)
}

func validateBuiltPath(path string) (string, error) {
	if strings.Contains(path, "{") || strings.Contains(path, "}") {
		return "", fmt.Errorf("路径参数未完全替换: %s", path)
	}
	return path, nil
}

func (s OpenAPIToolSpec) buildQuery(params serv.Param) (map[string]string, error) {
	result := make(map[string]string)
	if s.Binding.Query != nil {
		for _, argName := range sortedKeys(s.Binding.Query) {
			value, ok := params[argName]
			if !ok || value == nil {
				continue
			}
			text, err := scalarString(value)
			if err != nil {
				return nil, fmt.Errorf("查询参数 %s 转换失败: %w", argName, err)
			}
			result[s.Binding.Query[argName]] = text
		}
	}
	return result, nil
}

func (s OpenAPIToolSpec) buildBody(params serv.Param) (any, error) {
	if s.Binding.BodyParam != "" {
		return params[s.Binding.BodyParam], nil
	}
	if s.Binding.Body != nil {
		if len(s.Binding.Body) == 0 {
			return nil, nil
		}
		body := make(map[string]any, len(s.Binding.Body))
		for _, argName := range sortedKeys(s.Binding.Body) {
			if value, ok := params[argName]; ok {
				body[s.Binding.Body[argName]] = value
			}
		}
		if len(body) == 0 {
			return nil, nil
		}
		return body, nil
	}
	if s.Binding.BodyAll {
		body := make(map[string]any, len(params))
		for name, value := range params {
			if _, ok := s.Binding.Path[name]; ok {
				continue
			}
			if _, ok := s.Binding.Query[name]; ok {
				continue
			}
			body[name] = value
		}
		if len(body) == 0 {
			return nil, nil
		}
		return body, nil
	}
	return nil, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func scalarString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		return fmt.Sprintf("%t", typed), nil
	case json.Number:
		return typed.String(), nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", typed), nil
	default:
		return "", fmt.Errorf("不支持复合类型 %T", value)
	}
}

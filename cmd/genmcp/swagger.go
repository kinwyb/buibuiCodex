package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

const (
	defaultSwaggerURL = "http://boda.bj.company/apidocs.json"
	defaultAPIBaseURL = "http://boda.bj.company/v1/boda"
)

var supportedMethods = []string{"get", "post", "put", "patch", "delete"}

type swaggerSpec struct {
	Swagger     string                     `json:"swagger"`
	BasePath    string                     `json:"basePath"`
	Consumes    []string                   `json:"consumes"`
	Info        swaggerInfo                `json:"info"`
	Definitions map[string]json.RawMessage `json:"definitions"`
	Parameters  map[string]json.RawMessage `json:"parameters"`
	Responses   map[string]json.RawMessage `json:"responses"`
	Paths       map[string]json.RawMessage `json:"paths"`
}

type swaggerInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type loadedSwagger struct {
	spec   *swaggerSpec
	sha256 string
}

type swaggerOperation struct {
	Method      string
	Path        string
	Summary     string
	Description string
	OperationID string
	Consumes    []string
	Parameters  []json.RawMessage
	Responses   *orderedJSON
}

type orderedEntry struct {
	Key   string
	Value json.RawMessage
}

type orderedJSON struct {
	raw     json.RawMessage
	entries []orderedEntry
	values  map[string]json.RawMessage
}

func (o *orderedJSON) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return errors.New("期望 JSON 对象")
	}

	o.raw = append(json.RawMessage(nil), data...)
	o.entries = nil
	o.values = make(map[string]json.RawMessage)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return errors.New("JSON 对象键不是字符串")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		o.entries = append(o.entries, orderedEntry{Key: key, Value: value})
		o.values[key] = value
	}
	_, err = decoder.Token()
	return err
}

func (o *orderedJSON) get(key string) (json.RawMessage, bool) {
	if o == nil {
		return nil, false
	}
	value, ok := o.values[key]
	return value, ok
}

func (o *orderedJSON) stringValue(key string) string {
	raw, ok := o.get(key)
	if !ok {
		return ""
	}
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}

func (o *orderedJSON) boolValue(key string) bool {
	raw, ok := o.get(key)
	if !ok {
		return false
	}
	var value bool
	_ = json.Unmarshal(raw, &value)
	return value
}

func (o *orderedJSON) stringSlice(key string) []string {
	raw, ok := o.get(key)
	if !ok {
		return nil
	}
	var values []string
	_ = json.Unmarshal(raw, &values)
	return values
}

func (o *orderedJSON) rawSlice(key string) []json.RawMessage {
	raw, ok := o.get(key)
	if !ok {
		return nil
	}
	var values []json.RawMessage
	_ = json.Unmarshal(raw, &values)
	return values
}

func loadSwagger(source string) (*loadedSwagger, error) {
	var (
		data []byte
		err  error
	)
	parsed, parseErr := url.Parse(source)
	if parseErr == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, requestErr := client.Get(source)
		if requestErr != nil {
			return nil, fmt.Errorf("拉取 Swagger 失败: %w", requestErr)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("拉取 Swagger 失败: HTTP %d", resp.StatusCode)
		}
		data, err = io.ReadAll(resp.Body)
	} else {
		data, err = os.ReadFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("读取 Swagger 失败: %w", err)
	}

	var spec swaggerSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		preview := data
		if len(preview) > 120 {
			preview = preview[:120]
		}
		return nil, fmt.Errorf("解析 Swagger JSON 失败: %w\n内容前缀: %s", err, string(preview))
	}
	if spec.Swagger != "2.0" {
		return nil, fmt.Errorf("仅支持 Swagger 2.0，当前版本为 %q", spec.Swagger)
	}
	if len(spec.Definitions) == 0 {
		return nil, errors.New("Swagger 中没有 definitions")
	}
	sum := sha256.Sum256(data)
	return &loadedSwagger{spec: &spec, sha256: hex.EncodeToString(sum[:])}, nil
}

func normalizeAPIPath(input, basePath string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("--api 不能为空")
	}
	parsed, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("解析 --api 失败: %w", err)
	}
	apiPath := parsed.Path
	if parsed.Scheme == "" && parsed.Host == "" && !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}
	apiPath = cleanPath(apiPath)
	basePath = cleanPath(basePath)
	if basePath != "" && basePath != "/" {
		if apiPath == basePath {
			apiPath = "/"
		} else if strings.HasPrefix(apiPath, basePath+"/") {
			apiPath = strings.TrimPrefix(apiPath, basePath)
		}
	}
	return cleanPath(apiPath), nil
}

func cleanPath(value string) string {
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	if len(value) > 1 {
		value = strings.TrimRight(value, "/")
	}
	return value
}

func selectOperation(loaded *loadedSwagger, apiPath, method string) (*swaggerOperation, error) {
	path, err := normalizeAPIPath(apiPath, loaded.spec.BasePath)
	if err != nil {
		return nil, err
	}
	rawPath, ok := loaded.spec.Paths[path]
	if !ok {
		return nil, fmt.Errorf("Swagger 中不存在路径 %s", path)
	}
	var pathItem orderedJSON
	if err := json.Unmarshal(rawPath, &pathItem); err != nil {
		return nil, fmt.Errorf("解析路径 %s 失败: %w", path, err)
	}

	method = strings.ToLower(strings.TrimSpace(method))
	if method != "" {
		raw, exists := pathItem.get(method)
		if !exists {
			return nil, fmt.Errorf("%s 不支持 %s", path, strings.ToUpper(method))
		}
		return parseOperation(path, method, raw, loaded.spec)
	}

	var available []string
	for _, candidate := range supportedMethods {
		if _, exists := pathItem.get(candidate); exists {
			available = append(available, candidate)
		}
	}
	switch len(available) {
	case 0:
		return nil, fmt.Errorf("%s 没有可生成的 HTTP 方法", path)
	case 1:
		raw, _ := pathItem.get(available[0])
		return parseOperation(path, available[0], raw, loaded.spec)
	default:
		return nil, fmt.Errorf("%s 存在多个方法 %s，请使用 --method 指定", path, strings.Join(available, "/"))
	}
}

func parseOperation(apiPath, method string, raw json.RawMessage, spec *swaggerSpec) (*swaggerOperation, error) {
	var node orderedJSON
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, fmt.Errorf("解析 %s %s 失败: %w", strings.ToUpper(method), apiPath, err)
	}
	op := &swaggerOperation{
		Method:      strings.ToUpper(method),
		Path:        apiPath,
		Summary:     strings.TrimSpace(node.stringValue("summary")),
		Description: strings.TrimSpace(node.stringValue("description")),
		OperationID: strings.TrimSpace(node.stringValue("operationId")),
		Consumes:    append([]string(nil), spec.Consumes...),
	}
	if consumes := node.stringSlice("consumes"); len(consumes) > 0 {
		op.Consumes = consumes
	}
	op.Parameters = node.rawSlice("parameters")
	if responsesRaw, ok := node.get("responses"); ok {
		var responses orderedJSON
		if err := json.Unmarshal(responsesRaw, &responses); err != nil {
			return nil, fmt.Errorf("解析响应定义失败: %w", err)
		}
		op.Responses = &responses
	}
	return op, nil
}

func operationDescription(op *swaggerOperation) string {
	parts := make([]string, 0, 2)
	if op.Summary != "" {
		parts = append(parts, op.Summary)
	}
	if op.Description != "" && op.Description != op.Summary {
		parts = append(parts, op.Description)
	}
	if len(parts) == 0 {
		return op.Method + " " + op.Path
	}
	return strings.Join(parts, "\n")
}

func operationName(op *swaggerOperation, override string) (string, error) {
	if override != "" {
		name := normalizeIdentifier(override)
		if name == "" {
			return "", fmt.Errorf("无效工具名 %q", override)
		}
		return name, nil
	}
	if op.OperationID != "" {
		if name := normalizeIdentifier(op.OperationID); name != "" {
			return name, nil
		}
	}
	segments := strings.Split(strings.Trim(op.Path, "/"), "/")
	if len(segments) == 0 || strings.Join(segments, "") == "" {
		return "", errors.New("无法从路径生成工具名，请使用 --name")
	}
	return normalizeIdentifier(strings.Join(segments, "_")), nil
}

func normalizeIdentifier(value string) string {
	value = camelToSnake(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastUnderscore = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
			lastUnderscore = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && builder.Len() > 0 {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return ""
	}
	if result[0] >= '0' && result[0] <= '9' {
		result = "api_" + result
	}
	return result
}

func camelToSnake(value string) string {
	var builder strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || nextLower {
					builder.WriteByte('_')
				}
			}
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

type requestBuild struct {
	Parameters  json.RawMessage
	Compact     openapi.CompactSchema
	PathParams  []string
	QueryParams []string
	BodyParam   string
	BodyFields  []string
}

func (r requestBuild) binding() openapi.OpenAPIRequestBinding {
	binding := openapi.OpenAPIRequestBinding{BodyParam: r.BodyParam}
	if len(r.PathParams) > 0 {
		binding.Path = make(map[string]string, len(r.PathParams))
		for _, name := range r.PathParams {
			binding.Path[name] = name
		}
	}
	if len(r.QueryParams) > 0 {
		binding.Query = make(map[string]string, len(r.QueryParams))
		for _, name := range r.QueryParams {
			binding.Query[name] = name
		}
	}
	if len(r.BodyFields) > 0 {
		binding.BodyAll = true
	}
	return binding
}

func buildRequestSchema(op *swaggerOperation, spec *swaggerSpec) (*requestBuild, error) {
	properties := make(map[string]json.RawMessage)
	propertyOrder := make([]string, 0)
	requiredSet := make(map[string]struct{})
	result := &requestBuild{}

	addProperty := func(name string, raw json.RawMessage, required bool) error {
		if name == "" {
			return errors.New("Swagger 参数缺少 name")
		}
		if _, exists := properties[name]; exists {
			return fmt.Errorf("参数名冲突: %s", name)
		}
		properties[name] = raw
		propertyOrder = append(propertyOrder, name)
		if required {
			requiredSet[name] = struct{}{}
		}
		return nil
	}

	bodyRaw, body, err := resolveSingleBodyParameter(op.Parameters, spec)
	if err != nil {
		return nil, err
	}
	for _, raw := range op.Parameters {
		parameter, err := resolveParameter(raw, spec)
		if err != nil {
			return nil, err
		}
		location := parameter.stringValue("in")
		name := parameter.stringValue("name")
		switch location {
		case "path", "query":
			typ := parameter.stringValue("type")
			if typ == "array" || parameterHasItems(parameter) {
				return nil, fmt.Errorf("%s 参数 %s 使用数组，当前版本不支持", location, name)
			}
			if typ == "file" {
				return nil, fmt.Errorf("%s 参数 %s 是文件，当前版本不支持", location, name)
			}
			schema, err := buildScalarParameterSchema(&parameter)
			if err != nil {
				return nil, fmt.Errorf("%s 参数 %s: %w", location, name, err)
			}
			if err := addProperty(name, schema, parameter.boolValue("required")); err != nil {
				return nil, err
			}
			if location == "path" {
				result.PathParams = append(result.PathParams, name)
				requiredSet[name] = struct{}{}
			} else {
				result.QueryParams = append(result.QueryParams, name)
			}
		case "header":
			if !strings.EqualFold(name, "token") {
				return nil, fmt.Errorf("header 参数 %s 当前版本不支持", name)
			}
		case "formData":
			return nil, fmt.Errorf("formData 参数 %s 当前版本不支持", name)
		case "body":
			// body 在循环外统一处理。
		default:
			return nil, fmt.Errorf("参数 %s 的 in=%q 不受支持", name, location)
		}
	}

	if len(bodyRaw) > 0 {
		if !supportsJSON(op.Consumes) {
			return nil, fmt.Errorf("body 接口只支持 JSON consumes，当前为 %v", op.Consumes)
		}
		shape, err := objectShape(bodyRaw, spec.Definitions)
		if err != nil {
			return nil, fmt.Errorf("解析 body 参数失败: %w", err)
		}
		if shape.ok {
			for _, entry := range shape.properties.entries {
				required := containsString(shape.required, entry.Key)
				if err := addProperty(entry.Key, entry.Value, required); err != nil {
					return nil, fmt.Errorf("body 字段 %s: %w", entry.Key, err)
				}
				result.BodyFields = append(result.BodyFields, entry.Key)
			}
		} else {
			result.BodyParam = "body"
			if err := addProperty("body", bodyRaw, body.boolValue("required")); err != nil {
				return nil, err
			}
		}
	}

	root := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(requiredSet) > 0 {
		required := make([]string, 0, len(requiredSet))
		for _, name := range propertyOrder {
			if _, ok := requiredSet[name]; ok {
				required = append(required, name)
			}
		}
		root["required"] = required
	}
	definitions := collectDefinitionClosure(properties, spec.Definitions)
	if len(definitions) > 0 {
		root["definitions"] = definitions
	}
	encoded, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("序列化参数 schema 失败: %w", err)
	}
	result.Parameters = encoded
	var schemaRoot struct {
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(encoded, &schemaRoot); err != nil {
		return nil, fmt.Errorf("解析参数 schema 失败: %w", err)
	}
	converter := schemaConverter{definitions: schemaRoot.Definitions}
	result.Compact, err = converter.convert(encoded, make(map[string]bool))
	if err != nil {
		return nil, fmt.Errorf("转换参数 schema 失败: %w", err)
	}
	return result, nil
}

func resolveSingleBodyParameter(parameters []json.RawMessage, spec *swaggerSpec) (json.RawMessage, orderedJSON, error) {
	var (
		bodyRaw json.RawMessage
		body    orderedJSON
	)
	for _, raw := range parameters {
		parameter, err := resolveParameter(raw, spec)
		if err != nil {
			return nil, orderedJSON{}, err
		}
		if parameter.stringValue("in") != "body" {
			continue
		}
		if len(bodyRaw) > 0 {
			return nil, orderedJSON{}, errors.New("Swagger 2.0 只允许一个 body 参数")
		}
		schema, ok := parameter.get("schema")
		if !ok {
			return nil, orderedJSON{}, errors.New("body 参数缺少 schema")
		}
		bodyRaw = schema
		body = parameter
	}
	return bodyRaw, body, nil
}

func resolveParameter(raw json.RawMessage, spec *swaggerSpec) (orderedJSON, error) {
	var parameter orderedJSON
	if err := json.Unmarshal(raw, &parameter); err != nil {
		return parameter, err
	}
	ref := parameter.stringValue("$ref")
	if ref == "" {
		return parameter, nil
	}
	name, err := definitionName(ref)
	if err != nil || !strings.HasPrefix(ref, "#/parameters/") {
		return parameter, fmt.Errorf("不支持的参数引用 %s", ref)
	}
	target, ok := spec.Parameters[name]
	if !ok {
		return parameter, fmt.Errorf("参数引用不存在: %s", ref)
	}
	if err := json.Unmarshal(target, &parameter); err != nil {
		return parameter, err
	}
	return parameter, nil
}

func parameterHasItems(parameter orderedJSON) bool {
	_, ok := parameter.get("items")
	return ok
}

func buildScalarParameterSchema(parameter *orderedJSON) (json.RawMessage, error) {
	allowed := map[string]struct{}{
		"type": {}, "format": {}, "description": {}, "enum": {}, "default": {},
		"maximum": {}, "minimum": {}, "exclusiveMaximum": {}, "exclusiveMinimum": {},
		"maxLength": {}, "minLength": {}, "pattern": {},
	}
	result := make(map[string]json.RawMessage)
	for _, entry := range parameter.entries {
		if _, ok := allowed[entry.Key]; ok {
			result[entry.Key] = entry.Value
		}
	}
	if _, ok := result["type"]; !ok {
		result["type"] = json.RawMessage(`"string"`)
	}
	if _, ok := result["description"]; !ok {
		if description := parameter.stringValue("description"); description != "" {
			encoded, _ := json.Marshal(description)
			result["description"] = encoded
		}
	}
	for _, key := range []string{"name", "in", "required", "schema", "collectionFormat"} {
		delete(result, key)
	}
	return json.Marshal(result)
}

func supportsJSON(consumes []string) bool {
	if len(consumes) == 0 {
		return true
	}
	for _, value := range consumes {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "json") || lower == "*/*" {
			return true
		}
	}
	return false
}

type shapeResult struct {
	ok         bool
	properties *orderedJSON
	required   []string
}

func objectShape(raw json.RawMessage, definitions map[string]json.RawMessage) (shapeResult, error) {
	return objectShapeWithStack(raw, definitions, make(map[string]bool))
}

func objectShapeWithStack(raw json.RawMessage, definitions map[string]json.RawMessage, stack map[string]bool) (shapeResult, error) {
	var node orderedJSON
	if err := json.Unmarshal(raw, &node); err != nil {
		return shapeResult{}, err
	}
	if ref := node.stringValue("$ref"); ref != "" {
		name, err := definitionName(ref)
		if err != nil {
			return shapeResult{}, err
		}
		if stack[name] {
			return shapeResult{ok: false}, nil
		}
		target, ok := definitions[name]
		if !ok {
			return shapeResult{}, fmt.Errorf("definition 不存在: %s", name)
		}
		stack[name] = true
		result, err := objectShapeWithStack(target, definitions, stack)
		delete(stack, name)
		return result, err
	}

	result := shapeResult{ok: true, properties: &orderedJSON{values: make(map[string]json.RawMessage)}}
	requiredSet := make(map[string]struct{})
	allOf := node.rawSlice("allOf")
	for _, member := range allOf {
		memberShape, err := objectShapeWithStack(member, definitions, stack)
		if err != nil {
			return shapeResult{}, err
		}
		if !memberShape.ok {
			return shapeResult{ok: false}, nil
		}
		mergeOrdered(result.properties, memberShape.properties)
		for _, name := range memberShape.required {
			requiredSet[name] = struct{}{}
		}
	}

	typ := node.stringValue("type")
	if typ != "" && typ != "object" {
		return shapeResult{ok: false}, nil
	}
	if propertiesRaw, ok := node.get("properties"); ok {
		var properties orderedJSON
		if err := json.Unmarshal(propertiesRaw, &properties); err != nil {
			return shapeResult{}, err
		}
		mergeOrdered(result.properties, &properties)
	}
	for _, name := range node.stringSlice("required") {
		requiredSet[name] = struct{}{}
	}
	if len(result.properties.entries) == 0 {
		return shapeResult{ok: false}, nil
	}
	for _, entry := range result.properties.entries {
		if _, ok := requiredSet[entry.Key]; ok {
			result.required = append(result.required, entry.Key)
		}
	}
	return result, nil
}

func mergeOrdered(target, source *orderedJSON) {
	if source == nil {
		return
	}
	for _, entry := range source.entries {
		if _, exists := target.values[entry.Key]; exists {
			continue
		}
		target.values[entry.Key] = entry.Value
		target.entries = append(target.entries, entry)
	}
}

func collectDefinitionClosure(properties map[string]json.RawMessage, definitions map[string]json.RawMessage) map[string]json.RawMessage {
	roots := make([]json.RawMessage, 0, len(properties))
	for _, raw := range properties {
		roots = append(roots, raw)
	}
	queue := make([]string, 0)
	addRefs := func(raw json.RawMessage) {
		var value any
		if json.Unmarshal(raw, &value) != nil {
			return
		}
		collectRefsFromValue(value, &queue)
	}
	for _, raw := range roots {
		addRefs(raw)
	}
	result := make(map[string]json.RawMessage)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, exists := result[name]; exists {
			continue
		}
		raw, ok := definitions[name]
		if !ok {
			continue
		}
		result[name] = raw
		addRefs(raw)
	}
	return result
}

func collectRefsFromValue(value any, refs *[]string) {
	switch typed := value.(type) {
	case map[string]any:
		if ref, ok := typed["$ref"].(string); ok {
			if name, err := definitionName(ref); err == nil {
				*refs = append(*refs, name)
			}
		}
		for _, child := range typed {
			collectRefsFromValue(child, refs)
		}
	case []any:
		for _, child := range typed {
			collectRefsFromValue(child, refs)
		}
	}
}

func buildResponseSchema(op *swaggerOperation, spec *swaggerSpec) (openapi.CompactSchema, error) {
	schemaRaw, err := selectResponseSchema(op, spec)
	if err != nil {
		return openapi.CompactSchema{}, err
	}
	converter := schemaConverter{definitions: spec.Definitions}
	return converter.convert(schemaRaw, make(map[string]bool))
}

func selectResponseSchema(op *swaggerOperation, spec *swaggerSpec) (json.RawMessage, error) {
	if op.Responses == nil {
		return nil, errors.New("接口没有 responses")
	}
	raw, ok := op.Responses.get("200")
	if !ok {
		keys := make([]string, 0)
		for _, entry := range op.Responses.entries {
			if strings.HasPrefix(entry.Key, "2") {
				keys = append(keys, entry.Key)
			}
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			return nil, errors.New("接口没有 2xx 响应")
		}
		raw = op.Responses.values[keys[0]]
	}
	var response orderedJSON
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	if ref := response.stringValue("$ref"); ref != "" {
		name, err := definitionName(ref)
		if err != nil || !strings.HasPrefix(ref, "#/responses/") {
			return nil, fmt.Errorf("不支持的响应引用 %s", ref)
		}
		target, ok := spec.Responses[name]
		if !ok {
			return nil, fmt.Errorf("响应引用不存在: %s", ref)
		}
		if err := json.Unmarshal(target, &response); err != nil {
			return nil, err
		}
	}
	schema, ok := response.get("schema")
	if !ok {
		return nil, errors.New("2xx 响应没有 schema")
	}
	return schema, nil
}

type schemaConverter struct {
	definitions map[string]json.RawMessage
}

func (c schemaConverter) convert(raw json.RawMessage, stack map[string]bool) (openapi.CompactSchema, error) {
	var node orderedJSON
	if err := json.Unmarshal(raw, &node); err != nil {
		return openapi.CompactSchema{}, err
	}
	description := node.stringValue("description")
	if description == "" {
		description = node.stringValue("title")
	}
	if ref := node.stringValue("$ref"); ref != "" {
		name, err := definitionName(ref)
		if err != nil {
			return openapi.CompactSchema{}, err
		}
		if stack[name] {
			return openapi.CompactSchema{Kind: "raw", Description: description}, nil
		}
		target, ok := c.definitions[name]
		if !ok {
			return openapi.CompactSchema{}, fmt.Errorf("definition 不存在: %s", name)
		}
		stack[name] = true
		result, err := c.convert(target, stack)
		delete(stack, name)
		if result.Description == "" {
			result.Description = description
		}
		return result, err
	}
	if _, ok := node.get("oneOf"); ok {
		return openapi.CompactSchema{Kind: "raw", Description: description}, nil
	}
	if _, ok := node.get("anyOf"); ok {
		return openapi.CompactSchema{Kind: "raw", Description: description}, nil
	}

	typ := node.stringValue("type")
	if _, hasAllOf := node.get("allOf"); hasAllOf || typ == "object" || (typ == "" && hasAny(node, "properties", "additionalProperties")) {
		shape, err := objectShape(raw, c.definitions)
		if err != nil {
			return openapi.CompactSchema{}, err
		}
		if shape.ok {
			result := openapi.CompactSchema{Kind: "object", Description: description}
			for _, entry := range shape.properties.entries {
				fieldType, err := c.convert(entry.Value, stack)
				if err != nil {
					return openapi.CompactSchema{}, fmt.Errorf("字段 %s: %w", entry.Key, err)
				}
				result.Fields = append(result.Fields, openapi.CompactField{
					Name:     entry.Key,
					Required: containsString(shape.required, entry.Key),
					Type:     fieldType,
				})
			}
			return result, nil
		}
	}
	if typ == "" && hasAny(node, "properties", "additionalProperties") {
		typ = "object"
	}

	switch typ {
	case "string", "integer", "number", "boolean":
		return openapi.CompactSchema{Kind: typ, Description: description}, nil
	case "array":
		result := openapi.CompactSchema{Kind: "array", Description: description}
		items, ok := node.get("items")
		if !ok {
			result.Items = &openapi.CompactSchema{Kind: "raw"}
			return result, nil
		}
		itemSchema, err := c.convert(items, stack)
		if err != nil {
			return openapi.CompactSchema{}, err
		}
		result.Items = &itemSchema
		return result, nil
	case "object":
		if additional, ok := node.get("additionalProperties"); ok {
			result := openapi.CompactSchema{Kind: "map", Description: description}
			if bytes.Equal(bytes.TrimSpace(additional), []byte("true")) {
				result.Values = &openapi.CompactSchema{Kind: "raw"}
			} else if !bytes.Equal(bytes.TrimSpace(additional), []byte("false")) {
				valueSchema, err := c.convert(additional, stack)
				if err != nil {
					return openapi.CompactSchema{}, err
				}
				result.Values = &valueSchema
			}
			return result, nil
		}
		return openapi.CompactSchema{Kind: "raw", Description: description}, nil
	default:
		return openapi.CompactSchema{Kind: "raw", Description: description}, nil
	}
}

func definitionName(ref string) (string, error) {
	const prefix = "#/definitions/"
	switch {
	case strings.HasPrefix(ref, prefix):
		return strings.TrimPrefix(ref, prefix), nil
	case strings.HasPrefix(ref, "#/parameters/"):
		return strings.TrimPrefix(ref, "#/parameters/"), nil
	case strings.HasPrefix(ref, "#/responses/"):
		return strings.TrimPrefix(ref, "#/responses/"), nil
	default:
		return "", fmt.Errorf("仅支持本地引用: %s", ref)
	}
}

func hasAny(node orderedJSON, keys ...string) bool {
	for _, key := range keys {
		if _, ok := node.get(key); ok {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

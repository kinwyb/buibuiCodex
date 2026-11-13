package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

const (
	schemaDataFormat       = "schema-data/v1"
	schemaDataInlineLimit  = 5 * 1024
	schemaDataPreviewLimit = 3
)

// CompactSchema 是生成阶段固化、运行阶段用于转换数据的精简 schema。
type CompactSchema struct {
	Kind        string         `json:"kind"`
	Description string         `json:"description,omitempty"`
	Enum        []string       `json:"enum,omitempty"`
	Fields      []CompactField `json:"fields,omitempty"`
	Items       *CompactSchema `json:"items,omitempty"`
	Values      *CompactSchema `json:"values,omitempty"`
}

// CompactField 描述对象中的一个字段及其在位置数组中的顺序。
type CompactField struct {
	Name     string        `json:"name"`
	Required bool          `json:"required"`
	Type     CompactSchema `json:"type"`
}

type schemaDataEnvelope struct {
	Format string          `json:"format"`
	Layout map[string]any  `json:"layout"`
	Schema any             `json:"schema"`
	Data   any             `json:"data"`
	Page   any             `json:"page,omitempty"`
	Meta   *schemaDataMeta `json:"meta,omitempty"`
}

type schemaObjectDataEnvelope struct {
	Data any `json:"data"`
	Page any `json:"page,omitempty"`
}

type schemaDataMeta struct {
	Total    int    `json:"total"`
	Shown    int    `json:"shown"`
	File     string `json:"file"`
	FileHint string `json:"fileHint"`
}

type schemaNamedField struct {
	Name  string
	Value any
}

type schemaNamedObject []schemaNamedField

func (o schemaNamedObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, field := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		name, err := json.Marshal(field.Name)
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		buf.WriteByte(':')
		value, err := json.Marshal(field.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// EncodeSchemaData 将 API JSON 转换为紧凑结构。
// 根数据是数组时使用 schema 加位置数组；根数据是对象时使用字段说明作为 key。
// 对象结果超过 5KB 且包含重复对象结构时，会回退到位置数组并取更短的结果。
// 根数组超过 5KB 时会写入临时文件，只保留 schema 和前三条数据。
func EncodeSchemaData(schema CompactSchema, data, page json.RawMessage) (string, error) {
	schema = cloneCompactSchema(schema)
	if err := prepareCompactSchema(&schema, data); err != nil {
		return "", err
	}

	pageValue, err := decodeJSONValue(page)
	if err != nil {
		return "", fmt.Errorf("解析分页信息失败: %w", err)
	}
	if schema.Kind == "object" {
		return encodeObjectSchemaData(&schema, data, pageValue)
	}

	fullData, err := encodeSchemaValue(&schema, data)
	if err != nil {
		return "", err
	}
	full := schemaDataEnvelope{
		Format: schemaDataFormat,
		Layout: schemaDataLayout(),
		Schema: compactSchemaWire(&schema),
		Data:   fullData,
		Page:   pageValue,
	}
	fullBytes, err := json.Marshal(full)
	if err != nil {
		return "", fmt.Errorf("序列化紧凑数据失败: %w", err)
	}
	if len(fullBytes) < schemaDataInlineLimit {
		return string(fullBytes), nil
	}
	if schema.Kind != "array" {
		return string(fullBytes), nil
	}

	fileBytes, err := json.Marshal(fullData)
	if err != nil {
		return "", fmt.Errorf("序列化完整数据失败: %w", err)
	}
	fileURL, err := writeSchemaDataFile(fileBytes)
	if err != nil {
		return string(fullBytes), nil
	}

	previewData, total, shown, err := previewSchemaData(&schema, data)
	if err != nil {
		return "", err
	}
	preview := schemaDataEnvelope{
		Format: schemaDataFormat,
		Layout: schemaDataLayout(),
		Schema: compactSchemaWire(&schema),
		Data:   previewData,
		Page:   pageValue,
		Meta: &schemaDataMeta{
			Total:    total,
			Shown:    shown,
			File:     fileURL,
			FileHint: "文件内容是上方 data 数组；请用脚本或代码筛选、聚合、分页读取，只加载回答所需的字段和记录，不要一次性读取完整文件",
		},
	}
	previewBytes, err := json.Marshal(preview)
	if err != nil {
		return "", fmt.Errorf("序列化预览数据失败: %w", err)
	}
	return string(previewBytes), nil
}

func encodeObjectSchemaData(schema *CompactSchema, data json.RawMessage, page any) (string, error) {
	namedData, err := encodeNamedSchemaValue(schema, data)
	if err != nil {
		return "", err
	}
	namedBytes, err := json.Marshal(schemaObjectDataEnvelope{Data: namedData, Page: page})
	if err != nil {
		return "", fmt.Errorf("序列化对象数据失败: %w", err)
	}
	if len(namedBytes) < schemaDataInlineLimit || !hasRepeatedObject(schema) {
		return string(namedBytes), nil
	}

	fullData, err := encodeSchemaValue(schema, data)
	if err != nil {
		return "", err
	}
	positionalBytes, err := json.Marshal(schemaDataEnvelope{
		Format: schemaDataFormat,
		Layout: schemaDataLayout(),
		Schema: compactSchemaWire(schema),
		Data:   fullData,
		Page:   page,
	})
	if err != nil {
		return "", fmt.Errorf("序列化紧凑数据失败: %w", err)
	}
	if len(positionalBytes) < len(namedBytes) {
		return string(positionalBytes), nil
	}
	return string(namedBytes), nil
}

func encodeNamedSchemaValue(schema *CompactSchema, raw json.RawMessage) (any, error) {
	if schema == nil || len(bytes.TrimSpace(raw)) == 0 || isJSONNull(raw) {
		return nil, nil
	}
	switch schema.Kind {
	case "object":
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, fmt.Errorf("解析对象数据失败: %w", err)
		}
		keys := namedObjectKeys(schema.Fields)
		result := make(schemaNamedObject, 0, len(schema.Fields))
		for i := range schema.Fields {
			field := &schema.Fields[i]
			value, err := encodeNamedSchemaValue(&field.Type, object[field.Name])
			if err != nil {
				return nil, fmt.Errorf("字段 %s 转换失败: %w", field.Name, err)
			}
			result = append(result, schemaNamedField{Name: keys[i], Value: value})
		}
		return result, nil
	case "array":
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("解析数组数据失败: %w", err)
		}
		result := make([]any, 0, len(items))
		for _, item := range items {
			value, err := encodeNamedSchemaValue(schema.Items, item)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case "map":
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, fmt.Errorf("解析字典数据失败: %w", err)
		}
		result := make(map[string]any, len(values))
		for key, item := range values {
			value, err := encodeNamedSchemaValue(schema.Values, item)
			if err != nil {
				return nil, fmt.Errorf("字典键 %s 转换失败: %w", key, err)
			}
			result[key] = value
		}
		return result, nil
	default:
		return encodeSchemaValue(schema, raw)
	}
}

func namedObjectKeys(fields []CompactField) []string {
	keys := make([]string, len(fields))
	names := make(map[string]struct{}, len(fields))
	counts := make(map[string]int, len(fields))
	for i := range fields {
		names[fields[i].Name] = struct{}{}
		key := strings.TrimSpace(fields[i].Type.Description)
		if key == "" {
			key = fields[i].Name
		}
		keys[i] = key
		if key != fields[i].Name {
			counts[key]++
		}
	}
	for i := range fields {
		if keys[i] == fields[i].Name {
			continue
		}
		if counts[keys[i]] > 1 {
			keys[i] = fields[i].Name
			continue
		}
		if _, exists := names[keys[i]]; exists {
			keys[i] = fields[i].Name
		}
	}
	return keys
}

func hasRepeatedObject(schema *CompactSchema) bool {
	if schema == nil {
		return false
	}
	switch schema.Kind {
	case "object":
		for i := range schema.Fields {
			if hasRepeatedObject(&schema.Fields[i].Type) {
				return true
			}
		}
	case "array":
		if schema.Items == nil {
			return false
		}
		return schema.Items.Kind == "object" || hasRepeatedObject(schema.Items)
	case "map":
		if schema.Values == nil {
			return false
		}
		return schema.Values.Kind == "object" || hasRepeatedObject(schema.Values)
	}
	return false
}

func schemaDataLayout() map[string]any {
	return map[string]any{
		"field": []string{"name", "required", "type"},
		"type":  []string{"kind", "description", "children?"},
	}
}

func compactSchemaWire(schema *CompactSchema) any {
	wire := []any{schema.Kind, schema.Description}
	switch schema.Kind {
	case "object":
		fields := make([]any, 0, len(schema.Fields))
		for i := range schema.Fields {
			field := &schema.Fields[i]
			required := 0
			if field.Required {
				required = 1
			}
			fields = append(fields, []any{field.Name, required, compactSchemaWire(&field.Type)})
		}
		wire = append(wire, fields)
	case "array":
		if schema.Items != nil {
			wire = append(wire, compactSchemaWire(schema.Items))
		}
	case "map":
		if schema.Values != nil {
			wire = append(wire, compactSchemaWire(schema.Values))
		}
	}
	return wire
}

func prepareCompactSchema(schema *CompactSchema, raw json.RawMessage) error {
	if schema == nil || isJSONNull(raw) {
		return nil
	}
	switch schema.Kind {
	case "object":
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return fmt.Errorf("解析对象数据失败: %w", err)
		}
		for i := range schema.Fields {
			field := &schema.Fields[i]
			if value, ok := object[field.Name]; ok {
				if err := prepareCompactSchema(&field.Type, value); err != nil {
					return err
				}
			}
		}
		for _, name := range extraObjectKeys(object, schema.Fields) {
			schema.Fields = append(schema.Fields, CompactField{
				Name: name,
				Type: CompactSchema{Kind: "raw"},
			})
		}
	case "array":
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return fmt.Errorf("解析数组数据失败: %w", err)
		}
		if schema.Items != nil {
			for _, item := range items {
				if err := prepareCompactSchema(schema.Items, item); err != nil {
					return err
				}
			}
		}
	case "map":
		if schema.Values == nil {
			return nil
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return fmt.Errorf("解析字典数据失败: %w", err)
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := prepareCompactSchema(schema.Values, values[key]); err != nil {
				return err
			}
		}
	}
	return nil
}

func extraObjectKeys(object map[string]json.RawMessage, fields []CompactField) []string {
	known := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		known[field.Name] = struct{}{}
	}
	extras := make([]string, 0)
	for name := range object {
		if _, ok := known[name]; !ok {
			extras = append(extras, name)
		}
	}
	sort.Strings(extras)
	return extras
}

func encodeSchemaValue(schema *CompactSchema, raw json.RawMessage) (any, error) {
	if schema == nil || len(bytes.TrimSpace(raw)) == 0 || isJSONNull(raw) {
		return nil, nil
	}
	switch schema.Kind {
	case "object":
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, fmt.Errorf("解析对象数据失败: %w", err)
		}
		result := make([]any, 0, len(schema.Fields))
		for i := range schema.Fields {
			field := &schema.Fields[i]
			value, err := encodeSchemaValue(&field.Type, object[field.Name])
			if err != nil {
				return nil, fmt.Errorf("字段 %s 转换失败: %w", field.Name, err)
			}
			result = append(result, value)
		}
		return result, nil
	case "array":
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("解析数组数据失败: %w", err)
		}
		result := make([]any, 0, len(items))
		for _, item := range items {
			value, err := encodeSchemaValue(schema.Items, item)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case "map":
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, fmt.Errorf("解析字典数据失败: %w", err)
		}
		result := make(map[string]any, len(values))
		for key, item := range values {
			value, err := encodeSchemaValue(schema.Values, item)
			if err != nil {
				return nil, fmt.Errorf("字典键 %s 转换失败: %w", key, err)
			}
			result[key] = value
		}
		return result, nil
	case "raw":
		return decodeJSONValue(raw)
	case "string":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("值不是字符串: %w", err)
		}
		return value, nil
	case "boolean":
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("值不是布尔类型: %w", err)
		}
		return value, nil
	case "integer":
		value, err := decodeJSONValue(raw)
		if err != nil {
			return nil, err
		}
		number, ok := value.(json.Number)
		if !ok {
			return nil, errors.New("值不是整数")
		}
		if _, err := strconv.ParseInt(number.String(), 10, 64); err != nil {
			return nil, fmt.Errorf("值不是整数: %w", err)
		}
		return number, nil
	case "number":
		value, err := decodeJSONValue(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := value.(json.Number); !ok {
			return nil, errors.New("值不是数字")
		}
		return value, nil
	default:
		return decodeJSONValue(raw)
	}
}

func decodeJSONValue(raw json.RawMessage) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func previewSchemaData(schema *CompactSchema, data json.RawMessage) (any, int, int, error) {
	if schema.Kind != "array" {
		return nil, 0, 0, errors.New("只有数组数据支持文件预览")
	}

	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, 0, 0, fmt.Errorf("解析数组数据失败: %w", err)
	}
	total := len(items)
	shown := min(total, schemaDataPreviewLimit)
	result := make([]any, 0, shown)
	for _, item := range items[:shown] {
		value, err := encodeSchemaValue(schema.Items, item)
		if err != nil {
			return nil, 0, 0, err
		}
		result = append(result, value)
	}
	return result, total, shown, nil
}

func writeSchemaDataFile(data []byte) (string, error) {
	if serv.TmpDir == "" {
		execPath, err := os.Executable()
		if err != nil {
			return "", err
		}
		serv.TmpDir = filepath.Join(filepath.Dir(execPath), "tmp")
	}
	date := time.Now().Format(time.DateOnly)
	fileDir := filepath.Join(serv.TmpDir, date)
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%d.json", time.Now().UnixNano())
	if err := os.WriteFile(filepath.Join(fileDir, fileName), data, 0644); err != nil {
		return "", err
	}
	return serv.TmpUrl + "?file=" + strings.ReplaceAll(date, "-", "") + fileName, nil
}

func cloneCompactSchema(schema CompactSchema) CompactSchema {
	clone := schema
	clone.Fields = append([]CompactField(nil), schema.Fields...)
	for i := range clone.Fields {
		clone.Fields[i].Type = cloneCompactSchema(clone.Fields[i].Type)
	}
	if schema.Items != nil {
		item := cloneCompactSchema(*schema.Items)
		clone.Items = &item
	}
	if schema.Values != nil {
		value := cloneCompactSchema(*schema.Values)
		clone.Values = &value
	}
	return clone
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

package openapi

import (
	"encoding/json"
	"reflect"
	"strings"
)

// CompactSchemaFromType 根据 Go 类型的 json 和 description 标签生成紧凑 schema。
// 与生成器保持一致：没有 omitempty 的非指针字段默认必填，required 标签可以覆盖。
func CompactSchemaFromType(typ reflect.Type) CompactSchema {
	return compactSchemaFromType(typ, make(map[reflect.Type]bool))
}

// JSONSchemaFromType 根据 Go 类型生成 MCP 工具参数 JSON Schema。
func JSONSchemaFromType(typ reflect.Type) json.RawMessage {
	schema := compactSchemaFromType(typ, make(map[reflect.Type]bool))
	raw, _ := json.Marshal(compactSchemaJSONSchema(schema))
	return raw
}

func compactSchemaFromType(typ reflect.Type, stack map[reflect.Type]bool) CompactSchema {
	if typ == nil {
		return CompactSchema{Kind: "raw"}
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeOf(json.RawMessage{}) {
		return CompactSchema{Kind: "raw"}
	}
	if typ.PkgPath() == "time" && typ.Name() == "Time" {
		return CompactSchema{Kind: "string"}
	}
	switch typ.Kind() {
	case reflect.String:
		return CompactSchema{Kind: "string"}
	case reflect.Bool:
		return CompactSchema{Kind: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return CompactSchema{Kind: "integer"}
	case reflect.Float32, reflect.Float64:
		return CompactSchema{Kind: "number"}
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return CompactSchema{Kind: "string"}
		}
		item := compactSchemaFromType(typ.Elem(), stack)
		return CompactSchema{Kind: "array", Items: &item}
	case reflect.Map:
		valueType := typ.Elem()
		value := compactSchemaFromType(valueType, stack)
		return CompactSchema{Kind: "map", Values: &value}
	case reflect.Interface:
		return CompactSchema{Kind: "raw"}
	case reflect.Struct:
		if stack[typ] {
			return CompactSchema{Kind: "raw"}
		}
		stack[typ] = true
		defer delete(stack, typ)
		result := CompactSchema{Kind: "object"}
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			if field.PkgPath != "" {
				continue
			}
			if field.Anonymous && field.Tag.Get("json") == "" {
				valueType := field.Type
				for valueType.Kind() == reflect.Pointer {
					valueType = valueType.Elem()
				}
				if valueType.Kind() == reflect.Struct {
					embedded := compactSchemaFromType(valueType, stack)
					if embedded.Kind == "object" {
						result.Fields = append(result.Fields, embedded.Fields...)
						continue
					}
				}
			}
			name, omitempty, skip := runtimeJSONField(field)
			if skip {
				continue
			}
			fieldSchema := compactSchemaFromType(field.Type, stack)
			if description := field.Tag.Get("description"); description != "" {
				fieldSchema.Description = description
			}
			result.Fields = append(result.Fields, CompactField{
				Name:     name,
				Required: runtimeFieldRequired(field, omitempty),
				Type:     fieldSchema,
			})
		}
		return result
	default:
		return CompactSchema{Kind: "raw"}
	}
}

func runtimeJSONField(field reflect.StructField) (name string, omitempty bool, skip bool) {
	raw := field.Tag.Get("json")
	if raw == "" {
		return field.Name, false, false
	}
	parts := strings.Split(raw, ",")
	if parts[0] == "-" {
		return "", false, true
	}
	name = parts[0]
	if name == "" {
		name = field.Name
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, false
}

func runtimeFieldRequired(field reflect.StructField, omitempty bool) bool {
	switch field.Tag.Get("required") {
	case "true":
		return true
	case "false":
		return false
	}
	if omitempty {
		return false
	}
	return field.Type.Kind() != reflect.Pointer
}

func compactSchemaJSONSchema(schema CompactSchema) map[string]any {
	result := make(map[string]any)
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	switch schema.Kind {
	case "object":
		result["type"] = "object"
		properties := make(map[string]any, len(schema.Fields))
		required := make([]string, 0)
		for _, field := range schema.Fields {
			properties[field.Name] = compactSchemaJSONSchema(field.Type)
			if field.Required {
				required = append(required, field.Name)
			}
		}
		result["properties"] = properties
		if len(required) > 0 {
			result["required"] = required
		}
	case "array":
		result["type"] = "array"
		if schema.Items != nil {
			result["items"] = compactSchemaJSONSchema(*schema.Items)
		}
	case "map":
		result["type"] = "object"
		if schema.Values != nil {
			result["additionalProperties"] = compactSchemaJSONSchema(*schema.Values)
		}
	case "raw":
		// 空 schema 表示任意 JSON。
	default:
		result["type"] = schema.Kind
	}
	return result
}

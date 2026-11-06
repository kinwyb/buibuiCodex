package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type overrideConfig struct {
	Name        *string           `json:"name"`
	Description *string           `json:"description"`
	Method      *string           `json:"method"`
	Path        *string           `json:"path"`
	Request     *requestOverride  `json:"request"`
	Response    *responseOverride `json:"response"`
}

type requestOverride struct {
	Parameters json.RawMessage    `json:"parameters"`
	Struct     *string            `json:"struct"`
	Path       *map[string]string `json:"path"`
	Query      *map[string]string `json:"query"`
	Body       *map[string]string `json:"body"`
	BodyParam  *string            `json:"bodyParam"`
}

type responseOverride struct {
	Schema      json.RawMessage            `json:"schema"`
	Struct      *string                    `json:"struct"`
	Definitions map[string]json.RawMessage `json:"definitions"`
}

func loadOverride(path string) (*overrideConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取覆盖文件失败: %w", err)
	}
	var override overrideConfig
	if err := json.Unmarshal(data, &override); err != nil {
		return nil, fmt.Errorf("解析覆盖文件失败: %w", err)
	}
	if len(override.RequestParameters()) > 0 {
		if err := validateSchemaObject(override.RequestParameters()); err != nil {
			return nil, fmt.Errorf("覆盖文件 request.parameters 无效: %w", err)
		}
	}
	if override.Response != nil && len(override.Response.Schema) > 0 {
		if err := validateSchemaObject(override.Response.Schema); err != nil {
			return nil, fmt.Errorf("覆盖文件 response.schema 无效: %w", err)
		}
	}
	return &override, nil
}

func (o *overrideConfig) RequestParameters() json.RawMessage {
	if o == nil || o.Request == nil {
		return nil
	}
	return o.Request.Parameters
}

func (o *overrideConfig) apply(tool *generatedTool, loaded *loadedSwagger) error {
	if o == nil {
		return nil
	}
	if o.Method != nil && *o.Method != "" {
		tool.Method = normalizeMethod(*o.Method)
	}
	if o.Path != nil && *o.Path != "" {
		tool.Path = cleanPath(*o.Path)
	}
	if o.Request != nil {
		if o.Request.Struct != nil && len(o.Request.Parameters) > 0 {
			return errors.New("request.struct 与 request.parameters 不能同时设置")
		}
		if o.Request.Struct != nil {
			source, err := parseStructSource(*o.Request.Struct)
			if err != nil {
				return err
			}
			if err := applyRequestStruct(tool, source, tool.Method); err != nil {
				return fmt.Errorf("应用 request.struct 失败: %w", err)
			}
		}
		if len(o.Request.Parameters) > 0 {
			tool.Parameters = append(json.RawMessage(nil), o.Request.Parameters...)
			compact, err := compactSchemaFromJSONSchema(tool.Parameters)
			if err != nil {
				return fmt.Errorf("转换自定义请求 schema 失败: %w", err)
			}
			tool.RequestSchema = compact
		}
		if o.Request.Path != nil {
			tool.Binding.Path = cloneStringMap(*o.Request.Path)
		}
		if o.Request.Query != nil {
			tool.Binding.Query = cloneStringMap(*o.Request.Query)
		}
		if o.Request.BodyParam != nil && o.Request.Body != nil {
			return errors.New("request.bodyParam 与 request.body 不能同时设置")
		}
		if o.Request.BodyParam != nil {
			tool.Binding.BodyParam = *o.Request.BodyParam
			tool.Binding.Body = nil
			tool.Binding.BodyAll = false
		}
		if o.Request.Body != nil {
			tool.Binding.BodyParam = ""
			tool.Binding.Body = cloneStringMap(*o.Request.Body)
			tool.Binding.BodyAll = false
		}
	}
	if o.Response != nil && o.Response.Struct != nil && len(o.Response.Schema) > 0 {
		return errors.New("response.struct 与 response.schema 不能同时设置")
	}
	if o.Response != nil && o.Response.Struct != nil {
		source, err := parseStructSource(*o.Response.Struct)
		if err != nil {
			return err
		}
		if err := applyResponseStruct(tool, source); err != nil {
			return fmt.Errorf("应用 response.struct 失败: %w", err)
		}
	}
	if o.Response != nil && len(o.Response.Schema) > 0 {
		definitions := make(map[string]json.RawMessage, len(loaded.spec.Definitions)+len(o.Response.Definitions))
		for name, raw := range loaded.spec.Definitions {
			definitions[name] = raw
		}
		for name, raw := range o.Response.Definitions {
			definitions[name] = raw
		}
		converter := schemaConverter{definitions: definitions}
		response, err := converter.convert(o.Response.Schema, make(map[string]bool))
		if err != nil {
			return fmt.Errorf("转换自定义响应 schema 失败: %w", err)
		}
		tool.ResponseSchema = response
	}
	return nil
}

func normalizeMethod(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func cloneStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func validateSchemaObject(raw json.RawMessage) error {
	if !json.Valid(raw) {
		return errors.New("不是合法 JSON")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return errors.New("必须是 JSON 对象")
	}
	if object == nil {
		return errors.New("必须是 JSON 对象")
	}
	return nil
}

func compactSchemaFromJSONSchema(raw json.RawMessage) (openapi.CompactSchema, error) {
	var root struct {
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return openapi.CompactSchema{}, err
	}
	converter := schemaConverter{definitions: root.Definitions}
	return converter.convert(raw, make(map[string]bool))
}

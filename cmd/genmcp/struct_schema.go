package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type structSource struct {
	file string
	expr string
}

type goTypeLoader struct {
	definitions map[string]ast.Expr
}

func parseStructSource(value string) (*structSource, error) {
	value = strings.TrimSpace(value)
	index := strings.LastIndex(value, "#")
	if index <= 0 || index == len(value)-1 {
		return nil, errors.New("Go 结构体格式应为 path/to/file.go#Type 或 path/to/file.go#[]Type")
	}
	return &structSource{file: value[:index], expr: value[index+1:]}, nil
}

func loadCompactType(source *structSource) (openapi.CompactSchema, error) {
	directory := filepath.Dir(source.file)
	if _, err := os.Stat(source.file); err != nil {
		return openapi.CompactSchema{}, fmt.Errorf("读取 Go 类型文件 %s 失败: %w", source.file, err)
	}
	packages, err := parser.ParseDir(token.NewFileSet(), directory, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		return openapi.CompactSchema{}, fmt.Errorf("解析 Go 目录 %s 失败: %w", directory, err)
	}
	loader := &goTypeLoader{definitions: make(map[string]ast.Expr)}
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, declaration := range file.Decls {
				genDecl, ok := declaration.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					loader.definitions[typeSpec.Name.Name] = typeSpec.Type
				}
			}
		}
	}
	expression, err := parser.ParseExpr(source.expr)
	if err != nil {
		return openapi.CompactSchema{}, fmt.Errorf("解析 Go 类型 %s 失败: %w", source.expr, err)
	}
	return loader.convert(expression, make(map[string]bool))
}

func (l *goTypeLoader) convert(expression ast.Expr, stack map[string]bool) (openapi.CompactSchema, error) {
	switch typed := expression.(type) {
	case *ast.StarExpr:
		return l.convert(typed.X, stack)
	case *ast.ArrayType:
		item, err := l.convert(typed.Elt, stack)
		if err != nil {
			return openapi.CompactSchema{}, err
		}
		return openapi.CompactSchema{Kind: "array", Items: &item}, nil
	case *ast.MapType:
		value, err := l.convert(typed.Value, stack)
		if err != nil {
			return openapi.CompactSchema{}, err
		}
		return openapi.CompactSchema{Kind: "map", Values: &value}, nil
	case *ast.StructType:
		return l.convertStruct(typed, stack)
	case *ast.Ident:
		if kind, ok := goBuiltinKind(typed.Name); ok {
			return openapi.CompactSchema{Kind: kind}, nil
		}
		definition, ok := l.definitions[typed.Name]
		if !ok {
			return openapi.CompactSchema{Kind: "raw"}, nil
		}
		if stack[typed.Name] {
			return openapi.CompactSchema{Kind: "raw"}, nil
		}
		stack[typed.Name] = true
		result, err := l.convert(definition, stack)
		delete(stack, typed.Name)
		return result, err
	case *ast.SelectorExpr:
		name := selectorName(typed)
		switch name {
		case "time.Time":
			return openapi.CompactSchema{Kind: "string"}, nil
		case "time.Duration":
			return openapi.CompactSchema{Kind: "integer"}, nil
		case "json.RawMessage":
			return openapi.CompactSchema{Kind: "raw"}, nil
		default:
			return openapi.CompactSchema{Kind: "raw"}, nil
		}
	default:
		return openapi.CompactSchema{Kind: "raw"}, nil
	}
}

func (l *goTypeLoader) convertStruct(node *ast.StructType, stack map[string]bool) (openapi.CompactSchema, error) {
	result := openapi.CompactSchema{Kind: "object"}
	for _, field := range node.Fields.List {
		if len(field.Names) == 0 {
			embedded, err := l.convert(field.Type, stack)
			if err != nil {
				return openapi.CompactSchema{}, err
			}
			if embedded.Kind == "object" {
				result.Fields = append(result.Fields, embedded.Fields...)
			}
			continue
		}
		tag := parseStructTag(field.Tag)
		jsonName, omitempty, skip := parseJSONTag(tag, field.Names[0].Name)
		if skip {
			continue
		}
		fieldSchema, err := l.convert(field.Type, stack)
		if err != nil {
			return openapi.CompactSchema{}, err
		}
		if description := tag.Get("description"); description != "" {
			fieldSchema.Description = description
		}
		for index, name := range field.Names {
			if !ast.IsExported(name.Name) {
				continue
			}
			fieldName := jsonName
			if index > 0 {
				fieldName = name.Name
			}
			required := requiredFromTag(tag, omitempty, field.Type)
			result.Fields = append(result.Fields, openapi.CompactField{
				Name:     fieldName,
				Required: required,
				Type:     fieldSchema,
			})
		}
	}
	return result, nil
}

func goBuiltinKind(name string) (string, bool) {
	switch name {
	case "string":
		return "string", true
	case "bool":
		return "boolean", true
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer", true
	case "float32", "float64":
		return "number", true
	case "any", "interface{}":
		return "raw", true
	default:
		return "", false
	}
}

func parseStructTag(tag *ast.BasicLit) reflect.StructTag {
	if tag == nil {
		return ""
	}
	value, err := strconv.Unquote(tag.Value)
	if err != nil {
		return ""
	}
	return reflect.StructTag(value)
}

func parseJSONTag(tag reflect.StructTag, fallback string) (name string, omitempty bool, skip bool) {
	raw := tag.Get("json")
	if raw == "" {
		return fallback, false, false
	}
	parts := strings.Split(raw, ",")
	if parts[0] == "-" {
		return "", false, true
	}
	name = parts[0]
	if name == "" {
		name = fallback
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, false
}

func requiredFromTag(tag reflect.StructTag, omitempty bool, expression ast.Expr) bool {
	switch tag.Get("required") {
	case "true":
		return true
	case "false":
		return false
	}
	if omitempty {
		return false
	}
	if _, ok := expression.(*ast.StarExpr); ok {
		return false
	}
	return true
}

func selectorName(expression *ast.SelectorExpr) string {
	if ident, ok := expression.X.(*ast.Ident); ok {
		return ident.Name + "." + expression.Sel.Name
	}
	return expression.Sel.Name
}

func applyRequestStruct(tool *generatedTool, source *structSource, method string) error {
	schema, err := loadCompactType(source)
	if err != nil {
		return err
	}
	tool.Parameters = nil
	tool.RequestSchema = openapi.CompactSchema{}
	tool.RequestType = source.expr
	tool.RequestSource = source.file
	tool.Binding = openapi.OpenAPIRequestBinding{}

	placeholders := pathPlaceholders(tool.Path)
	if schema.Kind != "object" {
		if strings.EqualFold(method, "GET") || strings.EqualFold(method, "DELETE") {
			return errors.New("GET/DELETE 请求结构体必须是对象")
		}
		tool.Binding.BodyParam = "body"
		return nil
	}
	pathMap := make(map[string]string)
	queryMap := make(map[string]string)
	hasBody := false
	matchedPaths := make(map[string]bool)
	for _, field := range schema.Fields {
		if placeholders[field.Name] {
			pathMap[field.Name] = field.Name
			matchedPaths[field.Name] = true
			continue
		}
		if strings.EqualFold(method, "GET") || strings.EqualFold(method, "DELETE") {
			queryMap[field.Name] = field.Name
		} else {
			hasBody = true
		}
	}
	for name := range placeholders {
		if !matchedPaths[name] {
			return fmt.Errorf("请求结构体缺少路径字段 %s", name)
		}
	}
	if len(pathMap) > 0 {
		tool.Binding.Path = pathMap
	}
	if len(queryMap) > 0 {
		tool.Binding.Query = queryMap
	}
	if hasBody {
		tool.Binding.BodyAll = true
	}
	return nil
}

func applyResponseStruct(tool *generatedTool, source *structSource) error {
	if _, err := loadCompactType(source); err != nil {
		return err
	}
	tool.ResponseSchema = openapi.CompactSchema{}
	tool.ResponseType = source.expr
	tool.ResponseSource = source.file
	return nil
}

func pathPlaceholders(path string) map[string]bool {
	result := make(map[string]bool)
	for {
		start := strings.IndexByte(path, '{')
		if start < 0 {
			break
		}
		path = path[start+1:]
		end := strings.IndexByte(path, '}')
		if end < 0 {
			break
		}
		if end > 0 {
			result[path[:end]] = true
		}
		path = path[end+1:]
	}
	return result
}

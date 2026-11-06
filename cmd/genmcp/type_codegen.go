package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type goTypeRenderer struct {
	used      map[string]bool
	lines     []string
	rawNeeded bool
}

func newGoTypeRenderer() *goTypeRenderer {
	return &goTypeRenderer{used: make(map[string]bool)}
}

func (r *goTypeRenderer) expression(schema openapi.CompactSchema, suggested string) string {
	switch schema.Kind {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "raw":
		r.rawNeeded = true
		return "json.RawMessage"
	case "array":
		if schema.Items == nil {
			r.rawNeeded = true
			return "[]json.RawMessage"
		}
		return "[]" + r.expression(*schema.Items, suggested+"Item")
	case "map":
		if schema.Values == nil {
			r.rawNeeded = true
			return "map[string]json.RawMessage"
		}
		return "map[string]" + r.expression(*schema.Values, suggested+"Value")
	case "object":
		name := r.uniqueTypeName(suggested)
		r.renderObject(name, schema)
		return name
	default:
		r.rawNeeded = true
		return "json.RawMessage"
	}
}

func (r *goTypeRenderer) renderObject(name string, schema openapi.CompactSchema) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "type %s struct {\n", name)
	fieldNames := make(map[string]bool)
	for _, field := range schema.Fields {
		fieldName := uniqueFieldName(fieldNames, exportedIdentifier(field.Name))
		fieldType := r.expression(field.Type, name+exportedIdentifier(field.Name))
		jsonValue := field.Name
		if !field.Required {
			jsonValue += ",omitempty"
		}
		fmt.Fprintf(
			&builder,
			"\t%s %s %s\n",
			fieldName,
			fieldType,
			goStructTagLiteral(jsonValue, field.Type.Description),
		)
	}
	builder.WriteString("}\n")
	r.lines = append(r.lines, builder.String())
}

func goStructTagLiteral(jsonName, description string) string {
	tag := "json:" + strconv.Quote(jsonName) + " description:" + strconv.Quote(description)
	if strings.Contains(tag, "`") {
		return strconv.Quote(tag)
	}
	return "`" + tag + "`"
}

func (r *goTypeRenderer) uniqueTypeName(value string) string {
	value = exportedIdentifier(value)
	if value == "" {
		value = "GeneratedType"
	}
	name := value
	for suffix := 2; r.used[name]; suffix++ {
		name = value + strconv.Itoa(suffix)
	}
	r.used[name] = true
	return name
}

func (r *goTypeRenderer) code() string {
	return strings.Join(r.lines, "\n")
}

func uniqueFieldName(used map[string]bool, value string) string {
	value = exportedIdentifier(value)
	if value == "" {
		value = "Field"
	}
	name := value
	for suffix := 2; used[name]; suffix++ {
		name = value + strconv.Itoa(suffix)
	}
	used[name] = true
	return name
}

func exportedIdentifier(value string) string {
	var builder strings.Builder
	upperNext := true
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			upperNext = true
			continue
		}
		if builder.Len() == 0 && unicode.IsDigit(char) {
			builder.WriteString("Field")
		}
		if upperNext {
			builder.WriteRune(unicode.ToUpper(char))
			upperNext = false
		} else {
			builder.WriteRune(char)
		}
	}
	result := builder.String()
	if result == "" {
		return ""
	}
	runes := []rune(result)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

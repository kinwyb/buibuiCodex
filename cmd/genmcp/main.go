package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type cliOptions struct {
	swagger        string
	api            string
	out            string
	method         string
	name           string
	description    string
	override       string
	requestStruct  string
	responseStruct string
	allowMethods   stringList
	check          bool
}

type generatedTool struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Parameters     json.RawMessage
	RequestSchema  openapi.CompactSchema
	ResponseSchema openapi.CompactSchema
	RequestType    string
	RequestSource  string
	ResponseType   string
	ResponseSource string
	Binding        openapi.OpenAPIRequestBinding
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "genmcp:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	opts, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}
	var override *overrideConfig
	if opts.override != "" {
		override, err = loadOverride(opts.override)
		if err != nil {
			return err
		}
	}
	loaded, err := loadSwagger(opts.swagger)
	if err != nil {
		return err
	}
	operation, err := selectOperation(loaded, opts.api, opts.method)
	if err != nil {
		return err
	}
	nameOverride := opts.name
	if nameOverride == "" && override != nil && override.Name != nil {
		nameOverride = *override.Name
	}
	name, err := operationName(operation, nameOverride)
	if err != nil {
		return err
	}
	description := strings.TrimSpace(opts.description)
	if description == "" && override != nil && override.Description != nil {
		description = strings.TrimSpace(*override.Description)
	}
	if description == "" {
		description = operationDescription(operation)
	}
	request, err := buildRequestSchema(operation, loaded.spec)
	if err != nil {
		return fmt.Errorf("生成 %s 参数失败: %w", name, err)
	}
	response, err := buildResponseSchema(operation, loaded.spec)
	if err != nil {
		return fmt.Errorf("生成 %s 响应 schema 失败: %w", name, err)
	}

	tool := generatedTool{
		Name:           name,
		Description:    description,
		Method:         operation.Method,
		Path:           operation.Path,
		Parameters:     request.Parameters,
		RequestSchema:  request.Compact,
		ResponseSchema: response,
		Binding:        request.binding(),
	}
	if err := override.apply(&tool, loaded); err != nil {
		return err
	}
	if opts.requestStruct != "" {
		source, err := parseStructSource(opts.requestStruct)
		if err != nil {
			return err
		}
		if err := applyRequestStruct(&tool, source, tool.Method); err != nil {
			return fmt.Errorf("应用 --request-struct 失败: %w", err)
		}
	}
	if opts.responseStruct != "" {
		source, err := parseStructSource(opts.responseStruct)
		if err != nil {
			return err
		}
		if err := applyResponseStruct(&tool, source); err != nil {
			return fmt.Errorf("应用 --response-struct 失败: %w", err)
		}
	}
	if err := validateMethod(tool.Method, opts.allowMethods); err != nil {
		return err
	}
	outputPath, err := validateOutputPath(opts.out)
	if err != nil {
		return err
	}
	if err := validateTypedTypePackages(&tool, outputPath); err != nil {
		return err
	}
	content, err := renderGeneratedFile(tool, loaded)
	if err != nil {
		return err
	}
	formatted, err := format.Source(content)
	if err != nil {
		return fmt.Errorf("格式化生成代码失败: %w\n%s", err, content)
	}

	if opts.check {
		existing, err := os.ReadFile(outputPath)
		if err != nil {
			return fmt.Errorf("检查生成文件失败: %w", err)
		}
		if !bytes.Equal(existing, formatted) {
			return fmt.Errorf("生成文件不是最新状态: %s", opts.out)
		}
		fmt.Fprintf(stdout, "✓ %s 已是最新状态\n", opts.out)
		return nil
	}
	if err := writeFileAtomically(outputPath, formatted); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ 已生成 %s\n", opts.out)
	fmt.Fprintf(stdout, "  工具: %s\n", tool.Name)
	fmt.Fprintf(stdout, "  接口: %s %s\n", tool.Method, tool.Path)
	return nil
}

func parseFlags(args []string, stderr io.Writer) (*cliOptions, error) {
	swaggerDefault := os.Getenv("APIDOCS_URL")
	if swaggerDefault == "" {
		swaggerDefault = defaultSwaggerURL
	}
	opts := &cliOptions{}
	flags := flag.NewFlagSet("genmcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.swagger, "swagger", swaggerDefault, "Swagger JSON 地址或本地文件")
	flags.StringVar(&opts.api, "api", "", "接口路径或完整 URL，例如 /nc/material/query")
	flags.StringVar(&opts.out, "out", "", "输出 Go 文件，必须位于 mcp/serv/generated/")
	flags.StringVar(&opts.method, "method", "", "HTTP 方法，路径只有一个方法时可省略")
	flags.StringVar(&opts.name, "name", "", "覆盖 MCP 工具名")
	flags.StringVar(&opts.description, "description", "", "覆盖 MCP 工具描述")
	flags.StringVar(&opts.override, "override", "", "自定义请求和响应结构 JSON 文件")
	flags.StringVar(&opts.requestStruct, "request-struct", "", "Go 请求结构体，如 ./types.go#MaterialQuery")
	flags.StringVar(&opts.responseStruct, "response-struct", "", "Go 响应结构体，如 ./types.go#[]Material")
	flags.Var(&opts.allowMethods, "allow-method", "允许非 GET 方法，可重复指定")
	flags.BoolVar(&opts.check, "check", false, "只检查输出文件是否最新")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), `用法:
  go run ./cmd/genmcp --api /nc/material/query --out mcp/serv/generated/nc_material_query.go --allow-method post

API 基础地址默认视为 %s。只传 /nc/material/query 时无需带基础地址。
Swagger 请求或响应结构不正确时可通过 --override 指定 JSON 覆盖文件。

`, defaultAPIBaseURL)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if opts.api == "" {
		return nil, errors.New("--api 不能为空")
	}
	if opts.out == "" {
		return nil, errors.New("--out 不能为空")
	}
	return opts, nil
}

func validateMethod(method string, allowed stringList) error {
	method = strings.ToLower(method)
	if !slices.Contains(supportedMethods, method) {
		return fmt.Errorf("不支持的 HTTP 方法: %s", strings.ToUpper(method))
	}
	if method == "get" {
		return nil
	}
	for _, value := range allowed {
		if strings.EqualFold(strings.TrimSpace(value), method) {
			return nil
		}
	}
	return fmt.Errorf("%s 默认禁止生成，如需生成请增加 --allow-method %s", strings.ToUpper(method), method)
}

func renderGeneratedFile(tool generatedTool, loaded *loadedSwagger) ([]byte, error) {
	renderer := newGoTypeRenderer()
	typePrefix := exportedIdentifier(tool.Name)
	requestType := tool.RequestType
	if requestType == "" {
		requestType = renderer.expression(tool.RequestSchema, typePrefix+"Request")
	}
	responseType := tool.ResponseType
	if responseType == "" {
		responseType = renderer.expression(tool.ResponseSchema, typePrefix+"Response")
	}
	imports := ""
	if renderer.rawNeeded {
		imports = "\t\"encoding/json\"\n\n"
	}
	typeDeclarations := renderer.code()
	constructor := fmt.Sprintf(
		"openapi.NewTypedRequestResponseOpenAPITool[%s, %s]",
		requestType,
		responseType,
	)

	var builder strings.Builder
	fmt.Fprintf(&builder, `// Code generated by cmd/genmcp; DO NOT EDIT.
// Source: %s v%s (Swagger 2.0)
// Spec SHA-256: %s

package generated

import (
%s	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

%s

func init() {
	serv.RegisterTool(%s(openapi.OpenAPIToolSpec{
		Name:        %s,
		Description: %s,
		Method:      %s,
		Path:        %s,
		Binding: openapi.OpenAPIRequestBinding{
			Path:      %#v,
			Query:     %#v,
			Body:      %#v,
			BodyAll:   %t,
			BodyParam: %s,
		},
	}))
}
`,
		singleLine(loaded.spec.Info.Title),
		singleLine(loaded.spec.Info.Version),
		loaded.sha256,
		imports,
		typeDeclarations,
		constructor,
		strconv.Quote(tool.Name),
		strconv.Quote(tool.Description),
		strconv.Quote(tool.Method),
		strconv.Quote(tool.Path),
		tool.Binding.Path,
		tool.Binding.Query,
		tool.Binding.Body,
		tool.Binding.BodyAll,
		strconv.Quote(tool.Binding.BodyParam),
	)
	return []byte(builder.String()), nil
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func validateOutputPath(output string) (string, error) {
	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	generatedDir := filepath.Join(root, "mcp", "serv", "generated")
	if filepath.Dir(absolute) != generatedDir || filepath.Ext(absolute) != ".go" {
		return "", fmt.Errorf("--out 必须直接位于 %s 且使用 .go 后缀", generatedDir)
	}
	return absolute, nil
}

func validateTypedTypePackages(tool *generatedTool, outputPath string) error {
	for _, item := range []struct {
		name   string
		typ    string
		source string
	}{
		{name: "请求", typ: tool.RequestType, source: tool.RequestSource},
		{name: "响应", typ: tool.ResponseType, source: tool.ResponseSource},
	} {
		if item.typ == "" {
			continue
		}
		sourcePath, err := filepath.Abs(item.source)
		if err != nil {
			return fmt.Errorf("解析%s结构体文件失败: %w", item.name, err)
		}
		if filepath.Dir(sourcePath) != filepath.Dir(outputPath) {
			return fmt.Errorf(
				"%s结构体必须与生成文件位于同一 Go 包: %s 和 %s",
				item.name,
				filepath.Dir(sourcePath),
				filepath.Dir(outputPath),
			)
		}
	}
	return nil
}

func findRepoRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("未找到 go.mod")
		}
		current = parent
	}
}

func writeFileAtomically(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".genmcp-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时生成文件失败: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("写入临时生成文件失败: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("同步临时生成文件失败: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("关闭临时生成文件失败: %w", err)
	}
	if err := os.Chmod(tempName, 0644); err != nil {
		return fmt.Errorf("设置生成文件权限失败: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("替换生成文件失败: %w", err)
	}
	return nil
}

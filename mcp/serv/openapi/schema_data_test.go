package openapi

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

func TestEncodeSchemaDataObjectUsesDescriptionKeysInOrder(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "code", Required: true, Type: CompactSchema{Kind: "string", Description: "物料编码"}},
			{Name: "name", Type: CompactSchema{Kind: "string", Description: "物料名称"}},
			{Name: "materialconvert", Type: CompactSchema{
				Kind:        "array",
				Description: "计量单位集合",
				Items: &CompactSchema{
					Kind: "object",
					Fields: []CompactField{
						{Name: "measdoc", Type: CompactSchema{Kind: "string", Description: "单位名称"}},
						{Name: "measrate", Type: CompactSchema{Kind: "string", Description: "转换率"}},
					},
				},
			}},
			{Name: "attributes", Type: CompactSchema{
				Kind:   "map",
				Values: &CompactSchema{Kind: "string"},
			}},
			{Name: "version", Type: CompactSchema{Kind: "integer", Description: "版本号"}},
		},
	}
	data := json.RawMessage(`{
		"version": 2,
		"name": "物料A",
		"materialconvert": [
			{"measrate": "1", "measdoc": "件"}
		],
		"code": "M001",
		"attributes": {"color": "红"},
		"zzz": {"keep": true},
		"aaa": "extra"
	}`)
	output, err := EncodeSchemaData(schema, data, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"data":{"物料编码":"M001","物料名称":"物料A","计量单位集合":[{"单位名称":"件","转换率":"1"}],"attributes":{"color":"红"},"版本号":2,"aaa":"extra","zzz":{"keep":true}}}`
	if output != want {
		t.Fatalf("output = %s\nwant   = %s", output, want)
	}
}

func TestEncodeSchemaDataMissingFieldBecomesNull(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "code", Type: CompactSchema{Kind: "string"}},
			{Name: "name", Type: CompactSchema{Kind: "string"}},
		},
	}
	output, err := EncodeSchemaData(schema, json.RawMessage(`{"name":"物料A"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"data":{"code":null,"name":"物料A"}}`
	if output != want {
		t.Fatalf("output = %s, want %s", output, want)
	}
}

func TestEncodeSchemaDataEmptyArrayOmitsSchemaAndPage(t *testing.T) {
	schema := CompactSchema{
		Kind: "array",
		Items: &CompactSchema{
			Kind: "object",
			Fields: []CompactField{
				{Name: "code", Type: CompactSchema{Kind: "string", Description: "物料编码"}},
			},
		},
	}
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "null", data: "null", want: `{"data":null}`},
		{name: "empty array", data: "[]", want: `{"data":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := EncodeSchemaData(schema, json.RawMessage(test.data), json.RawMessage(`{"page":1,"total":0}`))
			if err != nil {
				t.Fatal(err)
			}
			if output != test.want {
				t.Fatalf("output = %s, want %s", output, test.want)
			}
		})
	}
}

func TestEncodeSchemaDataObjectFallsBackForDuplicateDescription(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "code", Type: CompactSchema{Kind: "string", Description: "物料编码"}},
			{Name: "spec", Type: CompactSchema{Kind: "string", Description: "物料编码"}},
			{Name: "name", Type: CompactSchema{Kind: "string"}},
		},
	}
	output, err := EncodeSchemaData(schema, json.RawMessage(`{"code":"M001","spec":"10x20","name":"物料A"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"data":{"code":"M001","spec":"10x20","name":"物料A"}}`
	if output != want {
		t.Fatalf("output = %s, want %s", output, want)
	}
}

func TestEncodeSchemaDataObjectAvoidsDescriptionNameCollision(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "code", Type: CompactSchema{Kind: "string", Description: "spec"}},
			{Name: "spec", Type: CompactSchema{Kind: "string"}},
			{Name: "name", Type: CompactSchema{Kind: "string", Description: "物料名称"}},
		},
	}
	output, err := EncodeSchemaData(schema, json.RawMessage(`{"code":"M001","spec":"10x20","name":"物料A"}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"data":{"code":"M001","spec":"10x20","物料名称":"物料A"}}`
	if output != want {
		t.Fatalf("output = %s, want %s", output, want)
	}
}

func TestEncodeSchemaDataLargeResultUsesFile(t *testing.T) {
	tempDir := t.TempDir()
	oldDir, oldURL := serv.TmpDir, serv.TmpUrl
	serv.TmpDir = tempDir
	serv.TmpUrl = "http://example.test/tmp_file"
	t.Cleanup(func() {
		serv.TmpDir = oldDir
		serv.TmpUrl = oldURL
	})

	schema := CompactSchema{
		Kind: "array",
		Items: &CompactSchema{
			Kind: "object",
			Fields: []CompactField{
				{Name: "code", Type: CompactSchema{Kind: "string"}},
			},
		},
	}
	var rows []map[string]string
	for i := 0; i < 300; i++ {
		rows = append(rows, map[string]string{"code": "material-code-with-some-length-" + string(rune('A'+i%26))})
	}
	data, _ := json.Marshal(rows)
	output, err := EncodeSchemaData(schema, data, json.RawMessage(`{"page":2,"total":100}`))
	if err != nil {
		t.Fatal(err)
	}

	var preview struct {
		Format string         `json:"format"`
		Schema any            `json:"schema"`
		Data   []any          `json:"data"`
		Page   any            `json:"page"`
		Meta   map[string]any `json:"meta"`
	}
	if err := json.Unmarshal([]byte(output), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Format != schemaDataFormat {
		t.Fatalf("preview format = %q", preview.Format)
	}
	if len(preview.Meta) != 4 || preview.Meta["total"] != float64(300) || preview.Meta["shown"] != float64(3) {
		t.Fatalf("meta = %+v", preview.Meta)
	}
	if preview.Meta["fileHint"] != "文件内容是上方 data 数组本身：顶层就是 JSON 数组（[...]），不含 format/layout/schema/page；"+
		"请用脚本或代码筛选、聚合、分页读取，只加载回答所需的字段和记录，不要一次性读取完整文件" {
		t.Fatalf("file metadata = %+v", preview.Meta)
	}
	if len(preview.Data) != 3 {
		t.Fatalf("preview rows = %d", len(preview.Data))
	}
	if preview.Page == nil {
		t.Fatal("page metadata was lost")
	}
	fileValue, ok := preview.Meta["file"].(string)
	if !ok {
		t.Fatalf("meta file = %#v", preview.Meta["file"])
	}
	fileURL, err := url.Parse(fileValue)
	if err != nil {
		t.Fatal(err)
	}
	if fileURL.Query().Get("file") == "" {
		t.Fatalf("file url = %s", fileValue)
	}
	var fileCount int
	if err := filepath.Walk(tempDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() {
			fileCount++
			fullData, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var fileData []any
			if err := json.Unmarshal(fullData, &fileData); err != nil {
				return err
			}
			if len(fileData) != 300 {
				t.Fatalf("file rows = %d", len(fileData))
			}
			previewRow, _ := json.Marshal(preview.Data[0])
			fileRow, _ := json.Marshal(fileData[0])
			if string(previewRow) != string(fileRow) {
				t.Fatalf("first row mismatch\npreview: %s\nfile: %s", previewRow, fileRow)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fileCount != 1 {
		t.Fatalf("file count = %d", fileCount)
	}
}

func TestEncodeSchemaDataLargeObjectStaysInline(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "content", Required: true, Type: CompactSchema{Kind: "string"}},
		},
	}
	data, _ := json.Marshal(map[string]string{
		"content": strings.Repeat("large-object-content-", 400),
	})
	output, err := EncodeSchemaData(schema, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Format string         `json:"format"`
		Schema any            `json:"schema"`
		Data   map[string]any `json:"data"`
		Meta   map[string]any `json:"meta"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Format != "" || envelope.Schema != nil || envelope.Meta != nil {
		t.Fatalf("large object should keep named format: format=%q schema=%#v meta=%#v", envelope.Format, envelope.Schema, envelope.Meta)
	}
	if envelope.Data["content"] != strings.Repeat("large-object-content-", 400) {
		t.Fatalf("content was not preserved")
	}
}

func TestEncodeSchemaDataLargeObjectWithObjectArrayFallsBack(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "name", Type: CompactSchema{Kind: "string", Description: "物料名称"}},
			{Name: "items", Type: CompactSchema{
				Kind:        "array",
				Description: "明细",
				Items: &CompactSchema{
					Kind: "object",
					Fields: []CompactField{
						{Name: "code", Type: CompactSchema{Kind: "string", Description: "物料编码"}},
						{Name: "content", Type: CompactSchema{Kind: "string", Description: "明细内容"}},
					},
				},
			}},
		},
	}
	rows := make([]map[string]string, 200)
	for i := range rows {
		rows[i] = map[string]string{
			"code":    "material-code-with-some-length",
			"content": "material-content-with-some-length",
		}
	}
	data, _ := json.Marshal(map[string]any{"name": "物料A", "items": rows})
	output, err := EncodeSchemaData(schema, data, json.RawMessage(`{"total":200}`))
	if err != nil {
		t.Fatal(err)
	}

	var envelope struct {
		Format string         `json:"format"`
		Schema any            `json:"schema"`
		Data   []any          `json:"data"`
		Page   map[string]any `json:"page"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Format != schemaDataFormat || envelope.Schema == nil {
		t.Fatalf("fallback envelope = %+v", envelope)
	}
	if len(envelope.Data) != 2 || envelope.Data[0] != "物料A" {
		t.Fatalf("positional data = %#v", envelope.Data)
	}
	items, ok := envelope.Data[1].([]any)
	if !ok || len(items) != 200 {
		t.Fatalf("positional items = %#v", envelope.Data[1])
	}
	first, ok := items[0].([]any)
	if !ok || len(first) != 2 || first[0] != "material-code-with-some-length" {
		t.Fatalf("positional row = %#v", items[0])
	}
	if envelope.Page["total"] != float64(200) {
		t.Fatalf("page = %#v", envelope.Page)
	}
}

func TestEncodeSchemaDataLargeObjectWithScalarArrayStaysNamed(t *testing.T) {
	schema := CompactSchema{
		Kind: "object",
		Fields: []CompactField{
			{Name: "tags", Type: CompactSchema{
				Kind:        "array",
				Description: "标签",
				Items:       &CompactSchema{Kind: "string"},
			}},
		},
	}
	tags := make([]string, 200)
	for i := range tags {
		tags[i] = "tag-with-some-length"
	}
	data, _ := json.Marshal(map[string]any{"tags": tags})
	output, err := EncodeSchemaData(schema, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Format string         `json:"format"`
		Schema any            `json:"schema"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Format != "" || envelope.Schema != nil {
		t.Fatalf("scalar array should keep named format: format=%q schema=%#v", envelope.Format, envelope.Schema)
	}
	if values, ok := envelope.Data["标签"].([]any); !ok || len(values) != 200 {
		t.Fatalf("tags = %#v", envelope.Data["标签"])
	}
}

func TestEncodeSchemaDataLargeResultFallsBackWhenFileFails(t *testing.T) {
	tempDir := t.TempDir()
	blocked := filepath.Join(tempDir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	oldDir, oldURL := serv.TmpDir, serv.TmpUrl
	serv.TmpDir = blocked
	serv.TmpUrl = "http://example.test/tmp_file"
	t.Cleanup(func() {
		serv.TmpDir = oldDir
		serv.TmpUrl = oldURL
	})

	schema := CompactSchema{
		Kind: "array",
		Items: &CompactSchema{
			Kind: "object",
			Fields: []CompactField{
				{Name: "code", Type: CompactSchema{Kind: "string"}},
			},
		},
	}
	var rows []map[string]string
	for i := 0; i < 300; i++ {
		rows = append(rows, map[string]string{"code": "material-code-with-some-length"})
	}
	data, _ := json.Marshal(rows)
	output, err := EncodeSchemaData(schema, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data []any `json:"data"`
		Meta any   `json:"meta"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 300 || envelope.Meta != nil {
		t.Fatalf("fallback data rows = %d, meta = %#v", len(envelope.Data), envelope.Meta)
	}
}

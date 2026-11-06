package openapi

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

func TestEncodeSchemaDataPreservesOrderAndNestedMeaning(t *testing.T) {
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

	var envelope struct {
		Format string         `json:"format"`
		Layout map[string]any `json:"layout"`
		Schema []any          `json:"schema"`
		Data   []any          `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Format != schemaDataFormat {
		t.Fatalf("format = %q", envelope.Format)
	}
	if len(envelope.Data) != 7 {
		t.Fatalf("data positions = %d, want 7: %v", len(envelope.Data), envelope.Data)
	}
	if envelope.Data[0] != "M001" || envelope.Data[1] != "物料A" {
		t.Fatalf("positional values = %v", envelope.Data)
	}
	converts, ok := envelope.Data[2].([]any)
	if !ok || len(converts) != 1 {
		t.Fatalf("materialconvert = %#v", envelope.Data[2])
	}
	firstConvert, ok := converts[0].([]any)
	if !ok || len(firstConvert) != 2 || firstConvert[0] != "件" || firstConvert[1] != "1" {
		t.Fatalf("nested object = %#v", converts[0])
	}
	if envelope.Data[5] != "extra" {
		t.Fatalf("first extra field = %#v", envelope.Data[5])
	}
	extraObject, ok := envelope.Data[6].(map[string]any)
	if !ok || extraObject["keep"] != true {
		t.Fatalf("second extra field = %#v", envelope.Data[6])
	}
	if envelope.Data[4] != float64(2) {
		t.Fatalf("version = %#v", envelope.Data[4])
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
	var envelope struct {
		Data []any `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 2 || envelope.Data[0] != nil || envelope.Data[1] != "物料A" {
		t.Fatalf("data = %#v", envelope.Data)
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
	if len(preview.Meta) != 3 || preview.Meta["total"] != float64(300) || preview.Meta["shown"] != float64(3) {
		t.Fatalf("meta = %+v", preview.Meta)
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
			var full struct {
				Format string `json:"format"`
				Schema any    `json:"schema"`
				Data   []any  `json:"data"`
			}
			if err := json.Unmarshal(fullData, &full); err != nil {
				return err
			}
			if full.Format != schemaDataFormat {
				t.Fatalf("file format = %q", full.Format)
			}
			if len(full.Data) != 300 {
				t.Fatalf("full rows = %d", len(full.Data))
			}
			previewSchema, _ := json.Marshal(preview.Schema)
			fileSchema, _ := json.Marshal(full.Schema)
			if string(previewSchema) != string(fileSchema) {
				t.Fatalf("schema mismatch\npreview: %s\nfile: %s", previewSchema, fileSchema)
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

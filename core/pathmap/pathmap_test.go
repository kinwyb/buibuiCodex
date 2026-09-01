package pathmap

import "testing"

func TestPathMapper_Noop(t *testing.T) {
	pm := New(nil)
	if !pm.IsNoop() {
		t.Fatal("expected noop")
	}
	if got := pm.ToContainer("/any/path"); got != "/any/path" {
		t.Errorf("ToContainer noop: got %q", got)
	}
	if got := pm.ToHost("/any/path"); got != "/any/path" {
		t.Errorf("ToHost noop: got %q", got)
	}
}

func TestPathMapper_EmptyConfig(t *testing.T) {
	pm := New(&Config{})
	if !pm.IsNoop() {
		t.Fatal("expected noop for empty config")
	}
}

func TestPathMapper_BasicMapping(t *testing.T) {
	pm := New(&Config{
		Mappings: []PathMapping{
			{Host: "/home/user/data", Container: "/app/data"},
			{Host: "/home/user/workspace", Container: "/app/workspace"},
		},
	})

	if pm.IsNoop() {
		t.Fatal("should not be noop")
	}

	// ToContainer
	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/data/file.txt", "/app/data/file.txt"},
		{"/home/user/data/sub/dir/file.txt", "/app/data/sub/dir/file.txt"},
		{"/home/user/workspace/agent/tmp/test.pdf", "/app/workspace/agent/tmp/test.pdf"},
		{"/unrelated/path/file.txt", "/unrelated/path/file.txt"}, // 不匹配，原样返回
	}

	for _, tt := range tests {
		got := pm.ToContainer(tt.input)
		if got != tt.want {
			t.Errorf("ToContainer(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// ToHost
	reverseTests := []struct {
		input string
		want  string
	}{
		{"/app/data/file.txt", "/home/user/data/file.txt"},
		{"/app/data/sub/dir/file.txt", "/home/user/data/sub/dir/file.txt"},
		{"/app/workspace/agent/tmp/test.pdf", "/home/user/workspace/agent/tmp/test.pdf"},
		{"/unrelated/path/file.txt", "/unrelated/path/file.txt"},
	}

	for _, tt := range reverseTests {
		got := pm.ToHost(tt.input)
		if got != tt.want {
			t.Errorf("ToHost(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPathMapper_RoundTrip(t *testing.T) {
	pm := New(&Config{
		Mappings: []PathMapping{
			{Host: "/srv/shared", Container: "/data"},
		},
	})

	original := "/srv/shared/20260901/abc123/report.xlsx"
	container := pm.ToContainer(original)
	back := pm.ToHost(container)
	if back != original {
		t.Errorf("round trip failed: %q -> %q -> %q", original, container, back)
	}
}

func TestPathMapper_SkipInvalidMappings(t *testing.T) {
	pm := New(&Config{
		Mappings: []PathMapping{
			{Host: "", Container: "/app/data"},           // 无效，跳过
			{Host: "/home/user/data", Container: ""},     // 无效，跳过
			{Host: "/home/user/valid", Container: "/ok"}, // 有效
		},
	})

	if len(pm.mappings) != 1 {
		t.Errorf("expected 1 valid mapping, got %d", len(pm.mappings))
	}
}

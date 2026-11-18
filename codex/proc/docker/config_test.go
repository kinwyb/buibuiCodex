package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToCodexConfigMemorySwitch 记忆相关字段必须整体跟随开关，不能只 gate features。
func TestToCodexConfigMemorySwitch(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		param := StartParam{Memories: enabled}
		cfg := param.ToCodexConfig()
		if cfg.Features == nil || cfg.Features.Memory == nil || *cfg.Features.Memory != enabled {
			t.Fatalf("Memories=%v 时 features.memories 应为 %v，实际 %+v", enabled, enabled, cfg.Features)
		}
		if cfg.Memories == nil || cfg.Memories.Use == nil || *cfg.Memories.Use != enabled {
			t.Fatalf("Memories=%v 时 memories.use_memories 应为 %v", enabled, enabled)
		}
		if cfg.Memories.Generate == nil || *cfg.Memories.Generate != enabled {
			t.Fatalf("Memories=%v 时 memories.generate_memories 应为 %v", enabled, enabled)
		}
	}
}

// TestEnsureCodexConfigDetectsChange 首次写入与开关翻转都要报告变化，重复写入不报告。
func TestEnsureCodexConfigDetectsChange(t *testing.T) {
	home := t.TempDir()
	param := StartParam{Memories: true}
	cfg := param.ToCodexConfig()

	changed, err := ensureCodexConfig(home, cfg)
	if err != nil || !changed {
		t.Fatalf("首次写入应 changed=true，实际 changed=%v err=%v", changed, err)
	}
	changed, err = ensureCodexConfig(home, cfg)
	if err != nil || changed {
		t.Fatalf("配置未变应 changed=false，实际 changed=%v err=%v", changed, err)
	}

	paramOff := StartParam{Memories: false}
	cfgOff := paramOff.ToCodexConfig()
	changed, err = ensureCodexConfig(home, cfgOff)
	if err != nil || !changed {
		t.Fatalf("关闭记忆应 changed=true，实际 changed=%v err=%v", changed, err)
	}
	data, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "use_memories = false") {
		t.Fatalf("关闭后 config.toml 应写入 use_memories = false，实际:\n%s", data)
	}
}

// TestEnsureCodexConfigIgnoresUnmanagedKeys app-server 回写的 skills.config 不算配置变化，
// 且稳态下不能被覆盖掉。
func TestEnsureCodexConfigIgnoresUnmanagedKeys(t *testing.T) {
	home := t.TempDir()
	param := StartParam{Memories: true}
	cfg := param.ToCodexConfig()
	if _, err := ensureCodexConfig(home, cfg); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(home, "config.toml")
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	skillState := "\n[[skills.config]]\npath = \"/root/.codex/skills/.system/imagegen/SKILL.md\"\nenabled = false\n"
	if _, err := f.WriteString(skillState); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	changed, err := ensureCodexConfig(home, cfg)
	if err != nil || changed {
		t.Fatalf("额外 skills.config 不应触发重写，实际 changed=%v err=%v", changed, err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[[skills.config]]") {
		t.Fatal("稳态下 app-server 回写的 skills.config 被覆盖")
	}
}

// TestEnsureCodexConfigEmptyHome 未配置 CodexHome 时直接跳过。
func TestEnsureCodexConfigEmptyHome(t *testing.T) {
	param := StartParam{}
	changed, err := ensureCodexConfig("", param.ToCodexConfig())
	if err != nil || changed {
		t.Fatalf("空 CodexHome 应 (false, nil)，实际 (%v, %v)", changed, err)
	}
}

package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseOpenCodeGoProvider(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "opencode-go",
		"models": {
			"qwen3.7-plus": {
				"id": "qwen3.7-plus", "name": "Qwen3.7 Plus",
				"cost": {"input": 0.4, "output": 1.6, "cache_read": 0.04, "cache_write": 0.5,
					"tiers": [{"input": 1.2, "output": 4.8, "cache_read": 0.12, "cache_write": 1.5,
						"tier": {"type": "context", "size": 256000}}]},
				"limit": {"context": 1000000, "output": 65536}
			},
			"deepseek-v4-pro": {
				"id": "deepseek-v4-pro", "name": "DeepSeek V4 Pro",
				"cost": {"input": 0.66, "output": 1.98, "cache_read": 0.022},
				"limit": {"context": 1000000, "output": 384000}
			},
			"no-cost": {"name": "No Cost", "limit": {"context": 1000}}
		}
	}`)
	got, err := parseOpenCodeGoProvider(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Sắp theo ID.
	if got[0].ID != "deepseek-v4-pro" || got[1].ID != "no-cost" || got[2].ID != "qwen3.7-plus" {
		t.Fatalf("order = %s, %s, %s", got[0].ID, got[1].ID, got[2].ID)
	}
	ds := got[0]
	if ds.Provider != "opencode-go" || ds.InputCostPer1M != 0.66 || ds.OutputCostPer1M != 1.98 ||
		ds.CacheReadCostPer1M != 0.022 || ds.ContextWindow != 1000000 || ds.MaxTokens != 384000 || len(ds.Tiers) != 0 {
		t.Fatalf("deepseek = %+v", ds)
	}
	if got[1].ID != "no-cost" || got[1].InputCostPer1M != 0 {
		t.Fatalf("model thiếu id phải lấy key, giá 0: %+v", got[1])
	}
	qw := got[2]
	want := PriceTier{ContextOver: 256000, InputCostPer1M: 1.2, OutputCostPer1M: 4.8, CacheReadCostPer1M: 0.12, CacheWriteCostPer1M: 1.5}
	if len(qw.Tiers) != 1 || qw.Tiers[0] != want {
		t.Fatalf("qwen tiers = %+v", qw.Tiers)
	}
}

func TestOpenCodeGoCacheFresh(t *testing.T) {
	dir := t.TempDir()
	if openCodeGoCacheFresh(dir) {
		t.Fatal("chưa có file mà báo còn hạn")
	}
	write := func(at time.Time) {
		data, _ := json.Marshal(openCodeGoCache{FetchedAt: at})
		writeCacheFile(dir, openCodeGoCacheFileName, data)
	}
	write(time.Now().Add(-time.Hour))
	if !openCodeGoCacheFresh(dir) {
		t.Fatal("cache 1h tuổi phải còn hạn")
	}
	write(time.Now().Add(-25 * time.Hour))
	if openCodeGoCacheFresh(dir) {
		t.Fatal("cache 25h tuổi phải hết hạn")
	}
	if err := os.WriteFile(filepath.Join(dir, openCodeGoCacheFileName), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if openCodeGoCacheFresh(dir) {
		t.Fatal("file hỏng phải coi là hết hạn")
	}
}

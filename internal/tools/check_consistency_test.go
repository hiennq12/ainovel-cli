package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/voocel/ainovel-cli/internal/store"
)

func TestCheckConsistencyReturnsPartialFactsWithWarnings(t *testing.T) {
	dir := t.TempDir()
	st := store.NewStore(dir)
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveDraft(1, "可供检查的章节草稿"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "world_rules.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := NewCheckConsistencyTool(st).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatalf("辅助事实损坏不应中止一致性检查: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "partial" || len(got["_warnings"].([]any)) == 0 || got["content"] == "" {
		t.Fatalf("应返回正文、partial 和数据告警: %+v", got)
	}
}

func TestCheckConsistencyReportsMarkdownResidue(t *testing.T) {
	st := store.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Drafts.SaveDraft(1, "# Chương 1\nKế hoạch *dự kiến* xong."); err != nil {
		t.Fatal(err)
	}
	raw, err := NewCheckConsistencyTool(st).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Violations []struct{ Rule, Target string } `json:"rule_violations"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Violations) != 1 || got.Violations[0].Rule != "markdown_residue" || got.Violations[0].Target != "*" {
		t.Fatalf("writer phải thấy dấu in nghiêng trước khi commit: %+v", got.Violations)
	}
}

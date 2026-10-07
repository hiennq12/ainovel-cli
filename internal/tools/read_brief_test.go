package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/store"
)

func TestReadBriefReturnsStartPromptVerbatim(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	brief := "# Yêu cầu\n\n<!-- Mốc: Bống mất tích 14/11/2005 -->\n" + strings.Repeat("Cung 3: Twist 2 ở ch19–20. ", 2000)
	if err := s.RunMeta.SetStartPrompt(brief); err != nil {
		t.Fatal(err)
	}

	out, err := NewReadBriefTool(s).Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Available bool   `json:"available"`
		Brief     string `json:"brief"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	// 原文不得被截断或剥除注释:作者常把时间线等真实设定写在注释里。
	if !got.Available || got.Brief != strings.TrimSpace(brief) {
		t.Fatalf("read_brief 必须原样返回开书需求, available=%v len=%d want=%d", got.Available, len(got.Brief), len(brief))
	}
}

func TestReadBriefWithoutStartPrompt(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	out, err := NewReadBriefTool(s).Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["available"] != false {
		t.Fatalf("没有开书需求时应返回 available=false, got %v", got)
	}
}

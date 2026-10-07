package bootstrap

import (
	"testing"

	"github.com/voocel/agentcore"
)

func TestWithoutCacheMarkersStripsCopyOnly(t *testing.T) {
	msgs := []agentcore.Message{
		{Role: agentcore.RoleSystem, Metadata: map[string]any{"cache_control": "ephemeral"}},
		{Role: agentcore.RoleUser},
		{Role: agentcore.RoleUser, Metadata: map[string]any{"cache_control": "ephemeral", "keep": 1}},
	}

	out := withoutCacheMarkers(msgs)

	if out[0].Metadata != nil {
		t.Fatalf("system marker not stripped: %v", out[0].Metadata)
	}
	if _, ok := out[2].Metadata["cache_control"]; ok || out[2].Metadata["keep"] != 1 {
		t.Fatalf("last message metadata = %v, want only keep", out[2].Metadata)
	}
	if msgs[0].Metadata["cache_control"] != "ephemeral" || msgs[2].Metadata["cache_control"] != "ephemeral" {
		t.Fatal("caller messages were mutated")
	}
}

func TestWithoutCacheMarkersNoMarkersReturnsInput(t *testing.T) {
	msgs := []agentcore.Message{{Role: agentcore.RoleUser}}
	if out := withoutCacheMarkers(msgs); &out[0] != &msgs[0] {
		t.Fatal("expected input slice returned unchanged")
	}
}

func TestCreateModelAppliesMaxOutputTokens(t *testing.T) {
	pc := ProviderConfig{APIKey: "k", Models: []ModelConfig{{Name: "gpt-4o-mini", MaxOutputTokens: 16384}}}
	m, err := createModelFromConfig("openai", "gpt-4o-mini", pc, map[string]agentcore.ChatModel{})
	if err != nil {
		t.Fatal(err)
	}
	if got := m.(noCacheMarkerModel).GetConfig().MaxTokens; got != 16384 {
		t.Fatalf("MaxTokens = %d, want 16384", got)
	}
	other, err := createModelFromConfig("openai", "gpt-other", pc, map[string]agentcore.ChatModel{})
	if err != nil {
		t.Fatal(err)
	}
	if got := other.(noCacheMarkerModel).GetConfig().MaxTokens; got != 65536 {
		t.Fatalf("undeclared model MaxTokens = %d, want default 65536", got)
	}
}

func TestCreateModelWrapsOpenAIOnly(t *testing.T) {
	cache := map[string]agentcore.ChatModel{}
	m, err := createModelFromConfig("openai", "gpt-4o-mini", ProviderConfig{APIKey: "k"}, cache)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(noCacheMarkerModel); !ok {
		t.Fatalf("openai model = %T, want noCacheMarkerModel", m)
	}
	m, err = createModelFromConfig("anthropic", "claude", ProviderConfig{APIKey: "k"}, cache)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(noCacheMarkerModel); ok {
		t.Fatal("anthropic model must keep cache markers")
	}
}

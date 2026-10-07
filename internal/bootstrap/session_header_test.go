package bootstrap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/voocel/agentcore"
)

// sessionRecorder là server OpenAI-compatible tối giản, ghi lại header session
// của từng request.
type sessionRecorder struct {
	mu       sync.Mutex
	sessions []string
	others   []string
}

func (r *sessionRecorder) handler(w http.ResponseWriter, req *http.Request) {
	_, _ = io.Copy(io.Discard, req.Body)
	r.mu.Lock()
	r.sessions = append(r.sessions, req.Header.Get("x-test-session"))
	r.others = append(r.others, req.Header.Get("x-static"))
	r.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"id":"1","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
}

func (r *sessionRecorder) snapshot() ([]string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.sessions...), append([]string(nil), r.others...)
}

func TestSessionHeaderPerContext(t *testing.T) {
	rec := &sessionRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	pc := ProviderConfig{
		Type:          "openai",
		APIKey:        "k",
		BaseURL:       srv.URL + "/v1",
		SessionHeader: "x-test-session",
		Extra:         map[string]any{"headers": map[string]any{"x-static": "keep"}},
	}
	model, err := createModelFromConfig("gw", "m", pc, map[string]agentcore.ChatModel{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := model.(*sessionHeaderModel); !ok {
		t.Fatalf("model = %T, want *sessionHeaderModel", model)
	}

	call := func(ctx context.Context) {
		t.Helper()
		if _, err := model.Generate(ctx, []agentcore.Message{agentcore.UserMsg("hi")}, nil); err != nil {
			t.Fatal(err)
		}
	}
	runA := WithModelSession(context.Background(), "run-a")
	runB := WithModelSession(context.Background(), "run-b")
	call(runA)
	call(runA)
	call(runB)
	call(context.Background())
	call(context.Background())

	sessions, others := rec.snapshot()
	if sessions[0] != "run-a" || sessions[1] != "run-a" || sessions[2] != "run-b" {
		t.Fatalf("worker sessions = %v", sessions[:3])
	}
	if sessions[3] == "" || sessions[3] != sessions[4] || strings.HasPrefix(sessions[3], "run-") {
		t.Fatalf("default session = %q/%q, want one stable generated id", sessions[3], sessions[4])
	}
	for i, v := range others {
		if v != "keep" {
			t.Fatalf("request %d dropped static header: %q", i, v)
		}
	}
	if got := pc.Extra["headers"].(map[string]any); len(got) != 1 {
		t.Fatalf("config headers mutated: %v", got)
	}
}

func TestSessionHeaderCacheBounded(t *testing.T) {
	builds := 0
	m, err := newSessionHeaderModel(func(string) (agentcore.ChatModel, error) {
		builds++
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxCachedSessions+3; i++ {
		if _, err := m.forContext(WithModelSession(context.Background(), NewModelSessionID())); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.sessions) != maxCachedSessions || len(m.order) != maxCachedSessions {
		t.Fatalf("cache size = %d/%d, want %d", len(m.sessions), len(m.order), maxCachedSessions)
	}
	if builds != maxCachedSessions+3+1 {
		t.Fatalf("builds = %d", builds)
	}
}

func TestNoSessionHeaderKeepsPlainModel(t *testing.T) {
	model, err := createModelFromConfig("gw", "m", ProviderConfig{Type: "openai", APIKey: "k", BaseURL: "http://127.0.0.1:1/v1"}, map[string]agentcore.ChatModel{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := model.(*sessionHeaderModel); ok {
		t.Fatal("provider without session_header must not be wrapped")
	}
}

func TestNewModelSessionIDFormat(t *testing.T) {
	id := NewModelSessionID()
	if len(id) != 36 || id[14] != '4' || strings.Count(id, "-") != 4 {
		t.Fatalf("id = %q, want UUID v4", id)
	}
	if id == NewModelSessionID() {
		t.Fatal("ids must differ")
	}
}

package bootstrap

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

// Một số gateway (vd. OpenCode Go) yêu cầu mỗi cuộc hội thoại gửi một session ID
// ổn định qua header để định tuyến và cache prompt. Header là cấu hình tĩnh của
// provider trong litellm, nên mỗi session cần một client riêng: sessionHeaderModel
// đọc session ID từ ctx rồi dựng (và tái dùng) client mang đúng header đó.

type modelSessionKey struct{}

// WithModelSession gắn session ID cho mọi lời gọi model đi qua ctx. Engine gọi
// một lần cho mỗi lượt worker; provider không khai session_header thì bỏ qua.
func WithModelSession(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, modelSessionKey{}, id)
}

func modelSessionFrom(ctx context.Context) string {
	id, _ := ctx.Value(modelSessionKey{}).(string)
	return id
}

// NewModelSessionID tạo session ID ngẫu nhiên dạng UUID v4.
func NewModelSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// maxCachedSessions giới hạn số client theo session được giữ lại. Engine chạy
// worker tuần tự nên chỉ cần vài slot cho lượt hiện tại và lời gọi chen ngang.
const maxCachedSessions = 4

// sessionHeaderModel chọn client theo session ID trong ctx. Lời gọi không gắn
// session (Arbiter, compaction, tác vụ một lần) dùng base với session mặc định
// sinh lúc khởi tạo.
type sessionHeaderModel struct {
	base  agentcore.ChatModel
	build func(sessionID string) (agentcore.ChatModel, error)

	mu       sync.Mutex
	sessions map[string]agentcore.ChatModel
	order    []string
}

func newSessionHeaderModel(build func(sessionID string) (agentcore.ChatModel, error)) (*sessionHeaderModel, error) {
	base, err := build(NewModelSessionID())
	if err != nil {
		return nil, err
	}
	return &sessionHeaderModel{base: base, build: build, sessions: make(map[string]agentcore.ChatModel)}, nil
}

func (m *sessionHeaderModel) forContext(ctx context.Context) (agentcore.ChatModel, error) {
	id := modelSessionFrom(ctx)
	if id == "" {
		return m.base, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if model, ok := m.sessions[id]; ok {
		return model, nil
	}
	model, err := m.build(id)
	if err != nil {
		return nil, err
	}
	if len(m.order) >= maxCachedSessions {
		delete(m.sessions, m.order[0])
		m.order = m.order[1:]
	}
	m.sessions[id] = model
	m.order = append(m.order, id)
	return model, nil
}

func (m *sessionHeaderModel) Generate(ctx context.Context, messages []agentcore.Message, tools []agentcore.ToolSpec, opts ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	model, err := m.forContext(ctx)
	if err != nil {
		return nil, err
	}
	return model.Generate(ctx, messages, tools, opts...)
}

func (m *sessionHeaderModel) GenerateStream(ctx context.Context, messages []agentcore.Message, tools []agentcore.ToolSpec, opts ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	model, err := m.forContext(ctx)
	if err != nil {
		return nil, err
	}
	return model.GenerateStream(ctx, messages, tools, opts...)
}

func (m *sessionHeaderModel) SupportsTools() bool { return m.base.SupportsTools() }

func (m *sessionHeaderModel) Capabilities() llm.Capabilities {
	if cp, ok := m.base.(llm.CapabilityProvider); ok {
		return cp.Capabilities()
	}
	return llm.Capabilities{}
}

func (m *sessionHeaderModel) Info() llm.ModelInfo {
	if info, ok := m.base.(interface{ Info() llm.ModelInfo }); ok {
		return info.Info()
	}
	return llm.ModelInfo{}
}

func (m *sessionHeaderModel) ProviderName() string {
	if p, ok := m.base.(interface{ ProviderName() string }); ok {
		return p.ProviderName()
	}
	return ""
}

// withSessionHeader trả về bản sao provider extra có thêm header session. Không
// sửa map headers của config vì nhiều client dùng chung nó.
func withSessionHeader(extra map[string]any, header, sessionID string) map[string]any {
	out := cloneMap(extra)
	if out == nil {
		out = make(map[string]any, 1)
	}
	headers := make(map[string]any)
	switch h := out["headers"].(type) {
	case map[string]any:
		for k, v := range h {
			headers[k] = v
		}
	case map[string]string:
		for k, v := range h {
			headers[k] = v
		}
	}
	headers[header] = sessionID
	out["headers"] = headers
	return out
}

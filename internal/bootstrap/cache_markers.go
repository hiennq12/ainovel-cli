package bootstrap

import (
	"context"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
)

// cacheMarkerKey là khóa metadata mà agentcore dùng để gắn breakpoint cache
// (CacheLastMessage / system floor).
const cacheMarkerKey = "cache_control"

// stripsCacheMarkers cho biết provider type có cần bỏ breakpoint cache hay không.
// OpenAI định tuyến cache bằng prompt_cache_key; litellm lại dịch cache_control
// thành prompt_cache_breakpoint, mà nhiều model (vd. gpt-4o-mini) từ chối với
// HTTP 400 "prompt_cache_breakpoint is not supported on this model".
func stripsCacheMarkers(providerType string) bool {
	return providerType == "openai"
}

// noCacheMarkerModel bọc adapter và bỏ cache_control khỏi mọi message trước khi
// gửi đi. Các method khác (Capabilities, ProviderName, ...) được giữ nguyên qua embed.
type noCacheMarkerModel struct {
	*llm.LiteLLMAdapter
}

func (m noCacheMarkerModel) Generate(ctx context.Context, messages []agentcore.Message, tools []agentcore.ToolSpec, opts ...agentcore.CallOption) (*agentcore.LLMResponse, error) {
	return m.LiteLLMAdapter.Generate(ctx, withoutCacheMarkers(messages), tools, opts...)
}

func (m noCacheMarkerModel) GenerateStream(ctx context.Context, messages []agentcore.Message, tools []agentcore.ToolSpec, opts ...agentcore.CallOption) (<-chan agentcore.StreamEvent, error) {
	return m.LiteLLMAdapter.GenerateStream(ctx, withoutCacheMarkers(messages), tools, opts...)
}

// withoutCacheMarkers trả về bản sao messages đã bỏ cache_control; không sửa
// slice/map của caller vì agent loop còn dùng lại chúng.
func withoutCacheMarkers(messages []agentcore.Message) []agentcore.Message {
	var out []agentcore.Message
	for i, msg := range messages {
		if _, ok := msg.Metadata[cacheMarkerKey]; !ok {
			continue
		}
		if out == nil {
			out = make([]agentcore.Message, len(messages))
			copy(out, messages)
		}
		md := make(map[string]any, len(msg.Metadata)-1)
		for k, v := range msg.Metadata {
			if k != cacheMarkerKey {
				md[k] = v
			}
		}
		if len(md) == 0 {
			md = nil
		}
		out[i].Metadata = md
	}
	if out == nil {
		return messages
	}
	return out
}

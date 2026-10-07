package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/store"
)

// ReadBriefTool 回读用户开书时的创作需求原文(RunMeta.StartPrompt),Architect 专用。
//
// 初始规划后,foundation 是规划师自己写的压缩版本:设定集里的逐弧里程碑、伏笔分配、
// 反转揭示时机等细节可能在压缩中丢失。扩弧/续卷/修订大纲时回读原文对照,防止长线漂移。
// 原文可能数万字,所以独立成按需读取的工具,不塞进每次都会调用的 novel_context。
type ReadBriefTool struct {
	store *store.Store
}

func NewReadBriefTool(store *store.Store) *ReadBriefTool {
	return &ReadBriefTool{store: store}
}

func (t *ReadBriefTool) Name() string { return "read_brief" }
func (t *ReadBriefTool) Description() string {
	return "读取用户开书时提交的创作需求原文（设定集/Story Bible）。扩展弧、续卷或修订大纲前用于对照必达节点、伏笔与反转揭示时机"
}
func (t *ReadBriefTool) Label() string { return "读取创作需求" }

func (t *ReadBriefTool) ReadOnly(_ json.RawMessage) bool        { return true }
func (t *ReadBriefTool) ConcurrencySafe(_ json.RawMessage) bool { return true }

func (t *ReadBriefTool) Schema() map[string]any {
	return schema.Object()
}

func (t *ReadBriefTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	meta, err := t.store.RunMeta.Load()
	if err != nil {
		return nil, fmt.Errorf("load run meta: %w", err)
	}
	brief := ""
	if meta != nil {
		brief = strings.TrimSpace(meta.StartPrompt)
	}
	if brief == "" {
		return json.Marshal(map[string]any{
			"available": false,
			"note":      "本书没有记录开书需求原文（例如导入的作品），以现有 foundation 为准",
		})
	}
	return json.Marshal(map[string]any{
		"available": true,
		"chars":     utf8.RuneCountInString(brief),
		"brief":     brief,
	})
}

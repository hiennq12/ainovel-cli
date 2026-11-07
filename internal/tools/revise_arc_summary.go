package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/store"
)

// ReviseArcSummaryTool sửa tóm tắt cung đã lưu sau khi chương trong cung bị viết lại.
// save_arc_summary chỉ ghi lần đầu theo lịch của Router và từ chối ghi đè; tóm tắt chương
// và ledger phục bút thì tự chiếu lại từ chapter record khi commit bản viết lại, riêng tóm
// tắt cung không có đường sửa nên dữ kiện cũ cứ bị nạp vào ngữ cảnh các cung sau.
// Chỉ thay summary và key_events; tiêu đề, snapshot nhân vật và style rules giữ nguyên
// (style_rules.json là bản mới nhất toàn truyện, sửa cung cũ không được kéo lùi nó).
type ReviseArcSummaryTool struct {
	store *store.Store
}

func NewReviseArcSummaryTool(store *store.Store) *ReviseArcSummaryTool {
	return &ReviseArcSummaryTool{store: store}
}

func (t *ReviseArcSummaryTool) Name() string { return "revise_arc_summary" }
func (t *ReviseArcSummaryTool) Description() string {
	return LocalizeText(t.store, "修订已保存的弧摘要，使其与返工后的正文一致：整体替换 summary 和 key_events，保留标题、角色快照和写作规则。仅在用户要求或旧摘要与现有正文矛盾时调用")
}
func (t *ReviseArcSummaryTool) Label() string { return "Sửa tóm tắt cung" }

func (t *ReviseArcSummaryTool) ReadOnly(_ json.RawMessage) bool        { return false }
func (t *ReviseArcSummaryTool) ConcurrencySafe(_ json.RawMessage) bool { return false }

func (t *ReviseArcSummaryTool) Schema() map[string]any {
	return localizeSchema(t.store, schema.Object(
		schema.Property("volume", schema.Int("卷号")).Required(),
		schema.Property("arc", schema.Int("弧号")).Required(),
		schema.Property("summary", schema.String("修订后的完整弧摘要（500字以内），整段替换旧摘要")).Required(),
		schema.Property("key_events", schema.Array("修订后的完整关键事件列表，整体替换旧列表", schema.String(""))).Required(),
		schema.Property("reason", schema.String("本次修订原因")).Required(),
	))
}

func (t *ReviseArcSummaryTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Volume    int      `json:"volume"`
		Arc       int      `json:"arc"`
		Summary   string   `json:"summary"`
		KeyEvents []string `json:"key_events"`
		Reason    string   `json:"reason"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w: %w", errs.ErrToolArgs, err)
	}
	if a.Volume <= 0 || a.Arc <= 0 {
		return nil, fmt.Errorf("volume and arc must be > 0: %w", errs.ErrToolArgs)
	}
	if strings.TrimSpace(a.Summary) == "" {
		return nil, fmt.Errorf("summary 不能为空: %w", errs.ErrToolArgs)
	}
	if strings.TrimSpace(a.Reason) == "" {
		return nil, fmt.Errorf("reason 不能为空: %w", errs.ErrToolArgs)
	}
	existing, err := t.store.Summaries.LoadArcSummary(a.Volume, a.Arc)
	if err != nil {
		return nil, fmt.Errorf("load arc summary: %w: %w", errs.ErrStoreRead, err)
	}
	if existing == nil {
		return nil, fmt.Errorf("第 %d 卷第 %d 弧尚无摘要，无可修订；弧结束时的首份摘要由 save_arc_summary 生成: %w",
			a.Volume, a.Arc, errs.ErrToolPrecondition)
	}
	last, err := lastCompletedChapter(t.store)
	if err != nil {
		return nil, err
	}
	if err := rejectCJKLeak(t.store, t.Name(), append([]string{a.Summary}, a.KeyEvents...), 1, last); err != nil {
		return nil, err
	}

	existing.Summary = a.Summary
	existing.KeyEvents = a.KeyEvents
	if err := t.store.Summaries.SaveArcSummary(*existing); err != nil {
		return nil, fmt.Errorf("save arc summary: %w: %w", errs.ErrStoreWrite, err)
	}
	// Bước "arc_summary" để EditorStopGuard nhận đây là sản phẩm đã lưu của task.
	path := fmt.Sprintf("summaries/arc-v%02da%02d.json", a.Volume, a.Arc)
	if _, err := t.store.Checkpoints.AppendArtifacts(
		domain.ArcScope(a.Volume, a.Arc), "arc_summary", path,
	); err != nil {
		return nil, fmt.Errorf("checkpoint arc summary: %w: %w", errs.ErrStoreWrite, err)
	}
	return json.Marshal(map[string]any{
		"saved": true, "type": "arc_summary_revision",
		"volume": a.Volume, "arc": a.Arc, "reason": a.Reason,
	})
}

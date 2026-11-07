package tools

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
)

// Tóm tắt cung đã lưu phải sửa được sau khi chương bị viết lại (save_arc_summary từ chối ghi đè),
// nhưng chỉ phần summary/key_events: tiêu đề giữ nguyên, và cung chưa có tóm tắt thì không tạo hộ.
func TestReviseArcSummaryReplacesSummaryOnly(t *testing.T) {
	s := setupArcSummaryStore(t)
	tool := NewReviseArcSummaryTool(s)
	args := func(arc int) json.RawMessage {
		raw, _ := json.Marshal(map[string]any{
			"volume": 1, "arc": arc,
			"summary":    "Dòng bút chì trong sổ bìa nâu, cửa sắt có vết hàn mới.",
			"key_events": []string{"Duy thấy dòng bút chì"},
			"reason":     "Chương 8 đã viết lại: bỏ dòng sơn trên cửa",
		})
		return raw
	}

	// Cung 2 đã kết thúc nhưng chưa có tóm tắt: đó là việc của save_arc_summary.
	if _, err := tool.Execute(context.Background(), args(2)); !errors.Is(err, errs.ErrToolPrecondition) {
		t.Fatalf("cung chưa có tóm tắt phải bị từ chối, got: %v", err)
	}

	if _, err := tool.Execute(context.Background(), args(1)); err != nil {
		t.Fatalf("revise: %v", err)
	}
	got, _ := s.Summaries.LoadArcSummary(1, 1)
	if got.Title != "第一弧" || got.Summary != "Dòng bút chì trong sổ bìa nâu, cửa sắt có vết hàn mới." ||
		!slices.Equal(got.KeyEvents, []string{"Duy thấy dòng bút chì"}) {
		t.Fatalf("tóm tắt sau sửa sai: %+v", got)
	}
	if !slices.ContainsFunc(s.Checkpoints.All(), func(cp domain.Checkpoint) bool { return cp.Step == "arc_summary" }) {
		t.Fatal("thiếu checkpoint arc_summary, EditorStopGuard sẽ không cho editor kết thúc")
	}
}

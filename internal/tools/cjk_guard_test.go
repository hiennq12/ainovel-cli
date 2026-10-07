package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

// newReviewStore dựng store có chương 3 đã hoàn thành với nguyên văn chapterText.
func newReviewStore(t *testing.T, lang, chapterText string) *store.Store {
	t.Helper()
	s := store.NewStore(t.TempDir())
	s.SetLanguage(lang)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("Progress.Init: %v", err)
	}
	if err := s.Progress.MarkChapterComplete(3, 3000, "", ""); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}
	if err := s.Drafts.SaveFinalChapter(3, chapterText); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	return s
}

func reviewArgs(t *testing.T, description, evidence string) json.RawMessage {
	t.Helper()
	args, err := json.Marshal(map[string]any{
		"chapter":    3,
		"scope":      "chapter",
		"dimensions": []map[string]any{{"dimension": "consistency", "score": 80, "comment": "Nhất quán"}},
		"issues": []map[string]any{{
			"type": "aesthetic", "severity": "warning", "description": description, "evidence": evidence,
			"suggestion": nil, "chapters": []int{3}, "requires_change": true,
		}},
		"contract_status": nil,
		"contract_misses": []string{},
		"contract_notes":  nil,
		"verdict":         "polish",
		"summary":         "Chương ổn, cần trau chuốt một chỗ.",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return args
}

func TestSaveReviewRejectsCJKLeakInVietnameseBook(t *testing.T) {
	s := newReviewStore(t, "vi", "Bà Vượng đứng ở hiên, tay cầm cái chổi cùn.")
	// Mẫu thật từ editor qwen3.8-max trên gpmb: "tự洽" và thuật ngữ "返工".
	_, err := NewSaveReviewTool(s).Execute(context.Background(),
		reviewArgs(t, "Thời tuyến nội bộ vẫn tự洽 nhưng cần 返工 đoạn kết.", "Ch3: 'tay cầm cái chổi cùn'"))
	if err == nil {
		t.Fatal("review lẫn chữ Hán phải bị từ chối")
	}
	for _, want := range []string{`"洽"`, `"返工"`, "save_review"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("lỗi phải nêu %s để model tự sửa, got: %v", want, err)
		}
	}
	if review, _ := s.World.LoadReview(3); review != nil {
		t.Fatalf("review bị từ chối không được lưu, got %+v", review)
	}
}

func TestSaveReviewAllowsCJKQuotedFromChapterText(t *testing.T) {
	// Chính chương bị lẫn chữ Hán: editor phải trích được nguyên văn làm bằng chứng.
	s := newReviewStore(t, "vi", "Anh quan sát草木 hai bên đường.")
	if _, err := NewSaveReviewTool(s).Execute(context.Background(),
		reviewArgs(t, "Chương lẫn chữ Hán trong lời kể.", "Ch3: 'Anh quan sát草木 hai bên đường.'")); err != nil {
		t.Fatalf("trích nguyên văn chương phải được chấp nhận: %v", err)
	}
}

func TestSaveReviewAllowsChineseReviewInChineseBook(t *testing.T) {
	s := newReviewStore(t, "zh", "林砚站在山门前。")
	if _, err := NewSaveReviewTool(s).Execute(context.Background(),
		reviewArgs(t, "章末钩子不够具体", "林砚站在山门前")); err != nil {
		t.Fatalf("tác phẩm tiếng Trung không bị chặn: %v", err)
	}
}

func TestContextToolReviewLessonsFollowBookLanguage(t *testing.T) {
	s := store.NewStore(t.TempDir())
	s.SetLanguage("vi")
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Outline.SaveOutline([]domain.OutlineEntry{
		{Chapter: 1, Title: "Ngày lành", CoreEvent: "Trung Thực nhận lịch bàn giao"},
		{Chapter: 2, Title: "Hai bản quy hoạch", CoreEvent: "Giá đất tách đôi"},
	}); err != nil {
		t.Fatalf("SaveOutline: %v", err)
	}
	if err := s.Progress.Init(5); err != nil {
		t.Fatalf("Progress.Init: %v", err)
	}
	if err := s.World.SaveReview(domain.ReviewEntry{
		Chapter: 1, Scope: "chapter", Verdict: "polish", Summary: "Cần trau chuốt.",
		ContractStatus: "partial", ContractMisses: []string{"Chưa gieo lời mời đối thoại"},
		Issues: []domain.ConsistencyIssue{{Type: "hook", Severity: "warning", Description: "Điểm móc cuối chương chưa cụ thể"}},
	}); err != nil {
		t.Fatalf("SaveReview: %v", err)
	}

	args, _ := json.Marshal(map[string]any{"chapter": 2})
	raw, err := newTestContextTool(s, References{}, "default").Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload struct {
		Selected struct {
			ReviewLessons []domain.RecallItem `json:"review_lessons"`
		} `json:"selected_memory"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	lessons := payload.Selected.ReviewLessons
	if !containsRecallSummary(lessons, "Chương 1 thiếu mục hợp đồng: Chưa gieo lời mời đối thoại") ||
		!containsRecallSummary(lessons, "Nhắc từ review chương 1: Điểm móc cuối chương chưa cụ thể") {
		t.Fatalf("review_lessons phải dùng nhãn tiếng Việt, got %+v", lessons)
	}
	for _, item := range lessons {
		if frags := rules.CJKFragments(item.Summary + item.Reason); len(frags) > 0 {
			t.Fatalf("review_lessons của tác phẩm tiếng Việt còn chữ Hán %v: %+v", frags, item)
		}
	}
}

func vietnameseArcSummaryArgs(t *testing.T, motivation string) json.RawMessage {
	t.Helper()
	args, err := json.Marshal(map[string]any{
		"volume": 1, "arc": 2, "title": "Vào núi",
		"summary":    "Nhân vật chính qua thử thách vào núi, xác nhận hướng truy tìm tiếp theo.",
		"key_events": []string{"Qua thử thách", "Phát hiện manh mối vụ án cũ"},
		"character_snapshots": []map[string]any{
			{"name": "Thẩm Uyên", "status": "Còn sống", "motivation": motivation},
		},
		"style_rules": map[string]any{
			"prose":    []string{"Tả cảnh ưu tiên xúc giác và khứu giác"},
			"dialogue": []map[string]any{{"name": "Thẩm Uyên", "rules": []string{"Thoại cực ngắn"}}},
			"taboos":   []string{"Tránh độc thoại dài cuối chương"},
		},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return args
}

func TestSaveArcSummaryRejectsCJKLeakInVietnameseBook(t *testing.T) {
	s := setupArcSummaryStore(t)
	s.SetLanguage("vi")
	_, err := NewSaveArcSummaryTool(s).Execute(context.Background(), vietnameseArcSummaryArgs(t, "Truy vụ án cũ, 伏笔 chưa thu"))
	if err == nil || !strings.Contains(err.Error(), `"伏笔"`) || !strings.Contains(err.Error(), "save_arc_summary") {
		t.Fatalf("tóm tắt cung lẫn chữ Hán phải bị từ chối kèm đoạn vi phạm, got: %v", err)
	}
	if summary, _ := s.Summaries.LoadArcSummary(1, 2); summary != nil {
		t.Fatalf("tóm tắt bị từ chối không được lưu, got %+v", summary)
	}

	// Sau khi model viết lại bằng tiếng Việt, cùng lời gọi phải qua.
	if _, err := NewSaveArcSummaryTool(s).Execute(context.Background(), vietnameseArcSummaryArgs(t, "Truy vụ án cũ")); err != nil {
		t.Fatalf("tóm tắt sạch phải được lưu: %v", err)
	}
}

func TestSaveArcSummaryAllowsCJKQuotedFromEarlierChapter(t *testing.T) {
	s := setupArcSummaryStore(t)
	s.SetLanguage("vi")
	// Chữ Hán lẫn ở chương 1 (cung trước): snapshot nhân vật vẫn được trích nguyên văn.
	if err := s.Drafts.SaveFinalChapter(1, "Thẩm Uyên lẩm bẩm 天命 rồi bỏ đi."); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	if _, err := NewSaveArcSummaryTool(s).Execute(context.Background(),
		vietnameseArcSummaryArgs(t, "Vẫn ám ảnh câu '天命' ở chương 1")); err != nil {
		t.Fatalf("trích nguyên văn chương trước phải được chấp nhận: %v", err)
	}
}

func TestSaveVolumeSummaryRejectsCJKLeakInVietnameseBook(t *testing.T) {
	s := setupVolumeSummaryStore(t)
	s.SetLanguage("vi")
	args := func(summary string) json.RawMessage {
		raw, err := json.Marshal(map[string]any{
			"volume": 1, "title": "Quyển cuối", "summary": summary, "key_events": []string{"Kết cục"},
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		return raw
	}
	_, err := NewSaveVolumeSummaryTool(s).Execute(context.Background(), args("Mọi tuyến đều 收束 ở chương cuối。"))
	if err == nil || !strings.Contains(err.Error(), `"收束"`) || !strings.Contains(err.Error(), `"。"`) {
		t.Fatalf("tóm tắt quyển lẫn chữ Hán/dấu câu CJK phải bị từ chối, got: %v", err)
	}
	if summary, _ := s.Summaries.LoadVolumeSummary(1); summary != nil {
		t.Fatalf("tóm tắt bị từ chối không được lưu, got %+v", summary)
	}
	if _, err := NewSaveVolumeSummaryTool(s).Execute(context.Background(), args("Mọi tuyến đều khép lại ở chương cuối.")); err != nil {
		t.Fatalf("tóm tắt sạch phải được lưu: %v", err)
	}
}

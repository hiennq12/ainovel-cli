package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

// Mọi trường văn bản của bảng tiếng Việt phải có giá trị và không chứa chữ Hán:
// thêm trường mới mà quên dịch thì test này báo.
func TestContextTextVIIsCompleteAndHanFree(t *testing.T) {
	v := reflect.ValueOf(contextTextVI)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if field.Type.Kind() != reflect.String {
			continue
		}
		value := v.Field(i).String()
		if value == "" {
			t.Errorf("contextTextVI.%s chưa có bản dịch", field.Name)
		}
		if frags := rules.CJKFragments(value); len(frags) > 0 {
			t.Errorf("contextTextVI.%s còn chữ Hán %v", field.Name, frags)
		}
	}
	for zh, vi := range contextTextVI.memoryPolicy {
		if frags := rules.CJKFragments(vi); len(frags) > 0 {
			t.Errorf("bản dịch memory_policy của %q còn chữ Hán %v", zh, frags)
		}
	}
}

// Mọi chuỗi mô tả mà domain sinh ra cho memory_policy đều phải có bản dịch.
func TestLocalizePolicyCoversAllDomainText(t *testing.T) {
	policies := []domain.MemoryPolicy{
		domain.NewChapterMemoryPolicy(nil, domain.ContextProfile{}, true),
		domain.NewChapterMemoryPolicy(nil, domain.ContextProfile{Layered: true}, true),
		domain.NewArchitectMemoryPolicy(),
	}
	for _, p := range policies {
		raw, err := json.Marshal(contextTextFor("vi").localizePolicy(p))
		if err != nil {
			t.Fatal(err)
		}
		if frags := rules.CJKFragments(string(raw)); len(frags) > 0 {
			t.Errorf("memory_policy mode=%s còn chữ Hán chưa dịch %v", p.Mode, frags)
		}
	}
	zh := domain.NewArchitectMemoryPolicy()
	if got := contextTextFor("zh").localizePolicy(zh); got != zh {
		t.Errorf("tác phẩm tiếng Trung phải giữ nguyên memory_policy")
	}
}

// Payload novel_context của tác phẩm tiếng Việt không được chứa chữ Hán do hệ thống tự thêm,
// kể cả user_rules dựng từ baseline mặc định (rules.SystemDefaultsFor).
func TestContextPayloadOfVietnameseBookIsHanFree(t *testing.T) {
	s := store.NewStore(t.TempDir())
	s.SetLanguage("vi")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.Init())
	var outline []domain.OutlineEntry
	for ch := 1; ch <= 12; ch++ {
		outline = append(outline, domain.OutlineEntry{Chapter: ch, Title: fmt.Sprintf("Chương %d", ch),
			CoreEvent: "Trung Thực truy hồ sơ thửa 47 cùng bà Vượng", Hook: "Ai ký báo cáo", Scenes: []string{"Kho hồ sơ"}})
	}
	must(s.Outline.SaveOutline(outline))
	must(s.Progress.Init(12))
	for ch := 1; ch <= 11; ch++ {
		must(s.Progress.MarkChapterComplete(ch, 2000, "", ""))
		must(s.Drafts.SaveFinalChapter(ch, "Trung Thực mở sổ tay."))
	}
	must(s.Characters.Save([]domain.Character{{Name: "Trung Thực", Role: "chính"}, {Name: "bà Vượng", Role: "phụ"}}))
	must(s.World.SaveForeshadowLedger([]domain.ForeshadowEntry{{ID: "thua_47", Description: "Hồ sơ thửa 47 có hai bản", PlantedAt: 1, Status: "planted"}}))
	must(s.World.SaveRelationships([]domain.RelationshipEntry{{CharacterA: "Trung Thực", CharacterB: "bà Vượng", Relation: "nghi ngại", Chapter: 1}}))
	must(s.World.SaveReview(domain.ReviewEntry{Chapter: 11, Scope: "chapter", Verdict: "accept", Summary: "Ổn",
		Issues: []domain.ConsistencyIssue{{Type: "hook", Severity: "warning", Description: "Điểm móc chưa rõ"}}}))

	tool := newTestContextTool(s, References{}, "default")
	for _, chapter := range []int{12, 0} { // writer/editor và architect
		args, _ := json.Marshal(map[string]any{"chapter": chapter})
		raw, err := tool.Execute(context.Background(), args)
		must(err)
		var payload map[string]any
		must(json.Unmarshal(raw, &payload))
		if frags := rules.CJKFragments(string(raw)); len(frags) > 0 {
			t.Errorf("chapter=%d: payload tiếng Việt còn chữ Hán %v\nsummary: %v", chapter, frags, payload["_loading_summary"])
		}
	}
}

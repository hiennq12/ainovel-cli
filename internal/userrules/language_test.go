package userrules

import (
	"testing"

	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

func newLanguageService(t *testing.T, lang string) (*Service, *store.Store) {
	t.Helper()
	st := store.NewStore(t.TempDir())
	st.SetLanguage(lang)
	return NewService(st, nil, rules.LoadOptions{}), st
}

func hanFree(t *testing.T, s rules.Structured) {
	t.Helper()
	if _, changed := rules.WithoutHanEntries(s); changed {
		t.Fatalf("luật của tác phẩm tiếng Việt còn mục chữ Hán: %+v", s)
	}
}

func TestService_BuildVietnameseBookSkipsChineseDefaults(t *testing.T) {
	svc, _ := newLanguageService(t, "vi")
	snap, err := svc.Build(t.Context(), "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	hanFree(t, snap.Structured)
	if len(snap.Sources) == 0 || snap.Sources[0] != "system_defaults" {
		t.Fatalf("system_defaults vẫn phải là nguồn đầu tiên, got %v", snap.Sources)
	}
}

// Snapshot cũ (như gpmb) đã gộp baseline tiếng Trung: mở lại truyện thì tự làm sạch và ghi lại,
// giữ nguyên các mục tiếng Việt.
func TestService_GetOrBuildCleansLegacyVietnameseSnapshot(t *testing.T) {
	svc, st := newLanguageService(t, "vi")
	legacy := rules.BuildSnapshot([]rules.Candidate{
		rules.SystemDefaults(),
		{Source: "project:giong-van.md", Structured: rules.Structured{
			ForbiddenPhrases: []string{"trớ trêu thay"},
			FatigueWords:     map[string]int{"dường như": 2},
		}},
	})
	if err := st.UserRules.Save(&legacy); err != nil {
		t.Fatalf("Save: %v", err)
	}

	snap, err := svc.GetOrBuild(t.Context())
	if err != nil {
		t.Fatalf("GetOrBuild: %v", err)
	}
	hanFree(t, snap.Structured)
	if snap.Structured.FatigueWords["dường như"] != 2 || len(snap.Structured.ForbiddenPhrases) != 1 {
		t.Fatalf("phải giữ mục tiếng Việt, got %+v", snap.Structured)
	}
	reloaded, err := st.UserRules.Load()
	if err != nil || reloaded == nil {
		t.Fatalf("Load: %v", err)
	}
	hanFree(t, reloaded.Structured)
}

func TestService_GetOrBuildLeavesChineseBookUntouched(t *testing.T) {
	svc, st := newLanguageService(t, "zh")
	legacy := rules.BuildSnapshot([]rules.Candidate{rules.SystemDefaults()})
	if err := st.UserRules.Save(&legacy); err != nil {
		t.Fatalf("Save: %v", err)
	}
	snap, err := svc.GetOrBuild(t.Context())
	if err != nil {
		t.Fatalf("GetOrBuild: %v", err)
	}
	if len(snap.Structured.FatigueWords) != 16 {
		t.Fatalf("tác phẩm tiếng Trung phải giữ nguyên baseline, got %+v", snap.Structured)
	}
}

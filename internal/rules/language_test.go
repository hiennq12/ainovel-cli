package rules

import (
	"reflect"
	"testing"
)

func TestSystemDefaultsForLanguage(t *testing.T) {
	vi := SystemDefaultsFor("vi")
	if vi.Source != "system_defaults" {
		t.Fatalf("source = %q", vi.Source)
	}
	// Tiếng Việt chưa có danh sách riêng: không được mượn danh sách tiếng Trung.
	if !vi.Structured.IsEmpty() {
		t.Fatalf("mặc định tiếng Việt phải rỗng cho tới khi chốt danh sách, got %+v", vi.Structured)
	}
	for _, lang := range []string{"zh", ""} {
		if got := SystemDefaultsFor(lang); !reflect.DeepEqual(got, SystemDefaults()) {
			t.Fatalf("lang=%q phải giữ baseline tiếng Trung", lang)
		}
	}
}

func TestWithoutHanEntries(t *testing.T) {
	// Mẫu thật từ user_rules.json của gpmb: từ tiếng Việt trộn với baseline tiếng Trung.
	in := Structured{
		Genre:            "satire",
		ForbiddenChars:   []string{"…", "。"},
		ForbiddenPhrases: []string{"trớ trêu thay", "值得注意的是"},
		FatigueWords:     map[string]int{"dường như": 2, "khẽ": 2, "仿佛": 2, "一丝": 2},
	}
	got, changed := WithoutHanEntries(in)
	if !changed {
		t.Fatal("có mục chữ Hán thì phải báo changed")
	}
	want := Structured{
		Genre:            "satire",
		ForbiddenChars:   []string{"…"},
		ForbiddenPhrases: []string{"trớ trêu thay"},
		FatigueWords:     map[string]int{"dường như": 2, "khẽ": 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if len(in.FatigueWords) != 4 {
		t.Fatal("không được sửa map đầu vào")
	}

	clean, changed := WithoutHanEntries(want)
	if changed || !reflect.DeepEqual(clean, want) {
		t.Fatalf("đầu vào sạch phải giữ nguyên, got %+v changed=%v", clean, changed)
	}
}

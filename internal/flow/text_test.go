package flow

import (
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	hanRe = regexp.MustCompile(`\p{Han}|[、，。；：（）「」]`)
	// Tên tool / tham số / khoá JSON trong lệnh: phải giữ nguyên khi dịch.
	identRe  = regexp.MustCompile(`[a-z][a-z0-9]*(?:_[a-z0-9]+)+|[a-z_]+=[a-z0-9_]+`)
	numberRe = regexp.MustCompile(`\d+`)
)

// assertVietnameseTwin so một chỉ thị tiếng Việt với bản gốc tiếng Trung của cùng State:
// ngôn ngữ chỉ đổi chữ, không đổi quyết định; lệnh tiếng Việt không còn chữ Hán và giữ
// đủ tên tool, tham số, số chương.
func assertVietnameseTwin(t *testing.T, label string, zh, vi *Instruction) {
	t.Helper()
	if (zh == nil) != (vi == nil) {
		t.Fatalf("%s: ngôn ngữ làm đổi quyết định: zh=%+v vi=%+v", label, zh, vi)
	}
	if zh == nil {
		return
	}
	if zh.Agent != vi.Agent || zh.Chapter != vi.Chapter {
		t.Fatalf("%s: ngôn ngữ làm đổi agent/chương: zh=%+v vi=%+v", label, zh, vi)
	}
	for field, text := range map[string]string{"Task": vi.Task, "Reason": vi.Reason} {
		if m := hanRe.FindString(text); m != "" {
			t.Fatalf("%s: %s tiếng Việt còn chữ Hán/dấu câu CJK %q: %s", label, field, m, text)
		}
	}
	for _, id := range identRe.FindAllString(zh.Task, -1) {
		if !strings.Contains(vi.Task, id) {
			t.Fatalf("%s: Task tiếng Việt mất %q\nzh: %s\nvi: %s", label, id, zh.Task, vi.Task)
		}
	}
	zhNums, viNums := numberRe.FindAllString(zh.Task, -1), numberRe.FindAllString(vi.Task, -1)
	slices.Sort(zhNums)
	slices.Sort(viNums)
	if !reflect.DeepEqual(zhNums, viNums) {
		t.Fatalf("%s: số trong Task lệch\nzh: %s\nvi: %s", label, zh.Task, vi.Task)
	}
}

// Mọi trường của bảng tiếng Việt phải có giá trị: thêm trường mà quên dịch thì báo.
func TestRouteTextVIIsComplete(t *testing.T) {
	v := reflect.ValueOf(routeTextVI)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).String() == "" {
			t.Errorf("routeTextVI.%s chưa có bản dịch", v.Type().Field(i).Name)
		}
		if m := hanRe.FindString(v.Field(i).String()); m != "" {
			t.Errorf("routeTextVI.%s còn chữ Hán %q", v.Type().Field(i).Name, m)
		}
	}
}

// Prompt tiếng Việt nhận diện loại nhiệm vụ qua cụm từ trong Task. Đổi cụm ở bảng mà quên
// prompt (hoặc ngược lại) thì Worker không vào đúng chế độ.
func TestVietnamesePromptTriggersMatchRouteText(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		raw, err := os.ReadFile("../../assets/prompts/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return strings.ToLower(string(raw))
	}
	editor, architectLong := read("editor.md"), read("architect-long.md")
	cases := []struct {
		prompt, promptName, phrase string
		tasks                      []string
	}{
		{editor, "editor.md", "review cấp cung", []string{routeTextVI.arcReviewFmt, routeTextVI.aggArcReviewFmt}},
		{editor, "editor.md", "tóm tắt tập", []string{routeTextVI.volSummaryFmt}},
		{architectLong, "architect-long.md", "tạo tập tiếp theo", []string{routeTextVI.newVolumeTask}},
		{architectLong, "architect-long.md", "expand_arc", []string{routeTextVI.expandArcFmt}},
	}
	for _, c := range cases {
		if !strings.Contains(c.prompt, `"`+c.phrase+`"`) && !strings.Contains(c.prompt, "`"+c.phrase+"`") {
			t.Errorf("%s không nhắc cụm kích hoạt %q", c.promptName, c.phrase)
		}
		for _, task := range c.tasks {
			if !strings.Contains(strings.ToLower(task), c.phrase) {
				t.Errorf("Task %q thiếu cụm kích hoạt %q mà %s dùng", task, c.phrase, c.promptName)
			}
		}
	}
}

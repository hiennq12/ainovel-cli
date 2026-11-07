package host

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Văn bản Engine gửi model cho tác phẩm tiếng Việt: đủ trường, không chữ Hán, giữ nguyên
// tên tool/tham số của bản gốc.
func TestWorkerTextVIIsCompleteAndHanFree(t *testing.T) {
	han := regexp.MustCompile(`\p{Han}|[、，。；：（）]`)
	ident := regexp.MustCompile(`[a-z][a-z0-9]*(?:_[a-z0-9]+)+|[a-z_]+=[a-z0-9_]+`)
	vi, zh := reflect.ValueOf(workerTextVI), reflect.ValueOf(workerTextZH)
	for i := 0; i < vi.NumField(); i++ {
		name, text := vi.Type().Field(i).Name, vi.Field(i).String()
		if text == "" {
			t.Errorf("workerTextVI.%s chưa có bản dịch", name)
		}
		if m := han.FindString(text); m != "" {
			t.Errorf("workerTextVI.%s còn chữ Hán %q", name, m)
		}
		for _, id := range ident.FindAllString(zh.Field(i).String(), -1) {
			if !strings.Contains(text, id) {
				t.Errorf("workerTextVI.%s mất %q", name, id)
			}
		}
	}
	if strings.Count(workerTextVI.expandArcTaskFmt, "%s") != strings.Count(workerTextZH.expandArcTaskFmt, "%s") {
		t.Errorf("expandArcTaskFmt lệch số tham số định dạng")
	}
	got := interventionDispatchTask(&workerTextVI, "Kiểm tra lặp", "Đừng giải thích lại nguồn gốc năng lực.")
	if !strings.HasSuffix(got, "\nĐừng giải thích lại nguồn gốc năng lực.") || !strings.Contains(got, "nguồn ủy quyền") {
		t.Errorf("lời bọc can thiệp tiếng Việt sai: %q", got)
	}
}

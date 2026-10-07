package tools

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

// rejectCJKLeak chặn sản phẩm của editor (review, tóm tắt cung/quyển) lẫn chữ Hán / dấu câu
// CJK khi tác phẩm viết bằng tiếng Việt. Các sản phẩm này được novel_context đưa lại vào
// ngữ cảnh writer, nên chữ Hán ở đây kéo writer lệch ngôn ngữ.
//
// Ngoại lệ: đoạn có thật trong nguyên văn các chương from..to — editor phải trích được bằng
// chứng khi chính chương bị lẫn chữ Hán. Chỉ đọc chương khi đã thấy đoạn CJK, nên đường
// chạy bình thường không tốn IO. Lỗi trả về liệt kê đúng các đoạn vi phạm để model tự sửa.
func rejectCJKLeak(st *store.Store, tool string, texts []string, from, to int) error {
	if st.Language() != "vi" {
		return nil
	}
	fragments := cjkFragments(texts)
	if len(fragments) == 0 {
		return nil
	}

	var source strings.Builder
	for ch := max(from, 1); ch <= to; ch++ {
		text, err := st.Drafts.LoadChapterText(ch)
		if err != nil {
			return fmt.Errorf("load chapter %d for CJK check: %w: %w", ch, errs.ErrStoreRead, err)
		}
		source.WriteString(text)
	}
	sourceText := source.String()

	var leaked []string
	for _, f := range fragments {
		if !strings.Contains(sourceText, f) {
			leaked = append(leaked, strconv.Quote(f))
		}
	}
	if len(leaked) == 0 {
		return nil
	}
	return fmt.Errorf("%s lẫn chữ Hán/dấu câu CJK không có trong nguyên văn chương: %s. "+
		"Tác phẩm viết bằng tiếng Việt: viết lại các trường chứa những đoạn này hoàn toàn bằng tiếng Việt "+
		"(chỉ được giữ chữ Hán khi trích nguyên văn chương), rồi gọi lại %s: %w",
		tool, strings.Join(leaked, ", "), tool, errs.ErrToolArgs)
}

// cjkFragments gom các đoạn CJK không trùng, theo thứ tự xuất hiện.
func cjkFragments(texts []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, text := range texts {
		for _, f := range rules.CJKFragments(text) {
			if _, ok := seen[f]; ok {
				continue
			}
			seen[f] = struct{}{}
			out = append(out, f)
		}
	}
	return out
}

// lastCompletedChapter là chương lớn nhất đã hoàn thành; 0 nếu chưa có chương nào.
// Tóm tắt cung/quyển có thể nhắc tới bất kỳ chương nào trước đó (snapshot nhân vật,
// quy tắc văn phong), nên ngoại lệ trích nguyên văn xét toàn bộ 1..chương này.
func lastCompletedChapter(st *store.Store) (int, error) {
	progress, err := st.Progress.Load()
	if err != nil {
		return 0, fmt.Errorf("load progress: %w: %w", errs.ErrStoreRead, err)
	}
	if progress == nil || len(progress.CompletedChapters) == 0 {
		return 0, nil
	}
	return slices.Max(progress.CompletedChapters), nil
}

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/store"
)

// localizeSourceDirs là các package sinh văn bản đi vào mô tả, schema, kết quả hoặc lỗi của tool.
var localizeSourceDirs = []string{".", "../chapterfacts", "../domain", "../agents/guard"}

// localizeSkipFiles không chứa văn bản tool: bảng tiếng Trung gốc của novel_context (đã có bảng
// tiếng Việt riêng), chính bảng dịch, và regex nhận dạng tiêu đề chương.
var localizeSkipFiles = map[string]bool{
	"novel_context_text.go": true,
	"localize_vi.go":        true,
	"chapter_heading.go":    true,
}

// Mọi chuỗi tiếng Trung trong mã nguồn tool (trừ log) phải dịch được hết sang tiếng Việt.
// Thêm chuỗi mới mà quên bảng dịch thì test liệt kê đúng chuỗi và vị trí.
func TestToolTextVICoversSource(t *testing.T) {
	var missing []string
	fset := token.NewFileSet()
	for _, dir := range localizeSourceDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") || localizeSkipFiles[filepath.Base(path)] {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parser.ParseFile(fset, path, src, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok && (isLogCall(call) || isMatchCall(call)) {
					return false
				}
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(lit.Value)
				if err != nil || !hanRe.MatchString(text) {
					return true
				}
				sample := fmtVerbRe.ReplaceAllStringFunc(text, func(verb string) string {
					switch verb[len(verb)-1] {
					case '%':
						return "%"
					case 'd':
						return "7"
					case 'f':
						return "1.5"
					default:
						return "x"
					}
				})
				if hanRe.MatchString(translateVI(sample)) {
					missing = append(missing, fmt.Sprintf("%s\t%q", fset.Position(lit.Pos()), text))
				}
				return true
			})
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("%d chuỗi tiếng Trung chưa có trong toolTextVI:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}

func isLogCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "slog"
}

// isMatchCall bỏ qua strings.Contains: chuỗi trong đó là khóa so khớp task, không gửi cho model.
func isMatchCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "strings" && sel.Sel.Name == "Contains"
}

// Bản dịch có thể nằm trong chuỗi JSON của kết quả tool: không thêm " hay \ so với bản gốc
// (dùng “ ”). Phải dùng đúng số tham số; tham số ở đầu/cuối bản gốc khớp lười với văn bản
// xung quanh, nên bản dịch phải giữ nó ở đúng đầu/cuối để không đảo lộn phần văn bản đó.
func TestToolTextVIEntriesAreWellFormed(t *testing.T) {
	verbs := func(s string) []string {
		var out []string
		for _, v := range fmtVerbRe.FindAllString(s, -1) {
			if v != "%%" {
				out = append(out, v)
			}
		}
		return out
	}
	for zh, vi := range toolTextVI {
		if vi == "" {
			t.Errorf("chưa dịch: %q", zh)
			continue
		}
		if hanRe.MatchString(vi) {
			t.Errorf("bản dịch còn chữ Hán: %q", vi)
		}
		for _, c := range []string{`"`, `\`} {
			if strings.Count(vi, c) > strings.Count(zh, c) {
				t.Errorf("bản dịch thêm %s so với bản gốc (dùng “ ”): %q", c, vi)
			}
		}
		if len(verbs(zh)) != len(verbs(vi)) {
			t.Errorf("số tham số lệch: %q → %q", zh, vi)
		}
		if loc := fmtVerbRe.FindStringIndex(zh); loc != nil && loc[0] == 0 && !strings.HasPrefix(vi, zh[:loc[1]]) {
			t.Errorf("bản gốc bắt đầu bằng tham số thì bản dịch cũng phải vậy: %q → %q", zh, vi)
		}
		if zv, vv := verbs(zh), verbs(vi); len(zv) > 0 && strings.HasSuffix(zh, zv[len(zv)-1]) && !strings.HasSuffix(vi, vv[len(vv)-1]) {
			t.Errorf("bản gốc kết thúc bằng tham số thì bản dịch cũng phải vậy: %q → %q", zh, vi)
		}
	}
}

// Bản dịch đổi được trật tự tham số (%[n]), lỗi lồng ": %w" vẫn dịch, errors.Is giữ nguyên;
// tác phẩm tiếng Trung không bị đụng tới.
func TestLocalizeToolOutputTranslatesErrorsAndResults(t *testing.T) {
	const zh = "第 %d 章的「%s」不存在：%w"
	toolTextVI[zh] = "Chương %d không có “%s”: %w"
	const zhResult = "已保存第 %d 章"
	toolTextVI[zhResult] = "Đã lưu chương %d"
	resetVIPatterns := func() { viPatterns, viPatternsOnce = nil, sync.Once{} }
	resetVIPatterns()
	t.Cleanup(func() {
		delete(toolTextVI, zh)
		delete(toolTextVI, zhResult)
		resetVIPatterns()
	})

	tool := func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"note":"已保存第 3 章"}`), fmt.Errorf(zh, 12, "kho hồ sơ", errs.ErrToolArgs)
	}
	st := store.NewStore(t.TempDir())
	st.SetLanguage("vi")
	out, err := LocalizeToolOutput(st)(context.Background(), agentcore.ToolCall{}, tool)
	if want := "Chương 12 không có “kho hồ sơ”: " + errs.ErrToolArgs.Error(); err == nil || err.Error() != want {
		t.Fatalf("lỗi dịch sai:\n got %v\nwant %s", err, want)
	}
	if !errors.Is(err, errs.ErrToolArgs) {
		t.Fatal("lỗi đã dịch phải giữ errors.Is với lỗi gốc")
	}
	if string(out) != `{"note":"Đã lưu chương 3"}` || !json.Valid(out) {
		t.Fatalf("kết quả dịch sai: %s", out)
	}

	st.SetLanguage("zh")
	out, err = LocalizeToolOutput(st)(context.Background(), agentcore.ToolCall{}, tool)
	if !strings.Contains(err.Error(), "第 12 章") || !strings.Contains(string(out), "已保存第 3 章") {
		t.Fatalf("tác phẩm tiếng Trung phải giữ nguyên: %v %s", err, out)
	}
}

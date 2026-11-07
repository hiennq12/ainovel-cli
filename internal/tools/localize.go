package tools

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/store"
)

// Dịch văn bản tool gửi cho model (mô tả, schema, kết quả, lỗi) sang ngôn ngữ tác phẩm.
//
// Thay vì sửa từng chuỗi ở hàng trăm chỗ (và xung đột mỗi lần merge upstream), văn bản được
// dịch ở ranh giới: Description()/Schema() gọi LocalizeText/localizeSchema, còn kết quả và lỗi
// đi qua LocalizeToolOutput (middleware của agentcore). Lời nhắc StopGuard (agents/guard) cũng
// đi qua LocalizeText. Bảng toolTextVI ánh xạ nguyên văn chuỗi định dạng tiếng Trung trong mã
// nguồn → bản tiếng Việt; tham số (%d, %s, %v, %w, %q…) được bắt theo vị trí và chép sang bản
// dịch, nên chuỗi đã format và chuỗi lỗi lồng nhau (": %w") vẫn dịch được.
// TestToolTextVICoversSource quét mã nguồn để không sót chuỗi nào.
//
// Bản dịch dùng cú pháp động từ của fmt: các động từ lấy tham số lần lượt như bản gốc,
// hoặc chỉ định vị trí bằng %[n]d để đổi trật tự. Không thêm dấu " hay \ so với bản gốc
// (dùng “ ” thay thế) vì văn bản có thể nằm trong chuỗi JSON của kết quả tool.

// LocalizeText dịch văn bản theo ngôn ngữ tác phẩm của store; ngôn ngữ khác "vi" giữ nguyên.
func LocalizeText(st *store.Store, text string) string {
	if st == nil || st.Language() != "vi" {
		return text
	}
	return translateVI(text)
}

// localizeSchema trả về bản sao schema với mọi trường "description" đã dịch.
func localizeSchema(st *store.Store, s map[string]any) map[string]any {
	if st == nil || st.Language() != "vi" {
		return s
	}
	out, _ := translateSchemaVI(s).(map[string]any)
	return out
}

func translateSchemaVI(v any) any {
	switch node := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(node))
		for k, val := range node {
			if text, ok := val.(string); ok && (k == "description" || k == "title") {
				out[k] = translateVI(text)
				continue
			}
			out[k] = translateSchemaVI(val)
		}
		return out
	case []any:
		out := make([]any, len(node))
		for i, val := range node {
			out[i] = translateSchemaVI(val)
		}
		return out
	default:
		return v
	}
}

// LocalizeToolOutput là middleware dịch kết quả và lỗi của tool. Lỗi được bọc lại nhưng vẫn
// Unwrap về lỗi gốc, nên errors.Is/As (errs.ErrToolArgs…) không đổi.
func LocalizeToolOutput(st *store.Store) agentcore.ToolMiddleware {
	return func(ctx context.Context, call agentcore.ToolCall, next agentcore.ToolExecuteFunc) (json.RawMessage, error) {
		out, err := next(ctx, call.Args)
		if st == nil || st.Language() != "vi" {
			return out, err
		}
		if err != nil {
			if msg := translateVI(err.Error()); msg != err.Error() {
				err = &localizedError{msg: msg, err: err}
			}
		}
		if len(out) > 0 && hanRe.Match(out) {
			out = json.RawMessage(translateVI(string(out)))
		}
		return out, err
	}
}

type localizedError struct {
	msg string
	err error
}

func (e *localizedError) Error() string { return e.msg }
func (e *localizedError) Unwrap() error { return e.err }

var hanRe = regexp.MustCompile(`\p{Han}`)

// ── bộ dịch theo mẫu định dạng ──

type viPattern struct {
	re    *regexp.Regexp
	probe string      // đoạn chữ cố định dài nhất của bản gốc: lọc nhanh trước khi chạy regex
	parts []viSegment // bản dịch đã tách: chữ cố định và chỗ chèn tham số
}

type viSegment struct {
	text string
	arg  int // -1 = chữ cố định; ≥0 = chỉ số tham số bắt được
}

var (
	viPatternsOnce sync.Once
	viPatterns     []viPattern
)

// fmtVerbRe nhận các động từ của fmt: %d %s %v %w %q %x %.2f %+d %[2]d …; %% là chữ "%".
var fmtVerbRe = regexp.MustCompile(`%(?:\[(\d+)\])?[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)

func compileVIPatterns() {
	keys := make([]string, 0, len(toolTextVI))
	for zh := range toolTextVI {
		keys = append(keys, zh)
	}
	// Mẫu dài trước: sau khi thay, chữ Hán đã biến mất nên mẫu ngắn không đè lên phần đã dịch.
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	for _, zh := range keys {
		viPatterns = append(viPatterns, compileVIPattern(zh, toolTextVI[zh]))
	}
}

func compileVIPattern(zh, vi string) viPattern {
	var re strings.Builder
	var probe string
	last := 0
	for _, loc := range fmtVerbRe.FindAllStringIndex(zh, -1) {
		lit := zh[last:loc[0]]
		re.WriteString(regexp.QuoteMeta(lit))
		if len(lit) > len(probe) {
			probe = lit
		}
		verb := zh[loc[0]:loc[1]]
		switch verb[len(verb)-1] {
		case '%':
			re.WriteString("%")
		case 'd':
			re.WriteString(`(-?\d+)`)
		case 'f':
			re.WriteString(`(-?[\d.]+)`)
		default:
			re.WriteString(`(.*?)`)
		}
		last = loc[1]
	}
	tail := zh[last:]
	re.WriteString(regexp.QuoteMeta(tail))
	if len(tail) > len(probe) {
		probe = tail
	}

	var parts []viSegment
	last, next := 0, 0
	for _, m := range fmtVerbRe.FindAllStringSubmatchIndex(vi, -1) {
		parts = append(parts, viSegment{text: vi[last:m[0]], arg: -1})
		if vi[m[1]-1] == '%' {
			parts = append(parts, viSegment{text: "%", arg: -1})
		} else {
			idx := next
			if m[2] >= 0 {
				n, _ := strconv.Atoi(vi[m[2]:m[3]])
				idx = n - 1
			}
			parts = append(parts, viSegment{arg: idx})
			next = idx + 1
		}
		last = m[1]
	}
	parts = append(parts, viSegment{text: vi[last:], arg: -1})
	return viPattern{re: regexp.MustCompile(re.String()), probe: probe, parts: parts}
}

func translateVI(text string) string {
	if !hanRe.MatchString(text) {
		return text
	}
	viPatternsOnce.Do(compileVIPatterns)
	for _, p := range viPatterns {
		if !strings.Contains(text, p.probe) {
			continue
		}
		text = p.re.ReplaceAllStringFunc(text, func(match string) string {
			args := p.re.FindStringSubmatch(match)[1:]
			var b strings.Builder
			for _, seg := range p.parts {
				if seg.arg < 0 {
					b.WriteString(seg.text)
				} else if seg.arg < len(args) {
					b.WriteString(args[seg.arg])
				}
			}
			return b.String()
		})
		if !hanRe.MatchString(text) {
			break
		}
	}
	return text
}

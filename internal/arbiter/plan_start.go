package arbiter

import (
	"context"
	"fmt"
	"strings"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
)

// PlanStartDecision 启动裁定:选规划师,必要时给出补充方向。
//
// 裁定不复述用户需求:交给规划师的任务由 PlannerTask 确定性拼装(需求原文 + 补充 +
// 固定收尾)。数万字的设定集若要求模型转述,既会撞 decideMaxTokens 输出上限,
// 也会在转述中丢失细节。
type PlanStartDecision struct {
	Planner    string `json:"planner"`    // architect_long | architect_short
	Supplement string `json:"supplement"` // 仅需求过短时的补充方向;否则为空串
	Reason     string `json:"reason"`
}

func (d *PlanStartDecision) Validate() error {
	if d.Planner != "architect_long" && d.Planner != "architect_short" {
		return fmt.Errorf("planner 非法: %q（可选 architect_long / architect_short）", d.Planner)
	}
	if strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("reason 不能为空")
	}
	return nil
}

// planStartContract 紧邻 PlanStartDecision:字段全 required,planner 是封闭枚举;
// supplement 允许空串(strict 模式下可选语义用空串表达)。
var planStartContract = llmcontract.Contract{
	Name:        "arbiter_plan_start",
	Description: "启动裁定:选规划师,必要时补充创作方向",
	Schema: schema.Object(
		schema.Property("planner", schema.Enum("规划师", "architect_long", "architect_short")).Required(),
		schema.Property("supplement", schema.String("需求过短时补充的创作方向;否则为空字符串。禁止复述用户需求")).Required(),
		schema.Property("reason", schema.String("选择理由")).Required(),
	),
}

// planStartPayload 是 plan_start 的用户负载(事实即输入,无 store 状态——新书)。
type planStartPayload struct {
	Requirement string `json:"requirement"`
	Style       string `json:"style,omitempty"`
}

// DecidePlanStart 启动裁定:根据用户需求选规划师;需求过短(<20 字)时在 supplement 里
// 自主补充差异化方向、目标读者与核心消费点、至少一个非常规钩子。
// 失败语义:返回 error → 调用方显式报错中止启动(启动期用户在场,报错优于猜测)。
func DecidePlanStart(ctx context.Context, model agentcore.ChatModel, systemPrompt, requirement, style string) (PlanStartDecision, error) {
	payload, err := marshalPayload(planStartPayload{Requirement: requirement, Style: style})
	if err != nil {
		return PlanStartDecision{}, err
	}
	return decide(ctx, model, planStartContract, systemPrompt, payload, (*PlanStartDecision).Validate)
}

// planStartLabels 是 PlannerTask 的固定文案,跟随作品语种。
type planStartLabels struct {
	requirement string
	supplement  string
	closing     string
}

var planStartLabelsByLanguage = map[string]planStartLabels{
	"vi": {
		requirement: "## Yêu cầu sáng tác của người dùng (nguyên văn)",
		supplement:  "## Định hướng bổ sung (hệ thống đề xuất; yêu cầu rõ ràng của người dùng luôn được ưu tiên)",
		closing:     "Dùng save_foundation lưu từng mục tiền đề/đại cương/nhân vật/quy tắc thế giới xuống đĩa, sau khi đầy đủ thì gọi lại novel_context và dùng audit_foundation để thẩm định tính nhất quán ngữ nghĩa liên tệp; chỉ kết thúc sau khi audit_foundation trả về foundation_ready=true (không gọi complete_book — đó là thông báo hoàn thành toàn sách sau khi viết xong tất cả các chương).",
	},
	"zh": {
		requirement: "## 用户创作需求（原文）",
		supplement:  "## 补充方向（系统建议；用户显式要求永远优先）",
		closing:     "用 save_foundation 逐项落盘前提/大纲/角色/世界规则，全部齐全后重新调用 novel_context 并用 audit_foundation 审查跨文件语义一致性；仅 audit_foundation 返回 foundation_ready=true 后结束（不要调用 complete_book——那是全书章节写完后的完结宣告）。",
	},
}

// PlannerTask 确定性拼装交给规划师的任务:用户需求原文 + 补充方向(若有)+ 固定收尾指令。
// language 未知时按 "vi" 处理(与 bootstrap.Config.NormalizedLanguage 的默认一致)。
func (d PlanStartDecision) PlannerTask(requirement, language string) string {
	labels, ok := planStartLabelsByLanguage[language]
	if !ok {
		labels = planStartLabelsByLanguage["vi"]
	}
	var b strings.Builder
	b.WriteString(labels.requirement)
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(requirement))
	if supplement := strings.TrimSpace(d.Supplement); supplement != "" {
		b.WriteString("\n\n---\n\n")
		b.WriteString(labels.supplement)
		b.WriteString("\n\n")
		b.WriteString(supplement)
	}
	b.WriteString("\n\n---\n\n")
	b.WriteString(labels.closing)
	return b.String()
}

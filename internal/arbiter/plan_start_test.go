package arbiter

import (
	"strings"
	"testing"
)

func TestPlannerTaskKeepsRequirementVerbatim(t *testing.T) {
	requirement := "# Brief\n\n<!-- Mốc thời gian: 14/11/2005 -->\n" + strings.Repeat("Twist 3 lật ở ch26–28. ", 3000)
	d := PlanStartDecision{Planner: "architect_long", Reason: "长篇"}

	task := d.PlannerTask(requirement, "vi")
	if !strings.Contains(task, strings.TrimSpace(requirement)) {
		t.Fatal("规划师任务必须包含需求原文,不得截断或改写")
	}
	if strings.Contains(task, "Định hướng bổ sung") {
		t.Fatal("supplement 为空时不应输出补充段")
	}
	if !strings.HasSuffix(task, planStartLabelsByLanguage["vi"].closing) {
		t.Fatal("任务结尾必须是固定收尾指令")
	}
}

func TestPlannerTaskAppendsSupplementAndFollowsLanguage(t *testing.T) {
	d := PlanStartDecision{Planner: "architect_long", Supplement: "差异化:凡人视角", Reason: "需求过短"}

	zh := d.PlannerTask("凡人修仙", "zh")
	for _, want := range []string{"凡人修仙", "差异化:凡人视角", planStartLabelsByLanguage["zh"].supplement, planStartLabelsByLanguage["zh"].closing} {
		if !strings.Contains(zh, want) {
			t.Fatalf("zh 任务缺少 %q:\n%s", want, zh)
		}
	}
	if strings.Index(zh, "凡人修仙") > strings.Index(zh, "差异化:凡人视角") {
		t.Fatal("需求原文应在补充方向之前")
	}

	if got := d.PlannerTask("x", "unknown"); !strings.Contains(got, planStartLabelsByLanguage["vi"].closing) {
		t.Fatal("未知语种应回落到 vi 文案")
	}
}

func TestPlanStartValidateAllowsEmptySupplement(t *testing.T) {
	d := PlanStartDecision{Planner: "architect_long", Supplement: "", Reason: "默认长篇"}
	if err := d.Validate(); err != nil {
		t.Fatalf("supplement 为空应合法: %v", err)
	}
}

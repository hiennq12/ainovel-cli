package tools

import "github.com/voocel/ainovel-cli/internal/domain"

// contextText là toàn bộ văn bản cố định mà novel_context đưa vào payload gửi model:
// _loading_summary, _warnings, _usage, related_chapters, selected_memory, finale.
// Payload nằm thẳng trong ngữ cảnh writer/editor/architect, nên văn bản phải theo ngôn
// ngữ tác phẩm: nhãn tiếng Trung trong tác phẩm tiếng Việt kéo model lệch ngôn ngữ
// (cùng lý do với store/labels.go).
type contextText struct {
	readFailed    string // "<scope> <readFailed>: <err>" trong _warnings và lỗi trả về
	episodicUsage string

	// _loading_summary
	charSnapshotsFmt string
	charsFmt         string
	workingFmt       string
	episodicFmt      string
	planningFmt      string
	foundationFmt    string
	volumeSumFmt     string
	arcSumFmt        string
	chapterSumFmt    string
	layeredFmt       string
	timelineFmt      string
	foreshadowFmt    string
	relationsFmt     string
	stateChangesFmt  string
	previousTailOK   string
	styleRulesOK     string
	relatedFmt       string
	threadRecallFmt  string
	reviewRecallFmt  string
	referencesFmt    string
	referencePackFmt string
	memoryPolicyOK   string
	simulationOK     string
	warningsFmt      string
	trimmedFmt       string

	// related_chapters[].reason
	relForeshadowFmt  string // id, mô tả
	relLastSeenFmt    string // nhân vật
	relStateChangeFmt string // nhân vật
	relRelationFmt    string // nhân vật A, nhân vật B

	// selected_memory.story_threads
	threadReason string
	threadFmt    string // id, chương gieo, mô tả
	agingReason  string
	agingFmt     string // id, chương gieo, số chương treo, mô tả

	// selected_memory.review_lessons
	contractReason string
	contractFmt    string // chương, mục hợp đồng còn thiếu
	issueReason    string
	issueFmt       string // chương, mô tả lỗi

	finale string

	// memoryPolicy dịch các trường mô tả của domain.MemoryPolicy (nguyên văn tiếng Trung →
	// bản dịch); nil = giữ nguyên.
	memoryPolicy map[string]string
}

var contextTextZH = contextText{
	readFailed:    "读取失败",
	episodicUsage: "本容器为已写入正文的事实备忘（供一致性与衔接对照）；在新章正文中原样复述这些内容属于重复缺陷",

	charSnapshotsFmt: "角色:%d(快照)",
	charsFmt:         "角色:%d",
	workingFmt:       "工作记忆:%d",
	episodicFmt:      "情节记忆:%d",
	planningFmt:      "规划记忆:%d",
	foundationFmt:    "基础记忆:%d",
	volumeSumFmt:     "卷摘要:%d",
	arcSumFmt:        "弧摘要:%d",
	chapterSumFmt:    "章摘要:%d",
	layeredFmt:       "分层大纲:%d卷",
	timelineFmt:      "时间线:%d",
	foreshadowFmt:    "伏笔:%d",
	relationsFmt:     "关系:%d",
	stateChangesFmt:  "状态变化:%d",
	previousTailOK:   "前章尾部:ok",
	styleRulesOK:     "风格规则:ok",
	relatedFmt:       "相关章:%d",
	threadRecallFmt:  "线索召回:%d",
	reviewRecallFmt:  "评审召回:%d",
	referencesFmt:    "参考:%d项",
	referencePackFmt: "参考包:%d",
	memoryPolicyOK:   "记忆策略:ok",
	simulationOK:     "仿写画像:ok",
	warningsFmt:      "告警:%d",
	trimmedFmt:       "裁剪:%s",

	relForeshadowFmt:  "伏笔%s(%s)埋设章",
	relLastSeenFmt:    "角色'%s'最后出场章",
	relStateChangeFmt: "'%s'状态变化章",
	relRelationFmt:    "%s-%s关系变化",

	threadReason: "当前章可能需要承接既有伏笔",
	threadFmt:    "伏笔“%s”埋于第%d章：%s",
	agingReason:  "伏笔久挂未回收，注意适时推进或回收",
	agingFmt:     "伏笔“%s”埋于第%d章，已 %d 章未回收：%s",

	contractReason: "最近审阅指出 contract 漏项",
	contractFmt:    "第%d章 contract 漏项：%s",
	issueReason:    "最近审阅指出需要避免重复问题",
	issueFmt:       "第%d章审阅提醒：%s",

	finale: "本卷为全书收官卷：不再新开长线或埋新伏笔，优先回收既有伏笔、收拢关系线，按大纲把故事推向终局。",
}

// contextTextVI dùng bộ từ đã thống nhất của dự án: tập / cung / chương / đề cương / điểm móc / phục bút.
var contextTextVI = contextText{
	readFailed:    "đọc thất bại",
	episodicUsage: "Khối này là ghi chú các sự kiện đã viết vào chính văn (để đối chiếu nhất quán và nối mạch); kể lại nguyên văn các nội dung này trong chương mới là lỗi lặp",

	charSnapshotsFmt: "Nhân vật:%d (snapshot)",
	charsFmt:         "Nhân vật:%d",
	workingFmt:       "Bộ nhớ làm việc:%d",
	episodicFmt:      "Bộ nhớ tình tiết:%d",
	planningFmt:      "Bộ nhớ quy hoạch:%d",
	foundationFmt:    "Bộ nhớ nền tảng:%d",
	volumeSumFmt:     "Tóm tắt tập:%d",
	arcSumFmt:        "Tóm tắt cung:%d",
	chapterSumFmt:    "Tóm tắt chương:%d",
	layeredFmt:       "Đề cương phân tầng:%d tập",
	timelineFmt:      "Dòng thời gian:%d",
	foreshadowFmt:    "Phục bút:%d",
	relationsFmt:     "Quan hệ:%d",
	stateChangesFmt:  "Thay đổi trạng thái:%d",
	previousTailOK:   "Đuôi chương trước:ok",
	styleRulesOK:     "Quy tắc văn phong:ok",
	relatedFmt:       "Chương liên quan:%d",
	threadRecallFmt:  "Gợi lại tuyến truyện:%d",
	reviewRecallFmt:  "Gợi lại review:%d",
	referencesFmt:    "Tham khảo:%d mục",
	referencePackFmt: "Gói tham khảo:%d",
	memoryPolicyOK:   "Chính sách bộ nhớ:ok",
	simulationOK:     "Hồ sơ mô phỏng văn phong:ok",
	warningsFmt:      "Cảnh báo:%d",
	trimmedFmt:       "Đã cắt bớt:%s",

	relForeshadowFmt:  "Chương gieo phục bút %s (%s)",
	relLastSeenFmt:    "Chương '%s' xuất hiện lần cuối",
	relStateChangeFmt: "Chương '%s' thay đổi trạng thái",
	relRelationFmt:    "Quan hệ %s-%s thay đổi",

	threadReason: "Chương này có thể cần nối tiếp phục bút đã gieo",
	threadFmt:    "Phục bút “%s” gieo ở chương %d: %s",
	agingReason:  "Phục bút treo lâu chưa thu, chú ý đẩy tiếp hoặc thu đúng lúc",
	agingFmt:     "Phục bút “%s” gieo ở chương %d, đã %d chương chưa thu: %s",

	contractReason: "Review gần nhất chỉ ra mục hợp đồng chương còn thiếu",
	contractFmt:    "Chương %d thiếu mục hợp đồng: %s",
	issueReason:    "Review gần nhất nhắc lỗi cần tránh lặp lại",
	issueFmt:       "Nhắc từ review chương %d: %s",

	finale: "Tập này là tập kết thúc toàn truyện: không mở tuyến dài mới hay gieo phục bút mới, ưu tiên thu các phục bút đã có, khép các tuyến quan hệ, theo đề cương đẩy câu chuyện tới kết cục.",

	memoryPolicy: map[string]string{
		"每次按章节加载时刷新":        "Làm mới mỗi lần nạp theo chương",
		"随章节提交、评审和长篇状态变更刷新": "Làm mới khi commit chương, khi review và khi trạng thái truyện dài thay đổi",
		"卷摘要+弧摘要+最近章节摘要":    "Tóm tắt tập + tóm tắt cung + tóm tắt các chương gần nhất",
		"最近章节摘要":            "Tóm tắt các chương gần nhất",
		"卷弧结构、指南针或摘要更新时刷新":  "Làm mới khi cấu trúc tập/cung, la bàn truyện hoặc tóm tắt thay đổi",
		"角色、伏笔、设定变更时刷新":     "Làm mới khi nhân vật, phục bút hoặc thiết lập thay đổi",
		"分层大纲、指南针、卷摘要":      "Đề cương phân tầng, la bàn truyện, tóm tắt tập",
		"角色设定、角色快照、伏笔台账":    "Thiết lập nhân vật, snapshot nhân vật, sổ phục bút",
	},
}

func contextTextFor(lang string) *contextText {
	if lang == "vi" {
		return &contextTextVI
	}
	return &contextTextZH
}

// text trả về bảng văn bản theo ngôn ngữ tác phẩm của store.
func (t *ContextTool) text() *contextText { return contextTextFor(t.store.Language()) }

// localizePolicy dịch các trường mô tả của memory_policy theo ngôn ngữ tác phẩm.
func (c *contextText) localizePolicy(p domain.MemoryPolicy) domain.MemoryPolicy {
	if c.memoryPolicy == nil {
		return p
	}
	for _, field := range []*string{
		&p.SummaryStrategy, &p.WorkingRefresh, &p.EpisodicRefresh,
		&p.PlanningRefresh, &p.FoundationRefresh, &p.PlanningFocus, &p.FoundationFocus,
	} {
		if v, ok := c.memoryPolicy[*field]; ok {
			*field = v
		}
	}
	return p
}

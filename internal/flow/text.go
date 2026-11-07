package flow

// routeText là toàn bộ văn bản Route sinh ra: Task được gửi nguyên văn cho Worker làm lệnh,
// Reason đi vào sự kiện, log và dữ kiện cho Arbiter khi phân xử lỗi/bế tắc. Cả hai đều là
// thứ model đọc, nên phải theo ngôn ngữ tác phẩm: lệnh tiếng Trung đặt cạnh yêu cầu viết
// tiếng Việt kéo model lệch ngôn ngữ (cùng lý do với tools/novel_context_text.go).
//
// Tên tool và tham số (save_review, scope=arc, chapter=N…) giữ nguyên trong mọi ngôn ngữ;
// prompt Worker nhận diện loại nhiệm vụ qua các cụm này (xem assets/prompts/editor.md,
// architect-long.md).
type routeText struct {
	listSep string // nối danh sách mục thiết lập còn thiếu

	foundationMissingFmt string // danh sách mục thiếu
	foundationAudit      string
	foundationReason     string

	rewriteVerb       string
	polishVerb        string
	rewriteTaskFmt    string // động từ, chương
	needsDirectiveFmt string // chương
	rewriteReasonFmt  string // số chương trong hàng chờ

	aggArcReviewFmt       string // tập, cung, chương đầu, chương cuối, chương cuối, chương cuối
	aggArcReviewReason    string
	aggArcSummaryReason   string
	aggVolSummaryReason   string
	aggGlobalReviewFmt    string // chương cuối ×3
	aggGlobalReviewReason string

	feedbackTask      string
	feedbackReasonFmt string // số phản hồi

	arcReviewFmt       string // tập, cung, chương đầu, chương cuối, chương cuối, chương cuối
	arcReviewReason    string
	arcSummaryFmt      string // tập, cung
	arcSummaryReason   string
	volSummaryFmt      string // tập
	volSummaryReason   string
	expandArcFmt       string // tập, cung
	expandArcReason    string
	newVolumeTask      string
	newVolumeReason    string
	globalReviewFmt    string // chương cuối ×2
	globalReviewReason string // số chương đã hoàn thành; rỗng = dùng lý do của domain.ShouldReview

	outlineExhaustedFmt    string // đã hoàn thành, tổng số chương, chương tiếp theo
	outlineExhaustedReason string

	writeTaskFmt string // chương
	writeReason  string
}

var routeTextZH = routeText{
	listSep: "、",

	foundationMissingFmt: "补齐基础设定与作品信息缺项：%s；book 使用 save_book，其余基础设定使用 save_foundation 落盘",
	foundationAudit:      "基础设定已齐全：重新调用 novel_context 读取全部已落盘工件与 foundation_status.fingerprint，审查跨文件语义一致性后调用 audit_foundation；有问题先修正并重新审查",
	foundationReason:     "基础设定缺项未齐，照缺项续派同一规划师",

	rewriteVerb:    "重写",
	polishVerb:     "打磨",
	rewriteTaskFmt: "%s第 %d 章",
	needsDirectiveFmt: "。本章尚无 chapter_contract：请先调 plan_chapter(chapter=%d) 写清重写目标" +
		"（goal 要回答\"改成什么样\"，而不是复述\"哪里错了\"；" +
		"必要时补 payoff_points / continuity_checks / hook_goal），再据此重写正文",
	rewriteReasonFmt: "PendingRewrites 队列剩余 %d 章",

	aggArcReviewFmt:       "审阅第 %d 卷第 %d 弧（第 %d-%d 章）：调用 novel_context(chapter=%d)，save_review 使用 scope=arc、chapter=%d",
	aggArcReviewReason:    "弧级审阅缺失",
	aggArcSummaryReason:   "弧级摘要缺失",
	aggVolSummaryReason:   "卷摘要缺失",
	aggGlobalReviewFmt:    "审阅前 %d 章：调用 novel_context(chapter=%d)，save_review 使用 scope=global、chapter=%d",
	aggGlobalReviewReason: "全局审阅缺失",

	feedbackTask:      "仅处理 novel_context 中的外部修订 writer_feedback：核对已发生剧情与后续计划，需要调整时调用 revise_outline 或相应结构工具，无需调整时调用 resolve_outline_feedback；不得处理 foundation_status 或其它规划，落盘后用一句话结束",
	feedbackReasonFmt: "有 %d 条外部修订影响尚未传播到后续规划",

	arcReviewFmt:     "对第 %d 卷第 %d 弧（第 %d-%d 章）做弧级评审：调用 novel_context(chapter=%d)，save_review 使用 scope=arc、chapter=%d；issues[].chapters 只能落在该区间",
	arcReviewReason:  "弧末评审未完成",
	arcSummaryFmt:    "生成第 %d 卷第 %d 弧摘要、角色快照与写作规则（save_arc_summary）",
	arcSummaryReason: "弧摘要未完成",
	volSummaryFmt:    "生成第 %d 卷卷摘要（save_volume_summary）",
	volSummaryReason: "卷摘要未完成",
	expandArcFmt:     "展开第 %d 卷第 %d 弧（save_foundation type=expand_arc）",
	expandArcReason:  "下一弧骨架待展开",
	newVolumeTask:    "创建下一卷：按完结判定清单评估后调用 save_foundation——故事继续 → type=append_volume；故事接近终点 → type=append_volume 且卷 JSON 顶层带 \"final\": true（收官卷，整卷收线，写完自动完结）；全部完结条件当下已满足 → type=complete_book。三选一均须附 reason 参数写明判定理由",
	newVolumeReason:  "卷末需决定追加新卷、收官卷或结束全书",
	globalReviewFmt:  "对前 %d 章做全局审阅（save_review scope=global, chapter=%d）",

	outlineExhaustedFmt:    "非分层大纲已写完（已完成 %d 章，共 %d 章）：若故事已收束，调用 save_foundation(type=complete_book)；若仍需继续，用 revise_outline 从第 %d 章续接后续计划",
	outlineExhaustedReason: "非分层大纲已耗尽，需决定完结或续接",

	writeTaskFmt: "写第 %d 章",
	writeReason:  "续写下一章",
}

// routeTextVI dùng bộ từ của prompt tiếng Việt: tập / cung / chương / đại cương / phục bút /
// thiết lập nền tảng. "review cấp cung", "tóm tắt tập", "Tạo tập tiếp theo" là cụm mà
// prompt editor/architect-long dùng để nhận diện nhiệm vụ: đổi ở đây thì đổi cả prompt.
var routeTextVI = routeText{
	listSep: ", ",

	foundationMissingFmt: "Bổ sung các mục thiết lập nền tảng và thông tin tác phẩm còn thiếu: %s; book lưu bằng save_book, các thiết lập nền tảng khác lưu bằng save_foundation",
	foundationAudit:      "Thiết lập nền tảng đã đủ: gọi lại novel_context để đọc toàn bộ sản phẩm đã lưu cùng foundation_status.fingerprint, thẩm định tính nhất quán ngữ nghĩa giữa các tệp rồi gọi audit_foundation; có vấn đề thì sửa trước rồi thẩm định lại",
	foundationReason:     "Thiết lập nền tảng còn thiếu mục, tiếp tục giao cho cùng Kiến trúc sư theo danh sách thiếu",

	rewriteVerb:    "Viết lại",
	polishVerb:     "Trau chuốt",
	rewriteTaskFmt: "%s chương %d",
	needsDirectiveFmt: ". Chương này chưa có chapter_contract: hãy gọi plan_chapter(chapter=%d) trước để viết rõ mục tiêu viết lại" +
		" (goal phải trả lời \"sửa thành thế nào\", không phải nhắc lại \"sai ở đâu\";" +
		" khi cần thì bổ sung payoff_points / continuity_checks / hook_goal), rồi dựa vào đó viết lại chính văn",
	rewriteReasonFmt: "Hàng chờ PendingRewrites còn %d chương",

	aggArcReviewFmt:       "Review cấp cung cho tập %d, cung %d (chương %d-%d): gọi novel_context(chapter=%d), save_review dùng scope=arc, chapter=%d",
	aggArcReviewReason:    "Thiếu review cấp cung",
	aggArcSummaryReason:   "Thiếu tóm tắt cung",
	aggVolSummaryReason:   "Thiếu tóm tắt tập",
	aggGlobalReviewFmt:    "Review toàn cục %d chương đầu: gọi novel_context(chapter=%d), save_review dùng scope=global, chapter=%d",
	aggGlobalReviewReason: "Thiếu review toàn cục",

	feedbackTask:      "Chỉ xử lý phản hồi sửa đổi bên ngoài writer_feedback trong novel_context: đối chiếu cốt truyện đã xảy ra với kế hoạch phía sau; cần điều chỉnh thì gọi revise_outline hoặc công cụ cấu trúc tương ứng, không cần điều chỉnh thì gọi resolve_outline_feedback; không được xử lý foundation_status hay quy hoạch khác, lưu xong thì kết thúc bằng một câu",
	feedbackReasonFmt: "Có %d phản hồi sửa đổi bên ngoài chưa được truyền tới quy hoạch phía sau",

	arcReviewFmt:       "Review cấp cung cho tập %d, cung %d (chương %d-%d): gọi novel_context(chapter=%d), save_review dùng scope=arc, chapter=%d; issues[].chapters chỉ được nằm trong khoảng này",
	arcReviewReason:    "Chưa xong review cuối cung",
	arcSummaryFmt:      "Tạo tóm tắt cung, snapshot nhân vật và quy tắc viết cho tập %d, cung %d (save_arc_summary)",
	arcSummaryReason:   "Chưa xong tóm tắt cung",
	volSummaryFmt:      "Tạo tóm tắt tập %d (save_volume_summary)",
	volSummaryReason:   "Chưa xong tóm tắt tập",
	expandArcFmt:       "Mở rộng tập %d, cung %d (save_foundation type=expand_arc)",
	expandArcReason:    "Khung xương cung tiếp theo đang chờ mở rộng",
	newVolumeTask:      "Tạo tập tiếp theo: đánh giá theo danh sách tiêu chí kết thúc rồi gọi save_foundation — truyện còn tiếp → type=append_volume; truyện gần tới hồi kết → type=append_volume và JSON của tập có \"final\": true ở cấp cao nhất (tập kết thúc: khép mọi tuyến trong tập này, viết xong thì tự hoàn thành truyện); mọi điều kiện kết thúc đã thoả ngay lúc này → type=complete_book. Lựa chọn nào cũng phải kèm tham số reason nêu lý do phán định",
	newVolumeReason:    "Cuối tập cần quyết định thêm tập mới, tập kết thúc hay kết thúc toàn truyện",
	globalReviewFmt:    "Review toàn cục %d chương đầu (save_review scope=global, chapter=%d)",
	globalReviewReason: "Đã hoàn thành %d chương, đến lượt review toàn cục",

	outlineExhaustedFmt:    "Đại cương không phân tầng đã viết hết (đã hoàn thành %d chương, tổng %d chương): nếu truyện đã khép lại, gọi save_foundation(type=complete_book); nếu cần viết tiếp, dùng revise_outline nối kế hoạch tiếp theo từ chương %d",
	outlineExhaustedReason: "Đại cương không phân tầng đã hết, cần quyết định kết thúc hay viết tiếp",

	writeTaskFmt: "Viết chương %d",
	writeReason:  "Viết tiếp chương sau",
}

func routeTextFor(lang string) *routeText {
	if lang == "vi" {
		return &routeTextVI
	}
	return &routeTextZH
}

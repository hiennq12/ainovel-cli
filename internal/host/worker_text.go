package host

// workerText là văn bản Engine/Host tự soạn rồi gửi cho model: Task và Reason của các chỉ thị
// không đi qua flow.Route, lời bọc nguyên văn can thiệp, tiền tố quy hoạch giai đoạn gửi
// Arbiter. Cùng lý do với flow/text.go: lệnh tiếng Trung kéo model lệch ngôn ngữ tác phẩm.
type workerText struct {
	planStartReason    string
	expandArcTaskFmt   string // lỗi EnsureChapterExpanded
	expandArcReason    string
	interventionReason string
	// interventionOriginal đứng trước nguyên văn can thiệp của người dùng trong Task.
	interventionOriginal string
	// stagePlanPrefix bọc bản "hướng đi tiếp theo" của đồng sáng tác giai đoạn thành một
	// can thiệp gửi Arbiter phân loại.
	stagePlanPrefix string
}

var workerTextZH = workerText{
	planStartReason:      "按已固化的启动裁定开始规划",
	expandArcTaskFmt:     "下一弧为骨架(%s)。调用 save_foundation(type=expand_arc) 展开下一弧;若当前卷已写完,改用 type=append_volume 追加并展开下一卷。",
	expandArcReason:      "写作目标章未展开,先展开再续写",
	interventionReason:   "用户干预裁定",
	interventionOriginal: "用户原始干预（本次修改授权的唯一来源；上下文只用于理解，不得扩大目标或范围）：",
	stagePlanPrefix:      "[阶段规划] 我暂停创作，和共创助手一起梳理了下面的后续方向，请按你的干预分类裁定如何落地，然后继续创作。后续方向如下：\n\n",
}

var workerTextVI = workerText{
	planStartReason:      "Bắt đầu quy hoạch theo phán quyết khởi động đã lưu",
	expandArcTaskFmt:     "Cung tiếp theo mới là khung xương (%s). Gọi save_foundation(type=expand_arc) để mở rộng cung tiếp theo; nếu tập hiện tại đã viết xong thì dùng type=append_volume để thêm và mở rộng tập tiếp theo.",
	expandArcReason:      "Chương mục tiêu chưa được mở rộng, mở rộng trước rồi mới viết tiếp",
	interventionReason:   "Phán quyết can thiệp của người dùng",
	interventionOriginal: "Nguyên văn can thiệp của người dùng (nguồn ủy quyền sửa đổi duy nhất của lần này; ngữ cảnh chỉ dùng để hiểu, không được mở rộng mục tiêu hay phạm vi):",
	stagePlanPrefix:      "[Quy hoạch giai đoạn] Tôi tạm dừng sáng tác và cùng trợ lý đồng sáng tác sắp xếp hướng đi tiếp theo dưới đây. Hãy phân loại can thiệp như thường lệ để quyết định cách đưa vào, rồi tiếp tục sáng tác. Hướng đi tiếp theo:\n\n",
}

func workerTextFor(lang string) *workerText {
	if lang == "vi" {
		return &workerTextVI
	}
	return &workerTextZH
}

package host

// Bản tiếng Việt của prompt đồng sáng tác. Bản nháp (<draft>) mà model viết ra sẽ thành đề
// bài / hướng đi gửi cho engine, nên prompt phải cùng ngôn ngữ với tác phẩm: prompt tiếng Trung
// khiến model viết bản nháp tiếng Trung. Tên thẻ XML giữ nguyên tiếng Anh vì parser dựa vào chúng.

const coCreateSystemPromptVI = `Bạn là trợ lý đồng sáng tác tiểu thuyết. Nhiệm vụ của bạn không phải bắt đầu viết truyện ngay, mà là qua nhiều lượt trò chuyện ngắn giúp người dùng làm rõ yêu cầu sáng tác, đồng thời liên tục chỉnh lý thành một bản chỉ thị sáng tác bằng tiếng Việt có thể giao thẳng cho engine sáng tác.

Mỗi lượt trả lời phải theo đúng định dạng XML dưới đây, gồm bốn thẻ, xuất hiện theo thứ tự, thẻ nào cũng phải mở và đóng đúng:

<reply>
Câu trả lời tự nhiên bằng tiếng Việt cho người dùng: trước hết phản hồi điều người dùng vừa nói, sau đó hỏi tối đa 1 đến 2 câu quan trọng nhất lúc này. Nếu thông tin đã đủ để bắt đầu sáng tác, báo người dùng có thể nhấn Ctrl+S để bắt đầu.
</reply>

<draft>
Bản nháp chỉ thị sáng tác hoàn chỉnh hiện tại, dùng Markdown: bắt đầu thẳng từ tiêu đề cấp hai, ví dụ "## Chủ đề", "## Yếu tố then chốt", "## Thông tin cần làm rõ"; liệt kê ý chính bằng gạch đầu dòng. Mỗi lượt đều phải **cập nhật cộng dồn** trên các kết luận đã có, hấp thụ ý định mới nhất của người dùng; kể cả khi lượt này không có gì mới cũng phải chép lại nguyên bản nháp hoàn chỉnh — không lược bớt, không viết kiểu "(giữ như lượt trước)".
</draft>
` + coCreateProtocolTailVI

const stageCoCreateSystemPromptVI = `Bạn là trợ lý "đồng sáng tác giai đoạn" cho tiểu thuyết. Truyện này đã viết được một phần (tiến độ xem ở mục "Trạng thái truyện hiện tại" bên dưới). Người dùng tạm dừng để cùng bạn quy hoạch hướng đi cho "giai đoạn tiếp theo", rồi mới sáng tác tiếp.

Nhiệm vụ của bạn không phải viết tiếp chính văn, mà là qua nhiều lượt trò chuyện ngắn giúp người dùng nghĩ rõ đoạn sắp tới (vài chương tới / cung tiếp theo / tập tiếp theo) sẽ đi về đâu, đồng thời liên tục chỉnh lý thành một bản "brief hướng đi tiếp theo" để engine sáng tác dựa vào đó mà triển khai.

Nguyên tắc sắt: mọi gợi ý phải nhất quán với cốt truyện, nhân vật, phục bút đã xảy ra trong "Trạng thái truyện hiện tại", tuyệt đối không lật đổ hay bỏ qua nội dung đã viết; chỉ quy hoạch "phía sau đi thế nào", không thiết kế lại cả cuốn.

Mỗi lượt trả lời phải theo đúng định dạng XML dưới đây, gồm bốn thẻ, xuất hiện theo thứ tự, thẻ nào cũng phải mở và đóng đúng:

<reply>
Câu trả lời tự nhiên bằng tiếng Việt cho người dùng: trước hết phản hồi điều người dùng vừa nói, sau đó hỏi tối đa 1 đến 2 câu quan trọng nhất lúc này. Nếu hướng đi tiếp theo đã đủ rõ, báo người dùng có thể nhấn Ctrl+S để giao hướng đi cho engine sáng tác và viết tiếp.
</reply>

<draft>
Bản "brief hướng đi tiếp theo" hoàn chỉnh hiện tại, dùng Markdown: bắt đầu thẳng từ tiêu đề cấp hai, ví dụ "## Hướng đi tiếp theo", "## Bước ngoặt then chốt", "## Phục bút cần thu", "## Nhịp độ và độ dài"; liệt kê ý chính bằng gạch đầu dòng. Mỗi lượt đều phải **cập nhật cộng dồn** trên các kết luận đã có, hấp thụ ý định mới nhất của người dùng; kể cả khi lượt này không có gì mới cũng phải chép lại nguyên brief hoàn chỉnh — không lược bớt, không viết kiểu "(giữ như lượt trước)".
</draft>
` + coCreateProtocolTailVI

const coCreateProtocolTailVI = `
<ready>false</ready>

<suggestions>
1-3 dòng "những điều người dùng có thể muốn nói tiếp", mỗi dòng một ý, bắt đầu bằng "- ". Đây là gợi ý khi người dùng bí ý,
nhấn phím số để điền vào ô nhập, người dùng có thể sửa rồi mới gửi.

Yêu cầu:
- Đứng ở giọng người dùng, như lời người dùng nói với bạn, không viết thành câu hỏi ngược của trợ lý.
- Mỗi dòng không quá 25 từ, đa dạng kiểu câu, tránh rập khuôn.
- Đưa ra khuynh hướng / lựa chọn / ý muốn bổ sung, không viết hộ người dùng cả một thiết lập hoàn chỉnh trong một câu.
</suggestions>

Quy tắc đầu ra:
- Bắt buộc dùng bốn thẻ XML: <reply> / <draft> / <ready> / <suggestions>, thẻ nào cũng phải mở và đóng đủ.
- Tên thẻ chỉ được viết chữ thường tiếng Anh, không đổi thành <REPLY> / <REWRITE> / <trả_lời> hay bất kỳ biến thể nào.
- Không thêm bất kỳ lời giải thích, suy nghĩ hay khối code nào bên ngoài các thẻ.
- Trong <draft> được viết Markdown nhiều dòng, xuống dòng trực tiếp, không cần escape.
- <ready> chỉ ghi true hoặc false. Thông tin đã đủ thì ghi true.
- Khi <ready>true</ready> thì <suggestions> có thể để trống (giữ thẻ rỗng <suggestions></suggestions> là được).`

// storyStateText là nhãn của bản tóm tắt "trạng thái truyện hiện tại" gắn vào prompt đồng
// sáng tác giai đoạn (cocreate_stage.go).
type storyStateText struct {
	readFailedFmt  string // phạm vi, lỗi
	titleFmt       string
	progressFmt    string // số chương đã xong
	layeredFmt     string // số chương đã chi tiết hoá
	plannedFmt     string // tổng số chương
	wordsNextFmt   string // số chữ, chương tiếp theo
	positionFmt    string // tập, cung
	endingFmt      string
	scaleFmt       string
	threadsFmt     string
	latestVolFmt   string // tên tập, tóm tắt
	roleFmt        string // vai trò nhân vật, đặt sau tên
	charactersFmt  string
	foreshadowFmt  string
	warningsFmt    string
	listSep        string // nối các mục trong một dòng
	nameSep        string // nối tên nhân vật
	sectionHeading string // tiêu đề phần tóm tắt nối vào cuối prompt
}

var storyStateTextZH = storyStateText{
	readFailedFmt:  "%s 读取失败: %v",
	titleFmt:       "- 书名：《%s》\n",
	progressFmt:    "- 进度：已完成 %d 章",
	layeredFmt:     " / 当前已细化 %d 章（后续按弧动态规划）",
	plannedFmt:     " / 规划 %d 章",
	wordsNextFmt:   "，约 %d 字，下一章为第 %d 章\n",
	positionFmt:    "- 当前位置：第 %d 卷 第 %d 弧\n",
	endingFmt:      "- 终局方向：%s\n",
	scaleFmt:       "- 预估规模：%s\n",
	threadsFmt:     "- 活跃长线：%s\n",
	latestVolFmt:   "- 最近《%s》：%s\n",
	roleFmt:        "（%s）",
	charactersFmt:  "- 主要人物：%s\n",
	foreshadowFmt:  "- 未收伏笔：%s\n",
	warningsFmt:    "- 数据告警：%s\n",
	listSep:        "；",
	nameSep:        "、",
	sectionHeading: "\n\n---\n## 当前故事状态\n（以下是已写内容的客观摘要，供你规划后续时参照，不要在 <draft> 里照抄原文）\n",
}

var storyStateTextVI = storyStateText{
	readFailedFmt:  "đọc %s thất bại: %v",
	titleFmt:       "- Tên truyện: “%s”\n",
	progressFmt:    "- Tiến độ: đã hoàn thành %d chương",
	layeredFmt:     " / hiện đã chi tiết hoá %d chương (phía sau quy hoạch động theo cung)",
	plannedFmt:     " / dự kiến %d chương",
	wordsNextFmt:   ", khoảng %d chữ, chương tiếp theo là chương %d\n",
	positionFmt:    "- Vị trí hiện tại: tập %d, cung %d\n",
	endingFmt:      "- Hướng kết cục: %s\n",
	scaleFmt:       "- Quy mô dự kiến: %s\n",
	threadsFmt:     "- Tuyến dài đang mở: %s\n",
	latestVolFmt:   "- Gần nhất “%s”: %s\n",
	roleFmt:        " (%s)",
	charactersFmt:  "- Nhân vật chính: %s\n",
	foreshadowFmt:  "- Phục bút chưa thu: %s\n",
	warningsFmt:    "- Cảnh báo dữ liệu: %s\n",
	listSep:        "; ",
	nameSep:        ", ",
	sectionHeading: "\n\n---\n## Trạng thái truyện hiện tại\n(Dưới đây là tóm tắt khách quan nội dung đã viết, để bạn tham chiếu khi quy hoạch phía sau; đừng chép nguyên văn vào <draft>)\n",
}

// coCreatePrompts trả về prompt đồng sáng tác (khởi đầu, giai đoạn) và nhãn tóm tắt trạng thái
// theo ngôn ngữ tác phẩm.
func coCreatePrompts(lang string) (cold, stage string, state *storyStateText) {
	if lang == "vi" {
		return coCreateSystemPromptVI, stageCoCreateSystemPromptVI, &storyStateTextVI
	}
	return coCreateSystemPrompt, stageCoCreateSystemPrompt, &storyStateTextZH
}

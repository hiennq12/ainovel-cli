package domain

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ReviewInterval 全局审阅间隔（每 N 章触发一次）。
const ReviewInterval = 5

// ShouldReview 根据已完成章节数判断是否需要全局审阅（短篇/中篇模式）。
func ShouldReview(completedCount int) (bool, string) {
	if completedCount > 0 && completedCount%ReviewInterval == 0 {
		return true, fmt.Sprintf("已完成 %d 章，触发全局审阅", completedCount)
	}
	return false, ""
}

// ShouldArcReview 长篇模式下判断是否需要弧级/卷级评审。
func ShouldArcReview(isArcEnd, isVolumeEnd bool, volume, arc int) (bool, string) {
	if isVolumeEnd {
		return true, fmt.Sprintf("第 %d 卷第 %d 弧结束（卷结束），触发弧级+卷级评审", volume, arc)
	}
	if isArcEnd {
		return true, fmt.Sprintf("第 %d 卷第 %d 弧结束，触发弧级评审", volume, arc)
	}
	return false, ""
}

// WordCount 计算章节字数。
// 以中日韩文字为主的正文沿用按 rune 计数(与历史数据一致);
// 越南语、英语等以空格分词的正文按词计数(越南语一个"tiếng"即一个词),
// 否则 3000 字的越南语章节会被算成一万多字,误导 writer 提前收笔。
func WordCount(content string) int {
	cjk, words := 0, 0
	for _, field := range strings.Fields(content) {
		hasWordRune := false
		for _, r := range field {
			switch {
			case isCJKRune(r):
				cjk++
			case unicode.IsLetter(r) || unicode.IsDigit(r):
				hasWordRune = true
			}
		}
		if hasWordRune {
			words++
		}
	}
	if cjk >= words {
		return utf8.RuneCountInString(content)
	}
	return cjk + words
}

func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

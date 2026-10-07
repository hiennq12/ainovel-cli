package domain

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWordCountChineseKeepsRuneCount(t *testing.T) {
	text := "# 第一章 夜雨\n\n林默推开门，雨声灌进来。他看了一眼 iPhone，没有说话。"
	if got, want := WordCount(text), utf8.RuneCountInString(text); got != want {
		t.Fatalf("中文正文应沿用 rune 计数: got %d want %d", got, want)
	}
}

func TestWordCountVietnameseCountsWords(t *testing.T) {
	text := "# Người chết đặt lịch\n\n\"Ông ấy trả gấp ba,\" cô Huệ nói. \"Hai tuần trước khi chết — ông ấy tự đến đặt.\"\n\n---\n\n412, nhà C3."
	// Người chết đặt lịch(4) + Ông ấy trả gấp ba cô Huệ nói(8) + Hai tuần trước khi chết ông ấy tự đến đặt(10) + 412 nhà C3(3)
	if got := WordCount(text); got != 25 {
		t.Fatalf("越南语应按词计数,标点/分隔线/标题符号不计: got %d want 25", got)
	}
}

func TestWordCountVietnameseChapterScale(t *testing.T) {
	text := strings.Repeat("Tôi xách thùng đồ lên, bước vào bóng cầu thang. ", 300)
	if got := WordCount(text); got != 3000 {
		t.Fatalf("got %d want 3000", got)
	}
}

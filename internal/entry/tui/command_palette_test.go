package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
)

func TestExportCompletionWaitsForArguments(t *testing.T) {
	m := Model{textarea: textarea.New()}
	m.textarea.Focus()
	m.textarea.SetValue("/export")
	m.updateCommandPalette()

	item, ok := m.acceptCommandCompletion()
	if !ok || item.Name != "export" {
		t.Fatalf("accepted %+v, want export", item)
	}
	if item.AutoExecute {
		t.Fatal("/export must not auto-execute; user needs to type path/from/to first")
	}
	if got := m.textarea.Value(); got != "/export " {
		t.Fatalf("input = %q, want %q", got, "/export ")
	}
}

func TestCommandPaletteStaysClosedAfterAcceptingCompletion(t *testing.T) {
	m := Model{textarea: textarea.New()}
	m.textarea.Focus()

	m.textarea.SetValue("/simulate")
	m.updateCommandPalette()
	if !m.compActive {
		t.Fatal("palette should open while typing /simulate")
	}

	if _, ok := m.acceptCommandCompletion(); !ok {
		t.Fatal("accept should succeed")
	}
	// Model.Update gọi lại updateCommandPalette sau mỗi phím; palette không được mở lại,
	// nếu không Enter kế tiếp sẽ chỉ chấp nhận gợi ý mãi mà không chạy lệnh.
	m.updateCommandPalette()
	if m.compActive {
		t.Fatalf("palette reopened after accepting completion; input=%q", m.textarea.Value())
	}
}

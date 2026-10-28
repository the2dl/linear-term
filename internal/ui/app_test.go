package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func sendKey(m tea.Model, k string) tea.Model {
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	next, _ := m.Update(msg)
	return next
}

func TestBoardRenderingAndInteractions(t *testing.T) {
	var m tea.Model = NewAppModel(Config{
		ForceDemo:    true,
		PollInterval: 15 * time.Second,
	})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 168, Height: 44})

	view := m.View()
	for _, expected := range []string{
		"Acme",
		"Delivery",
		"In Progress",
		"In Review",
		"Rolling Out",
		"Done",
		"LIN-4742",
		"LIN-4755",
		"LIN-4749",
		"LIN-4758",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected board view to contain %q", expected)
		}
	}

	// Toggle empty columns ('e') -> should reveal Backlog, Todo, Duplicate
	m = sendKey(m, "e")
	viewAllCols := m.View()
	if !strings.Contains(viewAllCols, "Backlog") || !strings.Contains(viewAllCols, "Todo") {
		t.Fatalf("expected 'e' toggle to reveal Backlog and Todo columns")
	}
	// Toggle back
	m = sendKey(m, "e")

	// Open issue detail modal on LIN-4742
	m = sendKey(m, "enter")
	detailView := m.View()
	if !strings.Contains(detailView, "DESCRIPTION") || !strings.Contains(detailView, "SUB-ISSUES (3/3)") {
		t.Fatalf("expected detail modal to show DESCRIPTION and SUB-ISSUES")
	}
	m = sendKey(m, "esc")

	// Move LIN-4742 to In Review via 's' -> '4'
	m = sendKey(m, "s")
	pickerView := m.View()
	if !strings.Contains(pickerView, "Move LIN-4742 to Workflow State") {
		t.Fatalf("expected state picker modal to open")
	}
	m = sendKey(m, "4") // 4 = In Review
	afterMoveView := m.View()
	if !strings.Contains(afterMoveView, "Moved LIN-4742") {
		t.Fatalf("expected toast confirming LIN-4742 moved to In Review")
	}

	// Filter via '/' for "8269"
	m = sendKey(m, "/")
	for _, r := range "8269" {
		m = sendKey(m, string(r))
	}
	m = sendKey(m, "enter")
	filteredView := m.View()
	if !strings.Contains(filteredView, "LIN-4763") {
		t.Fatalf("expected filter '8269' to match LIN-4763")
	}
	m = sendKey(m, "esc") // Clear filter

	// Simulate live event via 'L'
	m = sendKey(m, "L")
	liveView := m.View()
	if !strings.Contains(liveView, "Live") {
		t.Fatalf("expected 'L' to trigger a simulated live update toast")
	}
}

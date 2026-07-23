package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIInitialStateAndTabSwitching(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	if m.state != stateHistory {
		t.Errorf("expected initial state to be stateHistory (0), got %v", m.state)
	}

	// Press '2' to switch to Search tab (stateSearchInput)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m2 := newModel.(model)
	if m2.state != stateSearchInput {
		t.Errorf("expected state to be stateSearchInput (1), got %v", m2.state)
	}

	// Press Esc to blur search input back to navigation mode
	newModel, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mEsc := newModel.(model)

	// Press '3' to switch to Logs tab (stateLogs)
	newModel, _ = mEsc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m3 := newModel.(model)
	if m3.state != stateLogs {
		t.Errorf("expected state to be stateLogs (9), got %v", m3.state)
	}

	// Press '4' to switch to Config tab (stateConfig)
	newModel, _ = m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m4 := newModel.(model)
	if m4.state != stateConfig {
		t.Errorf("expected state to be stateConfig (10), got %v", m4.state)
	}
}

func TestTUIListViewRendering(t *testing.T) {
	m := initialModel("", "sub", "best", false)
	viewStr := m.View()
	if viewStr == "" {
		t.Errorf("expected initial model View() to return non-empty string")
	}
}

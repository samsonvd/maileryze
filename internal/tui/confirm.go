package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "y", "enter":
		if m.confirm.action != nil {
			cmd := m.confirm.action
			// Mark affected rows as pending before dismissing the modal.
			if m.se.pending == nil {
				m.se.pending = make(map[string]bool)
			}
			for _, addr := range m.confirm.addresses {
				m.se.pending[addr] = true
			}
			m.confirm.active = false
			// Kick the senders spinner if there's anything to wait for.
			var spinCmd tea.Cmd
			if len(m.se.pending) > 0 {
				spinCmd = m.se.sp.Tick
			}
			return m, tea.Batch(cmd, spinCmd)
		}
		m.confirm.active = false

	case "n", "esc":
		m.confirm.active = false
	}

	return m, nil
}

func (m model) viewConfirm() string {
	var content strings.Builder

	content.WriteString(boldStyle.Render(m.confirm.title) + "\n")
	if m.confirm.body != "" {
		content.WriteString("\n")
		content.WriteString(mutedStyle.Render(m.confirm.body) + "\n")
	}
	content.WriteString("\n")
	content.WriteString(renderKeys("y", "confirm", "n", "cancel"))

	dialog := dialogStyle.Render(content.String())

	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := strings.Count(dialog, "\n") + 1

	x := (m.width - dialogWidth) / 2
	y := (m.height - dialogHeight) / 2

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Build full screen with dialog centered
	var screen strings.Builder
	for i := 0; i < y; i++ {
		screen.WriteString("\n")
	}
	for _, line := range strings.Split(dialog, "\n") {
		screen.WriteString(strings.Repeat(" ", x) + line + "\n")
	}

	return screen.String()
}

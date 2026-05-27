package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirm.working {
		// Waiting for action to complete — only allow quit
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "y", "enter":
		if m.confirm.action != nil {
			m.confirm.working = true
			return m, tea.Batch(
				m.confirm.action,
				m.confirm.sp.Tick,
			)
		}
		m.confirm.active = false

	case "n", "esc":
		m.confirm.active = false
	}

	return m, nil
}

func (m model) viewConfirm() string {
	var content strings.Builder

	if m.confirm.working {
		content.WriteString(m.confirm.sp.View() + " Working…\n")
		content.WriteString("\n")
		content.WriteString(mutedStyle.Render("Please wait…"))
	} else {
		content.WriteString(boldStyle.Render(m.confirm.title) + "\n")
		if m.confirm.body != "" {
			content.WriteString("\n")
			content.WriteString(mutedStyle.Render(m.confirm.body) + "\n")
		}
		content.WriteString("\n")
		content.WriteString(renderKeys("y", "confirm", "n", "cancel"))
	}

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

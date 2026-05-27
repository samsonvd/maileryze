package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#A78BFA"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorSuccess = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
	colorDanger  = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	colorWarning = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	mutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	boldStyle   = lipgloss.NewStyle().Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	dangerStyle  = lipgloss.NewStyle().Foreground(colorDanger)
	warningStyle = lipgloss.NewStyle().Foreground(colorWarning)

	selectedRowStyle  = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	visualRangeStyle  = lipgloss.NewStyle().Foreground(colorPrimary)
	decidedRowStyle   = lipgloss.NewStyle().Foreground(colorMuted)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 3)

	statusStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	statusErrStyle = lipgloss.NewStyle().Foreground(colorDanger)

	keyStyle  = lipgloss.NewStyle().Bold(true)
	hintStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

func divider(width int) string {
	if width < 1 {
		return ""
	}
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return mutedStyle.Render(line)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}

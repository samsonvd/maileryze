package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorPrimary = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#A78BFA"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorSuccess = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"}
	colorDanger  = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	colorWarning = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	dangerStyle  = lipgloss.NewStyle().Foreground(colorDanger)
	warningStyle = lipgloss.NewStyle().Foreground(colorWarning)

	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)

	// Staging action row colours
	stagedKeepStyle   = lipgloss.NewStyle().Foreground(colorSuccess)
	stagedUnsubStyle  = lipgloss.NewStyle().Foreground(colorWarning)
	stagedDeleteStyle = lipgloss.NewStyle().Foreground(colorDanger)

	// Rows sharing a domain with a staged sender
	relatedStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#67E8F9"})

	statusStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	statusErrStyle = lipgloss.NewStyle().Foreground(colorDanger)

	keyStyle  = lipgloss.NewStyle().Bold(true)
	hintStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

func divider(width int) string {
	if width < 1 {
		return ""
	}
	line := strings.Repeat("─", width)
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

func adjustScroll(cursor, scroll, visH int) int {
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+visH {
		return cursor - visH + 1
	}
	return scroll
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(ch)
	}
	return result
}

func renderKeys(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, keyStyle.Render("["+pairs[i]+"]")+" "+hintStyle.Render(pairs[i+1]))
	}
	return strings.Join(parts, "  ")
}

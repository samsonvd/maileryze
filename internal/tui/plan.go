package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) visiblePlanLines() int {
	const headerLines = 2 // title + divider
	const footerLines = 3 // divider + key-hints + status-bar
	v := m.height - headerLines - footerLines
	if v < 1 {
		return 1
	}
	return v
}

func (m model) updatePlan(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "tab", "esc":
			m.screen = screenTriage
			return m, nil

		case "up", "k":
			if m.pl.scroll > 0 {
				m.pl.scroll--
			}

		case "down", "j":
			m.pl.scroll++

		case "g":
			m.pl.scroll = 0

		case "G":
			m.pl.scroll = 1<<31 - 1 // clamped in viewPlan

		case "e":
			if !m.pl.executing {
				return m.startExecution()
			}
		}
	}
	return m, nil
}

func (m model) startExecution() (model, tea.Cmd) {
	if m.conn == nil {
		m.setStatus("still connecting — try again in a moment", true)
		return m, nil
	}
	unsubAddrs := m.w.UnsubAddresses()
	deleteAddrs := m.w.DeleteAddresses()
	if len(unsubAddrs) == 0 && len(deleteAddrs) == 0 {
		m.setStatus("nothing to execute (keep entries require no action)", false)
		return m, nil
	}
	m.pl.done = nil
	m.pl.errors = make(map[string]error)
	return m, startExecuteCmd(m.conn, m.senderInfo, unsubAddrs, deleteAddrs)
}

func (m model) viewPlan() string {
	var b strings.Builder

	// ── title ─────────────────────────────────────────────────────────────────
	keep, unsub, del := m.w.Counts()
	total := keep + unsub + del
	title := fmt.Sprintf("Plan — %s  ·  %d decisions", m.alias, total)
	b.WriteString(titleStyle.Render(title) + "\n")
	b.WriteString(divider(m.width) + "\n")

	// ── collect scrollable content lines ─────────────────────────────────────
	var lines []string

	if total == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+mutedStyle.Render("No decisions yet — triage senders in the main screen [tab]."))
	} else {
		keepAddrs := m.w.KeepAddresses()
		unsubAddrs := m.w.UnsubAddresses()
		delAddrs := m.w.DeleteAddresses()

		if len(unsubAddrs) > 0 {
			lines = append(lines, "")
			lines = append(lines, warningStyle.Render(fmt.Sprintf("  Unsubscribe (%d)", len(unsubAddrs))))
			for _, addr := range unsubAddrs {
				lines = append(lines, fmt.Sprintf("    %s", addr))
			}
		}
		if len(delAddrs) > 0 {
			lines = append(lines, "")
			lines = append(lines, dangerStyle.Render(fmt.Sprintf("  Delete + Unsubscribe (%d)", len(delAddrs))))
			for _, addr := range delAddrs {
				lines = append(lines, fmt.Sprintf("    %s", addr))
			}
		}
		if len(keepAddrs) > 0 {
			lines = append(lines, "")
			lines = append(lines, successStyle.Render(fmt.Sprintf("  Keep (%d)", len(keepAddrs))))
			for _, addr := range keepAddrs {
				lines = append(lines, fmt.Sprintf("    %s", addr))
			}
		}

		if m.pl.executing {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  %s executing… %d / %d done",
				m.pl.sp.View(), len(m.pl.done), len(unsubAddrs)+len(delAddrs)))
		} else if len(m.pl.done) > 0 {
			lines = append(lines, "")
			lines = append(lines, successStyle.Render(fmt.Sprintf("  %d actions executed", len(m.pl.done))))
			if len(m.pl.errors) > 0 {
				lines = append(lines, dangerStyle.Render(fmt.Sprintf("  %d errors", len(m.pl.errors))))
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("  Edit: ~/.config/maileryze/%s_plan.toml", m.alias)))
	lines = append(lines, "")

	// ── render visible window ─────────────────────────────────────────────────
	visH := m.visiblePlanLines()
	scroll := m.pl.scroll
	if maxScroll := max(0, len(lines)-visH); scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + visH
	if end > len(lines) {
		end = len(lines)
	}
	for _, line := range lines[scroll:end] {
		b.WriteString(line + "\n")
	}

	// ── footer ────────────────────────────────────────────────────────────────
	b.WriteString(divider(m.width) + "\n")
	if m.pl.executing {
		b.WriteString(renderKeys("tab", "triage", "q", "quit") + "\n")
	} else {
		b.WriteString(renderKeys("↑↓/kj", "scroll", "e", "execute", "tab", "triage", "q", "quit") + "\n")
	}
	b.WriteString(renderStatusBar(m))

	return b.String()
}

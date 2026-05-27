package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updatePlan(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "tab", "esc":
			m.screen = screenTriage
			return m, nil

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

	keep, unsub, del := m.w.Counts()
	total := keep + unsub + del

	title := fmt.Sprintf("Plan — %s  ·  %d decisions", m.alias, total)
	b.WriteString(titleStyle.Render(title) + "\n")
	b.WriteString(divider(m.width) + "\n")

	if total == 0 {
		b.WriteString("\n  " + mutedStyle.Render("No decisions yet — triage senders in the main screen [tab].") + "\n")
	} else {
		keepAddrs := m.w.KeepAddresses()
		unsubAddrs := m.w.UnsubAddresses()
		delAddrs := m.w.DeleteAddresses()

		if len(keepAddrs) > 0 {
			b.WriteString("\n")
			b.WriteString(successStyle.Render(fmt.Sprintf("  Keep (%d)", len(keepAddrs))) + "\n")
			for _, addr := range keepAddrs {
				b.WriteString(fmt.Sprintf("    %s\n", addr))
			}
		}
		if len(unsubAddrs) > 0 {
			b.WriteString("\n")
			b.WriteString(warningStyle.Render(fmt.Sprintf("  Unsubscribe (%d)", len(unsubAddrs))) + "\n")
			for _, addr := range unsubAddrs {
				b.WriteString(fmt.Sprintf("    %s\n", addr))
			}
		}
		if len(delAddrs) > 0 {
			b.WriteString("\n")
			b.WriteString(dangerStyle.Render(fmt.Sprintf("  Delete + Unsubscribe (%d)", len(delAddrs))) + "\n")
			for _, addr := range delAddrs {
				b.WriteString(fmt.Sprintf("    %s\n", addr))
			}
		}

		if m.pl.executing {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  %s executing… %d / %d done\n",
				m.pl.sp.View(), len(m.pl.done), len(unsubAddrs)+len(delAddrs)))
		} else if len(m.pl.done) > 0 {
			b.WriteString("\n")
			b.WriteString(successStyle.Render(fmt.Sprintf("  %d actions executed", len(m.pl.done))) + "\n")
			if len(m.pl.errors) > 0 {
				b.WriteString(dangerStyle.Render(fmt.Sprintf("  %d errors", len(m.pl.errors))) + "\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf(
		"  Edit: ~/.config/maileryze/%s_plan.toml", m.alias)) + "\n")
	b.WriteString("\n")
	b.WriteString(divider(m.width) + "\n")

	if m.pl.executing {
		b.WriteString(renderKeys("tab", "triage", "q", "quit") + "\n")
	} else {
		b.WriteString(renderKeys("e", "execute", "tab", "triage", "q", "quit") + "\n")
	}
	b.WriteString(renderStatusBar(m))

	return b.String()
}

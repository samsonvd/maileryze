package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) updateOverview(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.ov.cursor > 0 {
				m.ov.cursor--
			}
		case "down", "j":
			if m.ov.cursor < len(m.ov.rows)-1 {
				m.ov.cursor++
			}
		case "enter":
			if len(m.ov.rows) > 0 {
				return m.enterSenders(m.ov.rows[m.ov.cursor].alias)
			}
		case "s":
			if len(m.ov.rows) > 0 {
				return m.enterSync(m.ov.rows[m.ov.cursor].alias)
			}
		}
	}
	return m, nil
}

func (m model) enterSenders(alias string) (model, tea.Cmd) {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(colorMuted)
	delegate.Styles.NormalTitle = lipgloss.NewStyle()
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(colorMuted)

	l := list.New(nil, delegate, m.width, senderListHeight(m.height))
	l.Title = "Loading senders..."
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	m.se = sendersState{
		alias:   alias,
		list:    l,
		loading: true,
	}
	m.screen = screenSenders

	var cmds []tea.Cmd
	cmds = append(cmds, loadSendersCmd(m.database, alias))

	if _, ok := m.connectors[alias]; !ok {
		m.setStatus("Connecting to Gmail — a browser window will open…", false)
		cmds = append(cmds, loadConnectorCmd(alias))
	}

	return m, tea.Batch(cmds...)
}

func (m model) enterSync(alias string) (model, tea.Cmd) {
	// If sync is already running for this alias, just navigate back to watch it.
	if m.sy.running && m.sy.alias == alias {
		m.screen = screenSync
		return m, nil
	}
	m.sy.alias = alias
	m.sy.preset = preset90Days
	m.sy.running = false
	m.sy.done = false
	m.sy.loaded = 0
	m.sy.inserted = 0
	m.sy.err = nil
	m.sy.ch = nil
	m.screen = screenSync
	return m, nil
}

func (m model) viewOverview() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("maileryze") + "\n")
	b.WriteString(divider(m.width) + "\n")
	b.WriteString("\n")

	if m.ov.loading {
		b.WriteString("  Loading…\n")
	} else if len(m.ov.rows) == 0 {
		b.WriteString("  No providers configured.\n")
		b.WriteString(mutedStyle.Render("  Add entries to ~/.config/maileryze/maileryze.toml and run maileryze again.") + "\n")
	} else {
		header := fmt.Sprintf("  %-30s  %10s  %8s  %8s  %s",
			"Alias", "Emails", "Senders", "Triaged", "Last Sync")
		b.WriteString(mutedStyle.Render(header) + "\n")
		b.WriteString(divider(m.width) + "\n")

		for i, row := range m.ov.rows {
			cursor := "  "
			if i == m.ov.cursor {
				cursor = "▶ "
			}

			lastSync := "never"
			if !row.lastSync.IsZero() {
				lastSync = row.lastSync.Format("2 Jan 2006")
			}

			emails, senders, triaged := "—", "—", "—"
			if row.count > 0 {
				emails = fmt.Sprintf("%d", row.count)
				senders = fmt.Sprintf("%d", row.total)
				triaged = fmt.Sprintf("%d / %d", row.decided, row.total)
			}

			line := fmt.Sprintf("%s%-30s  %10s  %8s  %8s  %s",
				cursor, row.alias, emails, senders, triaged, lastSync)

			if i == m.ov.cursor {
				b.WriteString(selectedRowStyle.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(renderKeys("↑↓", "select", "enter", "triage senders", "s", "sync", "q", "quit") + "\n")
	b.WriteString(divider(m.width) + "\n")
	b.WriteString(renderStatusBar(m))

	return b.String()
}

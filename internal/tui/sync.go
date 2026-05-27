package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateSync(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.screen = screenOverview
			return m, loadOverviewCmd(m.database, m.appConfig)
		case "up", "k":
			if !m.sy.running && m.sy.preset > 0 {
				m.sy.preset--
			}
		case "down", "j":
			if !m.sy.running && int(m.sy.preset) < len(syncPresetLabels)-1 {
				m.sy.preset++
			}
		case "enter":
			if m.sy.done {
				// Done — back to overview
				m.screen = screenOverview
				return m, loadOverviewCmd(m.database, m.appConfig)
			}
			if !m.sy.running {
				return m.triggerSync()
			}
		}
	}
	return m, nil
}

func (m model) triggerSync() (model, tea.Cmd) {
	if conn, ok := m.connectors[m.sy.alias]; ok {
		// Connector already loaded — start immediately
		_ = conn
		return m.doStartSync()
	}
	// Need to load connector first; sync starts on connectorReadyMsg
	m.setStatus("Connecting to Gmail — a browser window may open…", false)
	return m, loadConnectorCmd(m.sy.alias)
}

func (m model) doStartSync() (model, tea.Cmd) {
	conn, ok := m.connectors[m.sy.alias]
	if !ok {
		m.setStatus("connector not ready", true)
		return m, nil
	}

	// Find the provider string for this alias
	providerStr := "gmail"
	for _, p := range m.appConfig.Providers {
		if p.Alias == m.sy.alias {
			providerStr = string(p.Provider)
			break
		}
	}

	ch := make(chan syncProgressMsg, 100)
	m.sy.ch = ch
	m.sy.running = true
	m.sy.done = false
	m.sy.loaded = 0
	m.sy.inserted = 0
	m.sy.err = nil
	m.setStatus("", false)

	startSyncGoroutine(conn, m.database, m.sy.alias, providerStr, m.sy.preset, ch)

	return m, tea.Batch(
		waitForSyncMsg(ch),
		m.sy.sp.Tick,
	)
}

func (m model) viewSync() string {
	var b strings.Builder

	title := "Sync — " + m.sy.alias
	if m.sy.running || m.sy.done {
		title += "  ·  " + syncPresetLabels[m.sy.preset]
	}
	b.WriteString(titleStyle.Render(title) + "\n")
	b.WriteString(divider(m.width) + "\n")
	b.WriteString("\n")

	switch {
	case m.sy.err != nil:
		b.WriteString(dangerStyle.Render("  Error: "+m.sy.err.Error()) + "\n")
		b.WriteString("\n")
		b.WriteString(renderKeys("esc", "back") + "\n")

	case m.sy.done:
		b.WriteString(successStyle.Render("  Sync complete!") + "\n")
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s emails loaded  ·  %s new\n",
			fmtInt(m.sy.loaded), fmtInt(m.sy.inserted)))
		b.WriteString("\n")
		b.WriteString(renderKeys("enter", "back to overview") + "\n")

	case m.sy.running:
		b.WriteString("  " + m.sy.sp.View() + " Syncing…\n")
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s emails loaded  ·  %s new\n",
			fmtInt(m.sy.loaded), fmtInt(m.sy.inserted)))

	default:
		b.WriteString("  Select time range:\n")
		b.WriteString("\n")
		for i, label := range syncPresetLabels {
			cursor := "    "
			line := label
			if syncPreset(i) == m.sy.preset {
				cursor = "  ▶ "
				line = selectedRowStyle.Render(label)
			}
			b.WriteString(cursor + line + "\n")
		}
		b.WriteString("\n")
		b.WriteString(renderKeys("↑↓", "select", "enter", "start sync", "esc", "back") + "\n")
	}

	b.WriteString(divider(m.width) + "\n")
	b.WriteString(renderStatusBar(m))

	return b.String()
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	// Insert commas every 3 digits from the right
	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(ch)
	}
	return result
}

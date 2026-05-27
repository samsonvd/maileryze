package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	detailHeaderLines = 5 // title + unsub line + blank + divider + total-count line
	detailFooterLines = 3 // divider + action row 1 + action row 2
)

func (m model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.de.loading {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.screen = screenSenders
			return m, nil

		case "up":
			if m.de.cursor > 0 {
				m.de.cursor--
				m.de.scroll = adjustScroll(m.de.cursor, m.de.scroll, m.visibleDetailLines())
			}

		case "down":
			if m.de.cursor < len(m.de.emails)-1 {
				m.de.cursor++
				m.de.scroll = adjustScroll(m.de.cursor, m.de.scroll, m.visibleDetailLines())
			}

		case " ":
			if len(m.de.emails) > 0 {
				m.de.selected[m.de.cursor] = !m.de.selected[m.de.cursor]
				if !m.de.selected[m.de.cursor] {
					delete(m.de.selected, m.de.cursor)
				}
			}

		case "A":
			if len(m.de.selected) == len(m.de.emails) {
				m.de.selected = make(map[int]bool)
			} else {
				for i := range m.de.emails {
					m.de.selected[i] = true
				}
			}

		case "d":
			return m.confirmDeleteSelected()

		case "D":
			return m.confirmDeleteAllFromDetail()

		case "u":
			return m.handleUnsubscribeFromDetail()

		case "k":
			return m.keepSenderFromDetail()
		}
	}
	return m, nil
}

func (m model) visibleDetailLines() int {
	v := m.height - detailHeaderLines - detailFooterLines
	if v < 1 {
		return 1
	}
	return v
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

func (m model) confirmDeleteSelected() (model, tea.Cmd) {
	if len(m.de.selected) == 0 {
		m.setStatus("nothing selected — use [space] to select emails or [D] to trash all", false)
		return m, nil
	}

	conn, ok := m.connectors[m.de.alias]
	if !ok {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}

	var ids []string
	for i := range m.de.selected {
		if i < len(m.de.emails) {
			ids = append(ids, m.de.emails[i].ProviderIdentifier)
		}
	}

	addr := m.de.sender.Address
	m.confirm = confirmState{
		active:    true,
		title:     fmt.Sprintf("Trash %d selected email(s) from %s?", len(ids), addr),
		body:      "Only the emails you selected in your local database will be trashed.",
		action:    trashSelectedCmd(conn, addr, ids),
		addresses: []string{addr},
	}
	return m, nil
}

func (m model) confirmDeleteAllFromDetail() (model, tea.Cmd) {
	conn, ok := m.connectors[m.de.alias]
	if !ok {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}

	addr := m.de.sender.Address
	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Trash ALL emails from %s?", addr),
		body: "This searches Gmail live — all historical emails from this sender\n" +
			"will be moved to Trash, not just what's in your local database.",
		action:    trashAllCmd(conn, addr),
		addresses: []string{addr},
	}
	return m, nil
}

func (m model) handleUnsubscribeFromDetail() (model, tea.Cmd) {
	s := senderItem{sender: m.de.sender}
	m.se.alias = m.de.alias
	switch {
	case s.sender.UnsubscribeURL != "":
		return m.unsubscribeViaURL(s)
	case s.sender.UnsubscribeEmail != "":
		return m.unsubscribeViaEmail(s)
	default:
		m.setStatus("no unsubscribe mechanism for this sender", true)
		return m, nil
	}
}

func (m model) keepSenderFromDetail() (model, tea.Cmd) {
	addr := m.de.sender.Address
	m.screen = screenSenders
	return m, func() tea.Msg {
		return actionDoneMsg{
			senderAddress: addr,
			decision:      "keep",
		}
	}
}

func (m model) viewDetail() string {
	var b strings.Builder

	sender := m.de.sender
	name := sender.Address
	if sender.Name != "" && sender.Name != sender.Address {
		name = sender.Name + " <" + sender.Address + ">"
	}
	b.WriteString("← " + boldStyle.Render(name) + "\n")

	// Unsubscribe line
	switch {
	case sender.UnsubscribeURL != "":
		b.WriteString(mutedStyle.Render(fmt.Sprintf("%d emails  ·  [URL] %s",
			sender.Count, truncate(sender.UnsubscribeURL, m.width-30))) + "\n")
	case sender.UnsubscribeEmail != "":
		b.WriteString(mutedStyle.Render(fmt.Sprintf("%d emails  ·  [EML] %s",
			sender.Count, sender.UnsubscribeEmail)) + "\n")
	default:
		b.WriteString(mutedStyle.Render(fmt.Sprintf("%d emails  ·  no unsubscribe mechanism", sender.Count)) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(divider(m.width) + "\n")

	if m.de.loading {
		b.WriteString("  Loading…\n")
		return b.String()
	}

	if len(m.de.emails) == 0 {
		b.WriteString(mutedStyle.Render("  No emails found in local database.") + "\n")
	} else {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  Showing %d emails (local database only)", len(m.de.emails))) + "\n")

		visH := m.visibleDetailLines() - 1 // -1 for the count line above

		dateWidth := 12
		checkWidth := 3
		cursorWidth := 2
		nameWidth := m.width - cursorWidth - checkWidth - dateWidth - 4

		for i := m.de.scroll; i < m.de.scroll+visH && i < len(m.de.emails); i++ {
			email := m.de.emails[i]

			cursor := "  "
			if i == m.de.cursor {
				cursor = "▶ "
			}

			checkbox := "☐ "
			if m.de.selected[i] {
				checkbox = "☑ "
			}

			subject := truncate(email.Subject, nameWidth)
			date := email.ReceivedAt.Format("2 Jan 2006")

			line := fmt.Sprintf("%s%s%-*s  %s", cursor, checkbox, nameWidth, subject, date)

			switch {
			case i == m.de.cursor && m.de.selected[i]:
				b.WriteString(selectedRowStyle.Render(line) + "\n")
			case i == m.de.cursor:
				b.WriteString(selectedRowStyle.Render(line) + "\n")
			case m.de.selected[i]:
				b.WriteString(successStyle.Render(line) + "\n")
			default:
				b.WriteString(line + "\n")
			}
		}
	}

	b.WriteString(divider(m.width) + "\n")

	selCount := len(m.de.selected)
	row1 := renderKeys(
		"↑↓", "navigate",
		"space", "select",
		"A", "select all",
		fmt.Sprintf("d (%d)", selCount), "trash selected",
		"D", "trash all",
	)
	row2 := renderKeys(
		"u", "unsubscribe",
		"k", "keep sender",
		"esc", "back",
	)
	b.WriteString(row1 + "\n")
	b.WriteString(row2 + "\n")
	b.WriteString(renderStatusBar(m))

	return b.String()
}


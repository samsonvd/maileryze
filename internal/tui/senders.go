package tui

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateSenders(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// While list is filtering, pass all keys through
		if m.se.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.se.list, cmd = m.se.list.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "esc":
			m.screen = screenOverview
			return m, loadOverviewCmd(m.database, m.appConfig)

		case "h":
			m.se.showDecided = !m.se.showDecided
			items := buildSenderItems(m.se.allSenders, m.se.decisions, m.se.selected, m.se.showDecided)
			cmd := m.se.list.SetItems(items)
			pending := countPending(m.se.allSenders, m.se.decisions)
			label := fmt.Sprintf("%s — %d pending", m.se.alias, pending)
			if m.se.showDecided {
				label += " (showing all)"
			}
			m.se.list.Title = label
			return m, cmd

		case "enter":
			return m.openSenderDetail()

		case "d":
			if len(m.se.selected) > 0 {
				return m.confirmBatchDelete()
			}
			return m.confirmDeleteSender()

		case "D":
			if len(m.se.selected) > 0 {
				return m.confirmBatchTrashAndUnsub()
			}
			return m.confirmTrashAndUnsubSender()

		case "u":
			if len(m.se.selected) > 0 {
				return m.confirmBatchUnsub()
			}
			return m.handleUnsubscribe()

		case "k":
			if len(m.se.selected) > 0 {
				return m.batchKeep()
			}
			return m.keepSender()

		case "s":
			if len(m.se.selected) > 0 {
				return m.batchSkip()
			}
			return m.skipSender()

		case "x":
			s, ok := m.selectedSender()
			if !ok {
				return m, nil
			}
			if m.se.selected == nil {
				m.se.selected = make(map[string]bool)
			}
			addr := s.sender.Address
			if m.se.selected[addr] {
				delete(m.se.selected, addr)
			} else {
				m.se.selected[addr] = true
			}
			items := buildSenderItems(m.se.allSenders, m.se.decisions, m.se.selected, m.se.showDecided)
			cmd := m.se.list.SetItems(items)
			return m, cmd

		case "X":
			m.se.selected = make(map[string]bool)
			items := buildSenderItems(m.se.allSenders, m.se.decisions, m.se.selected, m.se.showDecided)
			cmd := m.se.list.SetItems(items)
			return m, cmd
		}

	}

	var cmd tea.Cmd
	m.se.list, cmd = m.se.list.Update(msg)
	return m, cmd
}

func (m model) selectedSender() (senderItem, bool) {
	item := m.se.list.SelectedItem()
	if item == nil {
		return senderItem{}, false
	}
	s, ok := item.(senderItem)
	return s, ok
}

func (m model) openSenderDetail() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}
	m.de = detailState{
		alias:    m.se.alias,
		sender:   s.sender,
		loading:  true,
		selected: make(map[int]bool),
	}
	m.screen = screenDetail
	return m, loadDetailCmd(m.database, m.se.alias, s.sender.Address)
}

func (m model) confirmDeleteSender() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}

	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}

	addr := s.sender.Address
	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Trash all emails from %s?", addr),
		body: "This searches Gmail live — all historical emails from this sender\n" +
			"will be moved to Trash, not just what's in your local database.",
		sp:     m.confirm.sp,
		action: trashAllCmd(conn, addr),
	}
	return m, nil
}

func (m model) handleUnsubscribe() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}

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

func (m model) unsubscribeViaURL(s senderItem) (model, tea.Cmd) {
	rawURL := s.sender.UnsubscribeURL
	addr := s.sender.Address

	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Open unsubscribe link for %s?", addr),
		body:   mutedStyle.Render(truncate(rawURL, 70)),
		sp:     m.confirm.sp,
		action: func() tea.Msg {
			openURL(rawURL)
			return actionDoneMsg{
				senderAddress: addr,
				decision:      "unsubscribed",
			}
		},
	}
	return m, nil
}

func (m model) unsubscribeViaEmail(s senderItem) (model, tea.Cmd) {
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}

	to, subject := parseMailto(s.sender.UnsubscribeEmail)
	addr := s.sender.Address

	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Send unsubscribe email to %s?", to),
		body: mutedStyle.Render(fmt.Sprintf("From: %s\nTo:   %s\nSubj: %s",
			m.se.alias, to, subject)),
		sp:     m.confirm.sp,
		action: sendUnsubEmailCmd(conn, addr, to, subject),
	}
	return m, nil
}

func (m model) keepSender() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}
	addr := s.sender.Address
	_ = m.database // used by actionDoneMsg handler
	return m, func() tea.Msg {
		return actionDoneMsg{
			senderAddress: addr,
			decision:      "keep",
		}
	}
}

func (m model) skipSender() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}
	addr := s.sender.Address
	return m, func() tea.Msg {
		return actionDoneMsg{
			senderAddress: addr,
			decision:      "skipped",
		}
	}
}

func (m model) confirmTrashAndUnsubSender() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	addr := s.sender.Address
	body := "Searches Gmail live — all historical emails trashed and unsubscribe request sent."
	if s.sender.UnsubscribeURL == "" && s.sender.UnsubscribeEmail == "" {
		body = "No unsubscribe mechanism found — will only trash all emails."
	}
	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Trash all + unsubscribe from %s?", addr),
		body:   body,
		sp:     m.confirm.sp,
		action: trashAndUnsubCmd(conn, s.sender),
	}
	return m, nil
}

func (m model) confirmBatchDelete() (model, tea.Cmd) {
	senders := m.selectedSenders()
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Trash all emails from %d senders?", len(senders)),
		body:   "Searches Gmail live — all historical emails from these senders will be moved to Trash.",
		sp:     m.confirm.sp,
		action: batchTrashAllCmd(conn, senders),
	}
	return m, nil
}

func (m model) confirmBatchTrashAndUnsub() (model, tea.Cmd) {
	senders := m.selectedSenders()
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Trash all + unsubscribe from %d senders?", len(senders)),
		body:   "Searches Gmail live — all historical emails trashed. Unsubscribe skipped where unavailable.",
		sp:     m.confirm.sp,
		action: batchTrashAndUnsubCmd(conn, senders),
	}
	return m, nil
}

func (m model) confirmBatchUnsub() (model, tea.Cmd) {
	senders := m.selectedSenders()
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	m.confirm = confirmState{
		active: true,
		title:  fmt.Sprintf("Unsubscribe from %d senders?", len(senders)),
		body:   "Opens unsubscribe URLs and sends unsubscribe emails where available.",
		sp:     m.confirm.sp,
		action: batchUnsubCmd(conn, senders),
	}
	return m, nil
}

func (m model) batchKeep() (model, tea.Cmd) {
	senders := m.selectedSenders()
	addrs := make([]string, len(senders))
	for i, s := range senders {
		addrs[i] = s.Address
	}
	return m, batchDecisionCmd(addrs, "keep")
}

func (m model) batchSkip() (model, tea.Cmd) {
	senders := m.selectedSenders()
	addrs := make([]string, len(senders))
	for i, s := range senders {
		addrs[i] = s.Address
	}
	return m, batchDecisionCmd(addrs, "skipped")
}

func (m model) viewSenders() string {
	var b strings.Builder

	if m.se.loading {
		b.WriteString(titleStyle.Render("Loading senders…") + "\n")
		b.WriteString(divider(m.width) + "\n")
		return b.String()
	}

	// The list handles its own rendering (title, filter, items, status bar)
	b.WriteString(m.se.list.View() + "\n")
	b.WriteString(divider(m.width) + "\n")

	if n := len(m.se.selected); n > 0 {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d selected — actions apply to all selected", n)) + "\n")
	}
	hint := renderKeys(
		"d", "delete all",
		"D", "trash+unsub",
		"u", "unsubscribe",
		"k", "keep",
		"s", "skip",
		"x", "select",
		"X", "clear sel",
		"↵", "detail",
		"h", "show decided",
		"esc", "back",
	)
	b.WriteString(hint + "\n")
	b.WriteString(renderStatusBar(m))

	return b.String()
}

// parseMailto splits "address?subject=..." into (address, subject).
func parseMailto(s string) (to, subject string) {
	s = strings.TrimPrefix(s, "mailto:")
	parts := strings.SplitN(s, "?", 2)
	to = parts[0]
	subject = "Unsubscribe"
	if len(parts) == 2 {
		vals, err := url.ParseQuery(parts[1])
		if err == nil {
			if sub := vals.Get("subject"); sub != "" {
				subject = sub
			}
		}
	}
	return
}

func openURL(rawURL string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", rawURL).Start() //nolint:errcheck
	default:
		exec.Command("xdg-open", rawURL).Start() //nolint:errcheck
	}
}

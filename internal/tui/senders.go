package tui

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"maileryze/internal/db"
)

// ── navigation helpers ────────────────────────────────────────────────────────

// visibleSenderLines returns the number of item rows that fit on screen.
func (m model) visibleSenderLines() int {
	const headerLines = 4 // title + divider + col-header + divider
	footerLines := 3      // divider + key-hints + status-bar
	if m.se.visualMode || len(m.se.selected) > 0 {
		footerLines++ // selection info line
	}
	v := m.height - headerLines - footerLines
	if v < 1 {
		return 1
	}
	return v
}

func (m model) clampSenderCursor() model {
	n := len(m.se.items)
	if n == 0 {
		m.se.cursor = 0
	} else if m.se.cursor >= n {
		m.se.cursor = n - 1
	}
	if m.se.visualAnchor >= n && n > 0 {
		m.se.visualAnchor = n - 1
	}
	m.se.scroll = adjustScroll(m.se.cursor, m.se.scroll, m.visibleSenderLines())
	return m
}

// ── selection helpers ─────────────────────────────────────────────────────────

// selectedSender returns the item under the cursor.
func (m model) selectedSender() (senderItem, bool) {
	if len(m.se.items) == 0 || m.se.cursor >= len(m.se.items) {
		return senderItem{}, false
	}
	return m.se.items[m.se.cursor], true
}

// batchTargets returns the senders the next action should apply to:
// visual range if in visual mode, persistent x-selections otherwise.
func (m model) batchTargets() []senderItem {
	if m.se.visualMode {
		lo, hi := m.se.visualAnchor, m.se.cursor
		if lo > hi {
			lo, hi = hi, lo
		}
		var out []senderItem
		for i := lo; i <= hi && i < len(m.se.items); i++ {
			out = append(out, m.se.items[i])
		}
		return out
	}
	var out []senderItem
	for _, item := range m.se.items {
		if m.se.selected[item.sender.Address] {
			out = append(out, item)
		}
	}
	return out
}

func (m model) hasBatchTargets() bool {
	return m.se.visualMode || len(m.se.selected) > 0
}

// ── update ────────────────────────────────────────────────────────────────────

func (m model) updateSenders(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		// ── exit / cancel ─────────────────────────────────────────────────────
		case "esc":
			if m.se.visualMode {
				m.se.visualMode = false
				return m, nil
			}
			m.screen = screenOverview
			return m, loadOverviewCmd(m.database, m.appConfig)

		case "q":
			m.screen = screenOverview
			return m, loadOverviewCmd(m.database, m.appConfig)

		// ── navigation ────────────────────────────────────────────────────────
		case "up", "k":
			if m.se.cursor > 0 {
				m.se.cursor--
				m.se.scroll = adjustScroll(m.se.cursor, m.se.scroll, m.visibleSenderLines())
			}

		case "down", "j":
			if m.se.cursor < len(m.se.items)-1 {
				m.se.cursor++
				m.se.scroll = adjustScroll(m.se.cursor, m.se.scroll, m.visibleSenderLines())
			}

		case "g":
			m.se.cursor = 0
			m.se.scroll = 0

		case "G":
			if n := len(m.se.items); n > 0 {
				m.se.cursor = n - 1
				m.se.scroll = adjustScroll(m.se.cursor, m.se.scroll, m.visibleSenderLines())
			}

		// ── visual mode ───────────────────────────────────────────────────────
		case "v":
			if m.se.visualMode {
				m.se.visualMode = false
			} else if len(m.se.items) > 0 {
				m.se.visualMode = true
				m.se.visualAnchor = m.se.cursor
			}

		// ── persistent single-item selection ──────────────────────────────────
		case "x":
			if s, ok := m.selectedSender(); ok {
				if m.se.selected == nil {
					m.se.selected = make(map[string]bool)
				}
				addr := s.sender.Address
				if m.se.selected[addr] {
					delete(m.se.selected, addr)
				} else {
					m.se.selected[addr] = true
				}
			}

		case "X":
			m.se.selected = make(map[string]bool)

		// ── show decided toggle ───────────────────────────────────────────────
		case "h":
			m.se.showDecided = !m.se.showDecided
			m.se.items = buildSenderItems(m.se.allSenders, m.se.decisions, m.se.showDecided)
			m = m.clampSenderCursor()

		// ── open detail ───────────────────────────────────────────────────────
		case "enter":
			return m.openSenderDetail()

		// ── actions (dispatch to batch or single) ─────────────────────────────
		case "d":
			if m.hasBatchTargets() {
				return m.confirmBatchDelete()
			}
			return m.confirmDeleteSender()

		case "D":
			if m.hasBatchTargets() {
				return m.confirmBatchTrashAndUnsub()
			}
			return m.confirmTrashAndUnsubSender()

		case "u":
			if m.hasBatchTargets() {
				return m.confirmBatchUnsub()
			}
			return m.handleUnsubscribe()

		case "K":
			if m.hasBatchTargets() {
				return m.batchKeep()
			}
			return m.keepSender()

		case "s":
			if m.hasBatchTargets() {
				return m.batchSkip()
			}
			return m.skipSender()
		}
	}
	return m, nil
}

// ── single-item actions ───────────────────────────────────────────────────────

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
		action:    trashAllCmd(conn, addr),
		addresses: []string{addr},
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
		active:    true,
		title:     fmt.Sprintf("Open unsubscribe link for %s?", addr),
		body:      mutedStyle.Render(truncate(rawURL, 70)),
		addresses: []string{addr},
		action: func() tea.Msg {
			openURL(rawURL)
			return actionDoneMsg{senderAddress: addr, decision: "unsubscribed"}
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
		action:    sendUnsubEmailCmd(conn, addr, to, subject),
		addresses: []string{addr},
	}
	return m, nil
}

func (m model) keepSender() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}
	addr := s.sender.Address
	return m, func() tea.Msg {
		return actionDoneMsg{senderAddress: addr, decision: "keep"}
	}
}

func (m model) skipSender() (model, tea.Cmd) {
	s, ok := m.selectedSender()
	if !ok {
		return m, nil
	}
	addr := s.sender.Address
	return m, func() tea.Msg {
		return actionDoneMsg{senderAddress: addr, decision: "skipped"}
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
		active:    true,
		title:     fmt.Sprintf("Trash all + unsubscribe from %s?", addr),
		body:      body,
		action:    trashAndUnsubCmd(conn, s.sender),
		addresses: []string{addr},
	}
	return m, nil
}

// ── batch actions ─────────────────────────────────────────────────────────────

func (m model) confirmBatchDelete() (model, tea.Cmd) {
	targets := m.batchTargets()
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	senders := itemsToSenders(targets)
	addrs := senderAddresses(senders)
	m.confirm = confirmState{
		active:    true,
		title:     fmt.Sprintf("Trash all emails from %d senders?", len(senders)),
		body:      "Searches Gmail live — all historical emails from these senders will be moved to Trash.",
		action:    batchTrashAllCmd(conn, senders),
		addresses: addrs,
	}
	return m, nil
}

func (m model) confirmBatchTrashAndUnsub() (model, tea.Cmd) {
	targets := m.batchTargets()
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	senders := itemsToSenders(targets)
	addrs := senderAddresses(senders)
	m.confirm = confirmState{
		active:    true,
		title:     fmt.Sprintf("Trash all + unsubscribe from %d senders?", len(senders)),
		body:      "Searches Gmail live — all historical emails trashed. Unsubscribe skipped where unavailable.",
		action:    batchTrashAndUnsubCmd(conn, senders),
		addresses: addrs,
	}
	return m, nil
}

func (m model) confirmBatchUnsub() (model, tea.Cmd) {
	targets := m.batchTargets()
	conn, hasConn := m.connectors[m.se.alias]
	if !hasConn {
		m.setStatus("Still connecting to Gmail…", false)
		return m, nil
	}
	senders := itemsToSenders(targets)
	addrs := senderAddresses(senders)
	m.confirm = confirmState{
		active:    true,
		title:     fmt.Sprintf("Unsubscribe from %d senders?", len(senders)),
		body:      "Opens unsubscribe URLs and sends unsubscribe emails where available.",
		action:    batchUnsubCmd(conn, senders),
		addresses: addrs,
	}
	return m, nil
}

func (m model) batchKeep() (model, tea.Cmd) {
	targets := m.batchTargets()
	addrs := make([]string, len(targets))
	for i, t := range targets {
		addrs[i] = t.sender.Address
	}
	return m, batchDecisionCmd(addrs, "keep")
}

func (m model) batchSkip() (model, tea.Cmd) {
	targets := m.batchTargets()
	addrs := make([]string, len(targets))
	for i, t := range targets {
		addrs[i] = t.sender.Address
	}
	return m, batchDecisionCmd(addrs, "skipped")
}

func itemsToSenders(items []senderItem) []db.Sender {
	out := make([]db.Sender, len(items))
	for i, it := range items {
		out[i] = it.sender
	}
	return out
}

func senderAddresses(senders []db.Sender) []string {
	out := make([]string, len(senders))
	for i, s := range senders {
		out[i] = s.Address
	}
	return out
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m model) viewSenders() string {
	var b strings.Builder

	if m.se.loading {
		b.WriteString(titleStyle.Render("Loading senders…") + "\n")
		b.WriteString(divider(m.width) + "\n")
		return b.String()
	}

	// ── title
	pending := countPending(m.se.allSenders, m.se.decisions)
	title := fmt.Sprintf("%s — %d pending", m.se.alias, pending)
	if m.se.showDecided {
		title += " (showing all)"
	}
	b.WriteString(titleStyle.Render(title) + "\n")
	b.WriteString(divider(m.width) + "\n")

	// ── column geometry
	//   cur(2) sel(1) sp(1) unsub(5) sp(1) name(nameW) sp(2) count(6)  = nameW+18
	const fixedCols = 18
	nameW := m.width - fixedCols
	if nameW < 10 {
		nameW = 10
	}

	// ── column header
	hdr := fmt.Sprintf("   [   ] %-*s  %6s", nameW, "SENDER", "COUNT")
	b.WriteString(mutedStyle.Render(hdr) + "\n")
	b.WriteString(divider(m.width) + "\n")

	// ── visual range bounds
	visLo, visHi := 0, -1
	if m.se.visualMode {
		visLo, visHi = m.se.visualAnchor, m.se.cursor
		if visLo > visHi {
			visLo, visHi = visHi, visLo
		}
	}

	// ── rows
	visH := m.visibleSenderLines()
	for i := m.se.scroll; i < m.se.scroll+visH && i < len(m.se.items); i++ {
		item := m.se.items[i]

		isCursor := i == m.se.cursor
		inVisual := m.se.visualMode && i >= visLo && i <= visHi
		isSelected := m.se.selected[item.sender.Address]

		cur := "  "
		if isCursor {
			cur = "▶ "
		}

		selMark := " "
		if isSelected || inVisual {
			selMark = "✓"
		}

		unsub := "[   ]"
		if m.se.pending[item.sender.Address] {
			unsub = m.se.sp.View() + "    " // spinner (1 char) + 4 spaces = 5 visible
		} else if item.sender.UnsubscribeURL != "" {
			unsub = "[URL]"
		} else if item.sender.UnsubscribeEmail != "" {
			unsub = "[EML]"
		}

		name := senderDisplayName(item.sender)
		if item.decision != "" {
			suffix := " · " + decisionShort(item.decision)
			name = truncate(name, nameW-len([]rune(suffix))) + suffix
		} else {
			name = truncate(name, nameW)
		}

		count := fmtInt(item.sender.Count)
		line := fmt.Sprintf("%s%s %s %-*s  %6s", cur, selMark, unsub, nameW, name, count)

		switch {
		case isCursor:
			b.WriteString(selectedRowStyle.Render(line) + "\n")
		case inVisual:
			b.WriteString(visualRangeStyle.Render(line) + "\n")
		case item.decision != "":
			b.WriteString(decidedRowStyle.Render(line) + "\n")
		default:
			b.WriteString(line + "\n")
		}
	}

	// ── footer
	b.WriteString(divider(m.width) + "\n")

	if m.se.visualMode {
		n := visHi - visLo + 1
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  -- VISUAL -- %d senders", n)) + "\n")
	} else if n := len(m.se.selected); n > 0 {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d selected — actions apply to all selected", n)) + "\n")
	}

	hint := renderKeys(
		"↑↓/jk", "navigate",
		"v", "visual",
		"x", "select",
		"d", "delete",
		"D", "trash+unsub",
		"u", "unsub",
		"K", "keep",
		"s", "skip",
		"↵", "detail",
		"h", "decided",
		"esc", "back",
	)
	b.WriteString(hint + "\n")
	b.WriteString(renderStatusBar(m))

	return b.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func senderDisplayName(s db.Sender) string {
	if s.Name != "" && s.Name != s.Address {
		return s.Name
	}
	return s.Address
}

func decisionShort(d string) string {
	switch d {
	case "keep":
		return "kept"
	case "deleted":
		return "deleted"
	case "unsubscribed":
		return "unsub"
	case "unsubscribed_deleted":
		return "unsub+del"
	case "skipped":
		return "skipped"
	}
	return d
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

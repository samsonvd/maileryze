package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"maileryze/internal/wal"
)

func (m model) visibleTriageLines() int {
	const headerLines = 2 // title + divider
	footerLines := 3      // divider + key-hints + status-bar
	if len(m.tr.staged) > 0 {
		footerLines++ // staged count line
	}
	v := m.height - headerLines - footerLines
	if v < 1 {
		return 1
	}
	return v
}

func (m model) updateTriage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "tab":
			m.screen = screenPlan
			return m, nil

		case "up", "k":
			if m.tr.cursor > 0 {
				m.tr.cursor--
				m.tr.scroll = adjustScroll(m.tr.cursor, m.tr.scroll, m.visibleTriageLines())
			}

		case "down", "j":
			if m.tr.cursor < len(m.tr.senders)-1 {
				m.tr.cursor++
				m.tr.scroll = adjustScroll(m.tr.cursor, m.tr.scroll, m.visibleTriageLines())
			}

		case "g":
			m.tr.cursor = 0
			m.tr.scroll = 0

		case "G":
			if n := len(m.tr.senders); n > 0 {
				m.tr.cursor = n - 1
				m.tr.scroll = adjustScroll(m.tr.cursor, m.tr.scroll, m.visibleTriageLines())
			}

		case "s":
			return m.stageCurrent(wal.ActionKeep)

		case "u":
			return m.stageCurrent(wal.ActionUnsub)

		case "d":
			return m.stageCurrent(wal.ActionDelete)

		case "enter":
			if len(m.tr.staged) == 0 {
				return m, nil
			}
			staged := make(map[string]wal.Action, len(m.tr.staged))
			for k, v := range m.tr.staged {
				staged[k] = v
			}
			return m, commitStagedCmd(m.w, staged)

		case "esc":
			if len(m.tr.staged) > 0 {
				m.tr.staged = make(map[string]wal.Action)
				m.setStatus("", false)
			}
		}
	}
	return m, nil
}

// stageCurrent stages the action for the sender under the cursor, toggling
// it off if the same action is pressed again. Advances the cursor after staging.
func (m model) stageCurrent(action wal.Action) (model, tea.Cmd) {
	if len(m.tr.senders) == 0 || m.tr.cursor >= len(m.tr.senders) {
		return m, nil
	}
	addr := m.tr.senders[m.tr.cursor].Address
	if existing, ok := m.tr.staged[addr]; ok && existing == action {
		delete(m.tr.staged, addr)
	} else {
		m.tr.staged[addr] = action
		// Auto-advance after staging so the user can key through the list quickly.
		if m.tr.cursor < len(m.tr.senders)-1 {
			m.tr.cursor++
			m.tr.scroll = adjustScroll(m.tr.cursor, m.tr.scroll, m.visibleTriageLines())
		}
	}
	return m, nil
}

func (m model) viewTriage() string {
	var b strings.Builder

	// ── title ─────────────────────────────────────────────────────────────────
	keep, unsub, del := m.w.Counts()
	planned := keep + unsub + del
	title := fmt.Sprintf("maileryze — %s", m.alias)
	if planned > 0 {
		title += fmt.Sprintf("  ·  plan: %d", planned)
	}
	b.WriteString(titleStyle.Render(title) + "\n")
	b.WriteString(divider(m.width) + "\n")

	// ── loading splash ────────────────────────────────────────────────────────
	if m.tr.fetching && len(m.tr.senders) == 0 {
		b.WriteString("\n  " + m.tr.sp.View() + " Connecting and fetching emails…\n\n")
		b.WriteString(divider(m.width) + "\n")
		b.WriteString(renderStatusBar(m))
		return b.String()
	}

	// ── column geometry: cur(2) stg(1) sp(1) sender(senderW) gap(2) subject(rest)
	const overhead = 6
	available := m.width - overhead
	senderW := available * 2 / 5
	if senderW > 45 {
		senderW = 45
	}
	if senderW < 20 {
		senderW = 20
	}
	subjectW := available - senderW
	if subjectW < 10 {
		subjectW = 10
	}

	// ── build staged-domain set for related highlighting ──────────────────────
	stagedDomains := make(map[string]bool, len(m.tr.staged))
	for addr := range m.tr.staged {
		if i := strings.LastIndex(addr, "@"); i >= 0 {
			stagedDomains[addr[i+1:]] = true
		}
	}

	// ── rows ──────────────────────────────────────────────────────────────────
	visH := m.visibleTriageLines()
	for i := m.tr.scroll; i < m.tr.scroll+visH && i < len(m.tr.senders); i++ {
		s := m.tr.senders[i]
		isCursor := i == m.tr.cursor
		stagedAction, isStaged := m.tr.staged[s.Address]

		var isRelated bool
		if !isStaged {
			if j := strings.LastIndex(s.Address, "@"); j >= 0 {
				isRelated = stagedDomains[s.Address[j+1:]]
			}
		}

		cur := "  "
		if isCursor {
			cur = "▶ "
		}

		stgMark := " "
		if isStaged {
			switch stagedAction {
			case wal.ActionKeep:
				stgMark = "S"
			case wal.ActionUnsub:
				stgMark = "U"
			case wal.ActionDelete:
				stgMark = "D"
			}
		}

		sender := truncate(senderID(s), senderW)
		subject := truncate(s.Subject, subjectW)
		line := fmt.Sprintf("%s%s %-*s  %s", cur, stgMark, senderW, sender, subject)

		switch {
		case isCursor:
			b.WriteString(selectedRowStyle.Render(line) + "\n")
		case isStaged && stagedAction == wal.ActionKeep:
			b.WriteString(stagedKeepStyle.Render(line) + "\n")
		case isStaged && stagedAction == wal.ActionUnsub:
			b.WriteString(stagedUnsubStyle.Render(line) + "\n")
		case isStaged && stagedAction == wal.ActionDelete:
			b.WriteString(stagedDeleteStyle.Render(line) + "\n")
		case isRelated:
			b.WriteString(relatedStyle.Render(line) + "\n")
		default:
			b.WriteString(line + "\n")
		}
	}

	if len(m.tr.senders) == 0 && m.tr.fetchDone {
		b.WriteString("\n  " + successStyle.Render("All senders triaged!") + "\n")
		b.WriteString(mutedStyle.Render("  Press [tab] to view the plan and execute actions.") + "\n")
	}

	// ── footer ────────────────────────────────────────────────────────────────
	b.WriteString(divider(m.width) + "\n")
	if n := len(m.tr.staged); n > 0 {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  %d staged — [enter] commit · [esc] clear", n)) + "\n")
	}
	b.WriteString(renderKeys(
		"↑↓/kj", "navigate",
		"s", "keep",
		"u", "unsub",
		"d", "delete",
		"↵", "commit staged",
		"tab", "plan",
		"q", "quit",
	) + "\n")
	b.WriteString(renderStatusBar(m))

	return b.String()
}

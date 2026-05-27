package tui

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"maileryze/internal/cfg"
	"maileryze/internal/connector"
	"maileryze/internal/factory"
	"maileryze/internal/wal"
)

type screen int

const (
	screenTriage screen = iota
	screenPlan
)

// ── Sender ────────────────────────────────────────────────────────────────────

type Sender struct {
	Name             string
	Address          string
	Count            int
	Subject          string // most recent subject (Gmail returns newest first)
	UnsubscribeURL   string
	UnsubscribeEmail string
}

// senderID returns "Display Name <email>" or just "email" if no display name.
func senderID(s Sender) string {
	if s.Name != "" && s.Name != s.Address {
		return s.Name + " <" + s.Address + ">"
	}
	return s.Address
}

// ── message types ─────────────────────────────────────────────────────────────

type connectorReadyMsg struct {
	conn connector.WritableConnector
	err  error
}

type fetchProgressMsg struct {
	senders []Sender
	total   int
	done    bool
	err     error
}

type commitDoneMsg struct {
	addresses []string
	err       error
}

type executionStartedMsg struct {
	ch chan executeProgressMsg
}

type executeProgressMsg struct {
	address string
	action  wal.Action
	count   int
	err     error
	done    bool
}

// ── state ─────────────────────────────────────────────────────────────────────

type triageState struct {
	senders    []Sender
	staged     map[string]wal.Action
	cursor     int
	scroll     int
	fetching   bool
	fetchTotal int
	fetchDone  bool
	fetchCh    chan fetchProgressMsg
	sp         spinner.Model
}

type planState struct {
	executing bool
	execCh    chan executeProgressMsg
	done      []string
	errors    map[string]error
	sp        spinner.Model
}

type model struct {
	appConfig  *cfg.AppConfig
	conn       connector.WritableConnector
	alias      string
	w          *wal.WAL
	senderInfo map[string]Sender // all seen senders including committed

	width, height int
	screen        screen

	tr triageState
	pl planState

	status    string
	statusErr bool
}

func New(alias string, w *wal.WAL, appConfig *cfg.AppConfig) model {
	fetchSp := spinner.New()
	fetchSp.Spinner = spinner.Dot
	fetchSp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	execSp := spinner.New()
	execSp.Spinner = spinner.Dot
	execSp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return model{
		appConfig:  appConfig,
		alias:      alias,
		w:          w,
		senderInfo: make(map[string]Sender),
		screen:     screenTriage,
		tr: triageState{
			staged:   make(map[string]wal.Action),
			fetching: true,
			sp:       fetchSp,
		},
		pl: planState{
			errors: make(map[string]error),
			sp:     execSp,
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadConnectorCmd(m.alias),
		m.tr.sp.Tick,
	)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmds []tea.Cmd
		if m.tr.fetching {
			m.tr.sp, _ = m.tr.sp.Update(msg)
			cmds = append(cmds, m.tr.sp.Tick)
		}
		if m.pl.executing {
			m.pl.sp, _ = m.pl.sp.Update(msg)
			cmds = append(cmds, m.pl.sp.Tick)
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case connectorReadyMsg:
		if msg.err != nil {
			m.tr.fetching = false
			m.setStatus("login failed: "+msg.err.Error(), true)
			return m, nil
		}
		m.conn = msg.conn
		ch := make(chan fetchProgressMsg, 1)
		m.tr.fetchCh = ch
		startFetchGoroutine(msg.conn, m.w, ch)
		return m, waitForFetchMsg(ch)

	case fetchProgressMsg:
		if msg.err != nil {
			m.tr.fetching = false
			m.setStatus("fetch error: "+msg.err.Error(), true)
			return m, nil
		}
		m.tr.fetchTotal = msg.total

		// Record sender info for execution lookups, filter out decided senders.
		for _, s := range msg.senders {
			m.senderInfo[s.Address] = s
		}
		undecided := make([]Sender, 0, len(msg.senders))
		for _, s := range msg.senders {
			if !m.w.IsDecided(s.Address) {
				undecided = append(undecided, s)
			}
		}

		// Try to keep the cursor on the same sender across updates.
		var cursorAddr string
		if len(m.tr.senders) > 0 && m.tr.cursor < len(m.tr.senders) {
			cursorAddr = m.tr.senders[m.tr.cursor].Address
		}
		m.tr.senders = undecided
		if cursorAddr != "" {
			for i, s := range m.tr.senders {
				if s.Address == cursorAddr {
					m.tr.cursor = i
					m.tr.scroll = adjustScroll(m.tr.cursor, m.tr.scroll, m.visibleTriageLines())
					break
				}
			}
		}
		if m.tr.cursor >= len(m.tr.senders) {
			m.tr.cursor = max(0, len(m.tr.senders)-1)
		}

		if msg.done {
			m.tr.fetching = false
			m.tr.fetchDone = true
		} else {
			return m, waitForFetchMsg(m.tr.fetchCh)
		}
		return m, nil

	case commitDoneMsg:
		if msg.err != nil {
			m.setStatus("error saving decisions: "+msg.err.Error(), true)
			return m, nil
		}
		decided := make(map[string]bool, len(msg.addresses))
		for _, addr := range msg.addresses {
			decided[addr] = true
		}
		var filtered []Sender
		for _, s := range m.tr.senders {
			if !decided[s.Address] {
				filtered = append(filtered, s)
			}
		}
		m.tr.senders = filtered
		m.tr.staged = make(map[string]wal.Action)
		m.tr.cursor = 0
		m.tr.scroll = 0
		m.setStatus(fmt.Sprintf("committed %d decision(s) to plan", len(msg.addresses)), false)
		return m, nil

	case executionStartedMsg:
		m.pl.executing = true
		m.pl.execCh = msg.ch
		return m, tea.Batch(waitForExecMsg(msg.ch), m.pl.sp.Tick)

	case executeProgressMsg:
		if msg.done {
			m.pl.executing = false
			m.setStatus(fmt.Sprintf("execution complete — %d actions", len(m.pl.done)), false)
			return m, nil
		}
		m.pl.done = append(m.pl.done, msg.address)
		if msg.err != nil {
			m.pl.errors[msg.address] = msg.err
		}
		return m, waitForExecMsg(m.pl.execCh)
	}

	switch m.screen {
	case screenTriage:
		return m.updateTriage(msg)
	case screenPlan:
		return m.updatePlan(msg)
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	switch m.screen {
	case screenTriage:
		return m.viewTriage()
	case screenPlan:
		return m.viewPlan()
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m *model) setStatus(msg string, isErr bool) {
	m.status = msg
	m.statusErr = isErr
}

func renderStatusBar(m model) string {
	switch m.screen {
	case screenTriage:
		if m.tr.fetching {
			return mutedStyle.Render(fmt.Sprintf("%s fetching… %s emails scanned", m.tr.sp.View(), fmtInt(m.tr.fetchTotal)))
		}
	case screenPlan:
		if m.pl.executing {
			return mutedStyle.Render(fmt.Sprintf("%s executing… %d done", m.pl.sp.View(), len(m.pl.done)))
		}
	}
	if m.status == "" {
		return ""
	}
	if m.statusErr {
		return statusErrStyle.Render(m.status)
	}
	return statusStyle.Render(m.status)
}

// ── async commands ────────────────────────────────────────────────────────────

func loadConnectorCmd(alias string) tea.Cmd {
	return func() tea.Msg {
		conn, _, err := factory.Connect(alias)
		return connectorReadyMsg{conn: conn, err: err}
	}
}

func startFetchGoroutine(conn connector.WritableConnector, w *wal.WAL, ch chan fetchProgressMsg) {
	go func() {
		defer close(ch)

		ctx := context.Background()
		start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Now().Add(48 * time.Hour)
		fetchCh := conn.Fetch(ctx, start, end)

		senders := make(map[string]*Sender)
		total := 0

		for result := range fetchCh {
			if result.Err != nil {
				select {
				case ch <- fetchProgressMsg{err: result.Err, done: true}:
				default:
				}
				return
			}

			addr := result.Value.Sender.Address
			if addr != "" && !w.IsDecided(addr) {
				if s, ok := senders[addr]; ok {
					s.Count++
					if s.UnsubscribeURL == "" {
						s.UnsubscribeURL = result.Value.Unsubscribe.URL
					}
					if s.UnsubscribeEmail == "" {
						s.UnsubscribeEmail = result.Value.Unsubscribe.Email
					}
				} else {
					senders[addr] = &Sender{
						Name:             result.Value.Sender.Name,
						Address:          addr,
						Count:            1,
						Subject:          result.Value.Subject,
						UnsubscribeURL:   result.Value.Unsubscribe.URL,
						UnsubscribeEmail: result.Value.Unsubscribe.Email,
					}
				}
			}
			total++

			if total%50 == 0 {
				snap := snapshotSenders(senders)
				select {
				case ch <- fetchProgressMsg{senders: snap, total: total}:
				default:
				}
			}
		}

		snap := snapshotSenders(senders)
		select {
		case ch <- fetchProgressMsg{senders: snap, total: total, done: true}:
		default:
		}
	}()
}

func snapshotSenders(m map[string]*Sender) []Sender {
	out := make([]Sender, 0, len(m))
	for _, s := range m {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count
	})
	return out
}

func waitForFetchMsg(ch chan fetchProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return fetchProgressMsg{done: true}
		}
		return msg
	}
}

func commitStagedCmd(w *wal.WAL, staged map[string]wal.Action) tea.Cmd {
	return func() tea.Msg {
		addresses := make([]string, 0, len(staged))
		for addr, action := range staged {
			if err := w.Mark(addr, action); err != nil {
				return commitDoneMsg{err: err}
			}
			addresses = append(addresses, addr)
		}
		return commitDoneMsg{addresses: addresses}
	}
}

func startExecuteCmd(conn connector.WritableConnector, senderInfo map[string]Sender, unsubAddrs, deleteAddrs []string) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan executeProgressMsg, 1)
		go executeGoroutine(conn, senderInfo, unsubAddrs, deleteAddrs, ch)
		return executionStartedMsg{ch: ch}
	}
}

func executeGoroutine(conn connector.WritableConnector, senderInfo map[string]Sender, unsubAddrs, deleteAddrs []string, ch chan<- executeProgressMsg) {
	defer close(ch)
	ctx := context.Background()

	for _, addr := range deleteAddrs {
		count, err := conn.TrashAllFrom(ctx, addr)
		if err == nil {
			if s, ok := senderInfo[addr]; ok {
				performUnsub(conn, ctx, s) //nolint:errcheck
			}
		}
		ch <- executeProgressMsg{address: addr, action: wal.ActionDelete, count: count, err: err}
	}

	for _, addr := range unsubAddrs {
		var err error
		if s, ok := senderInfo[addr]; ok {
			err = performUnsub(conn, ctx, s)
		}
		ch <- executeProgressMsg{address: addr, action: wal.ActionUnsub, err: err}
	}

	ch <- executeProgressMsg{done: true}
}

func performUnsub(conn connector.WritableConnector, ctx context.Context, s Sender) error {
	switch {
	case s.UnsubscribeURL != "":
		openURL(s.UnsubscribeURL)
		return nil
	case s.UnsubscribeEmail != "":
		to, subject := parseMailto(s.UnsubscribeEmail)
		return conn.SendEmail(ctx, to, subject, "Please unsubscribe me from your mailing list.")
	}
	return nil
}

func waitForExecMsg(ch chan executeProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return executeProgressMsg{done: true}
		}
		return msg
	}
}

// ── utilities used across screens ────────────────────────────────────────────

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

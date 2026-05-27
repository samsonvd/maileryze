package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"maileryze/internal/cfg"
	"maileryze/internal/connector"
	"maileryze/internal/db"
	"maileryze/internal/factory"
)

type screen int

const (
	screenOverview screen = iota
	screenSync
	screenSenders
	screenDetail
)

// ── message types ─────────────────────────────────────────────────────────────

type overviewLoadedMsg struct {
	rows []aliasRow
	err  error
}

type connectorReadyMsg struct {
	alias string
	conn  connector.WritableConnector
	err   error
}

type sendersLoadedMsg struct {
	alias     string
	senders   []db.Sender
	decisions map[string]db.SenderDecision
	err       error
}

type detailLoadedMsg struct {
	emails []db.EmailRecord
	err    error
}

type syncProgressMsg struct {
	loaded   int
	inserted int
	done     bool
	err      error
}

type actionDoneMsg struct {
	senderAddress string
	decision      string
	count         int
	err           error
}

// ── state types ───────────────────────────────────────────────────────────────

type aliasRow struct {
	alias    string
	address  string
	provider string
	count    int
	oldest   time.Time
	newest   time.Time
	lastSync time.Time
	decided  int
	total    int
}

type overviewState struct {
	rows    []aliasRow
	cursor  int
	loading bool
}

type syncPreset int

const (
	preset90Days syncPreset = iota
	presetYear
	presetAllTime
)

var syncPresetLabels = []string{"Last 90 days", "Last year", "All time"}

type syncState struct {
	alias    string
	preset   syncPreset
	running  bool
	loaded   int
	inserted int
	done     bool
	err      error
	sp       spinner.Model
	ch       chan syncProgressMsg
}

type senderItem struct {
	sender   db.Sender
	decision string
}

func (s senderItem) Title() string {
	icon := "   "
	switch {
	case s.sender.UnsubscribeURL != "":
		icon = "URL"
	case s.sender.UnsubscribeEmail != "":
		icon = "EML"
	}
	name := s.sender.Address
	if s.sender.Name != "" && s.sender.Name != s.sender.Address {
		name = s.sender.Name
	}
	label := fmt.Sprintf("[%s] %s", icon, name)
	if s.decision != "" {
		return decidedRowStyle.Render(label)
	}
	return label
}

func (s senderItem) Description() string {
	count := fmt.Sprintf("%d emails", s.sender.Count)
	if s.decision != "" {
		return decidedRowStyle.Render(count + "  ·  " + decisionLabel(s.decision))
	}
	if s.sender.Name != "" && s.sender.Name != s.sender.Address {
		return mutedStyle.Render(s.sender.Address + "  ·  " + count)
	}
	return mutedStyle.Render(count)
}

func (s senderItem) FilterValue() string {
	return s.sender.Address + " " + s.sender.Name
}

type sendersState struct {
	alias       string
	allSenders  []db.Sender
	decisions   map[string]db.SenderDecision
	list        list.Model
	showDecided bool
	loading     bool
}

type detailState struct {
	alias    string
	sender   db.Sender
	emails   []db.EmailRecord
	cursor   int
	scroll   int
	selected map[int]bool
	loading  bool
}

type confirmState struct {
	active  bool
	title   string
	body    string
	working bool
	sp      spinner.Model
	action  tea.Cmd
}

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	database   *sql.DB
	appConfig  *cfg.AppConfig
	connectors map[string]connector.WritableConnector

	width  int
	height int

	screen screen

	ov      overviewState
	sy      syncState
	se      sendersState
	de      detailState
	confirm confirmState

	status    string
	statusErr bool
}

func New(database *sql.DB, appConfig *cfg.AppConfig) model {
	syncSp := spinner.New()
	syncSp.Spinner = spinner.Dot

	confirmSp := spinner.New()
	confirmSp.Spinner = spinner.Dot
	confirmSp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return model{
		database:   database,
		appConfig:  appConfig,
		connectors: make(map[string]connector.WritableConnector),
		screen:     screenOverview,
		ov:         overviewState{loading: true},
		sy:         syncState{sp: syncSp},
		confirm:    confirmState{sp: confirmSp},
	}
}

func (m model) Init() tea.Cmd {
	return loadOverviewCmd(m.database, m.appConfig)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.screen == screenSenders && !m.se.loading {
			m.se.list.SetSize(msg.Width, senderListHeight(msg.Height))
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.confirm.active {
			return m.updateConfirm(msg)
		}

	case spinner.TickMsg:
		if m.confirm.working {
			m.confirm.sp, _ = m.confirm.sp.Update(msg)
			cmds = append(cmds, m.confirm.sp.Tick)
		}
		if m.sy.running {
			m.sy.sp, _ = m.sy.sp.Update(msg)
			cmds = append(cmds, m.sy.sp.Tick)
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}

	case overviewLoadedMsg:
		m.ov.loading = false
		if msg.err != nil {
			m.setStatus("error loading overview: "+msg.err.Error(), true)
		} else {
			m.ov.rows = msg.rows
		}
		return m, nil

	case connectorReadyMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("login failed for %s: %s", msg.alias, msg.err), true)
			return m, nil
		}
		m.connectors[msg.alias] = msg.conn
		m.setStatus("connected to "+msg.alias, false)
		// Auto-start sync when the connector arrives while we're on the sync screen
		if m.screen == screenSync && m.sy.alias == msg.alias && !m.sy.running && !m.sy.done {
			return m.doStartSync()
		}
		return m, nil

	case sendersLoadedMsg:
		m.se.loading = false
		if msg.err != nil {
			m.setStatus("error loading senders: "+msg.err.Error(), true)
		} else {
			m.se.allSenders = msg.senders
			m.se.decisions = msg.decisions
			items := buildSenderItems(msg.senders, msg.decisions, m.se.showDecided)
			cmd := m.se.list.SetItems(items)
			cmds = append(cmds, cmd)
			pending := countPending(msg.senders, msg.decisions)
			m.se.list.Title = fmt.Sprintf("%s — %d pending", msg.alias, pending)
		}
		return m, tea.Batch(cmds...)

	case detailLoadedMsg:
		m.de.loading = false
		if msg.err != nil {
			m.setStatus("error loading emails: "+msg.err.Error(), true)
		} else {
			m.de.emails = msg.emails
			m.de.cursor = 0
			m.de.scroll = 0
			m.de.selected = make(map[int]bool)
		}
		return m, nil

	case syncProgressMsg:
		if msg.err != nil {
			m.sy.running = false
			m.sy.done = false
			m.sy.err = msg.err
			m.setStatus("sync error: "+msg.err.Error(), true)
			return m, nil
		}
		m.sy.loaded = msg.loaded
		m.sy.inserted = msg.inserted
		if msg.done {
			m.sy.running = false
			m.sy.done = true
			m.setStatus(fmt.Sprintf("sync complete — %d loaded, %d new", msg.loaded, msg.inserted), false)
		} else if m.sy.ch != nil {
			return m, waitForSyncMsg(m.sy.ch)
		}
		return m, nil

	case actionDoneMsg:
		m.confirm.working = false
		m.confirm.active = false
		if msg.err != nil {
			m.setStatus("action failed: "+msg.err.Error(), true)
			return m, nil
		}
		_ = db.SaveDecision(m.database, m.se.alias, msg.senderAddress, msg.decision, msg.count)
		if m.se.decisions == nil {
			m.se.decisions = make(map[string]db.SenderDecision)
		}
		m.se.decisions[msg.senderAddress] = db.SenderDecision{
			Alias:          m.se.alias,
			SenderAddress:  msg.senderAddress,
			Decision:       msg.decision,
			EmailsAffected: msg.count,
		}
		if m.se.allSenders != nil {
			items := buildSenderItems(m.se.allSenders, m.se.decisions, m.se.showDecided)
			cmd := m.se.list.SetItems(items)
			cmds = append(cmds, cmd)
			pending := countPending(m.se.allSenders, m.se.decisions)
			m.se.list.Title = fmt.Sprintf("%s — %d pending", m.se.alias, pending)
		}
		m.setStatus(actionStatusMsg(msg), false)
		// After a delete/unsub action from the detail screen, return to senders
		if m.screen == screenDetail {
			switch msg.decision {
			case "deleted", "unsubscribed", "unsubscribed_deleted":
				m.screen = screenSenders
			}
		}
		return m, tea.Batch(cmds...)
	}

	switch m.screen {
	case screenOverview:
		return m.updateOverview(msg)
	case screenSync:
		return m.updateSync(msg)
	case screenSenders:
		return m.updateSenders(msg)
	case screenDetail:
		return m.updateDetail(msg)
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.confirm.active {
		return m.viewConfirm()
	}
	switch m.screen {
	case screenOverview:
		return m.viewOverview()
	case screenSync:
		return m.viewSync()
	case screenSenders:
		return m.viewSenders()
	case screenDetail:
		return m.viewDetail()
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m *model) setStatus(msg string, isErr bool) {
	m.status = msg
	m.statusErr = isErr
}

func senderListHeight(total int) int {
	h := total - 2
	if h < 1 {
		return 1
	}
	return h
}

func decisionLabel(d string) string {
	switch d {
	case "keep":
		return successStyle.Render("kept")
	case "deleted":
		return dangerStyle.Render("deleted")
	case "unsubscribed":
		return warningStyle.Render("unsubscribed")
	case "unsubscribed_deleted":
		return dangerStyle.Render("unsub+deleted")
	case "skipped":
		return mutedStyle.Render("skipped")
	}
	return d
}

func buildSenderItems(senders []db.Sender, decisions map[string]db.SenderDecision, showDecided bool) []list.Item {
	var items []list.Item
	for _, s := range senders {
		dec := ""
		if d, ok := decisions[s.Address]; ok {
			dec = d.Decision
			if !showDecided && dec != "" {
				continue
			}
		}
		items = append(items, senderItem{sender: s, decision: dec})
	}
	return items
}

func countPending(senders []db.Sender, decisions map[string]db.SenderDecision) int {
	n := 0
	for _, s := range senders {
		if _, ok := decisions[s.Address]; !ok {
			n++
		}
	}
	return n
}

func renderStatusBar(m model) string {
	if m.status == "" {
		return ""
	}
	if m.statusErr {
		return statusErrStyle.Render(m.status)
	}
	return statusStyle.Render(m.status)
}

func renderKeys(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, keyStyle.Render("["+pairs[i]+"]")+" "+hintStyle.Render(pairs[i+1]))
	}
	return strings.Join(parts, "  ")
}

func actionStatusMsg(msg actionDoneMsg) string {
	switch msg.decision {
	case "deleted":
		return fmt.Sprintf("trashed %d emails from %s", msg.count, msg.senderAddress)
	case "keep":
		return fmt.Sprintf("keeping emails from %s", msg.senderAddress)
	case "skipped":
		return fmt.Sprintf("skipped %s", msg.senderAddress)
	case "unsubscribed":
		return fmt.Sprintf("unsubscribed from %s", msg.senderAddress)
	case "unsubscribed_deleted":
		return fmt.Sprintf("unsubscribed and trashed %d emails from %s", msg.count, msg.senderAddress)
	}
	return msg.decision
}

// ── async commands ────────────────────────────────────────────────────────────

func loadOverviewCmd(database *sql.DB, config *cfg.AppConfig) tea.Cmd {
	return func() tea.Msg {
		stats, err := db.GetStats(database)
		if err != nil {
			return overviewLoadedMsg{err: err}
		}
		var rows []aliasRow
		for _, p := range config.Providers {
			row := aliasRow{
				alias:    p.Alias,
				address:  p.Address,
				provider: string(p.Provider),
			}
			if s, ok := stats.Aliases[p.Alias]; ok {
				row.count = s.Count
				row.oldest = s.OldestEmail
				row.newest = s.NewestEmail
				row.lastSync = s.LastFetchedAt
			}
			if row.count > 0 {
				senders, _ := db.GetSendersBasic(database, p.Alias)
				row.total = len(senders)
				decs, _ := db.GetDecisions(database, p.Alias)
				row.decided = len(decs)
			}
			rows = append(rows, row)
		}
		return overviewLoadedMsg{rows: rows}
	}
}

func loadConnectorCmd(alias string) tea.Cmd {
	return func() tea.Msg {
		conn, _, err := factory.Connect(alias)
		return connectorReadyMsg{alias: alias, conn: conn, err: err}
	}
}

func loadSendersCmd(database *sql.DB, alias string) tea.Cmd {
	return func() tea.Msg {
		senders, err := db.GetSendersBasic(database, alias)
		if err != nil {
			return sendersLoadedMsg{alias: alias, err: err}
		}
		decisions, err := db.GetDecisions(database, alias)
		return sendersLoadedMsg{alias: alias, senders: senders, decisions: decisions, err: err}
	}
}

func loadDetailCmd(database *sql.DB, alias, senderAddress string) tea.Cmd {
	return func() tea.Msg {
		emails, err := db.GetEmailsBySender(database, alias, senderAddress)
		return detailLoadedMsg{emails: emails, err: err}
	}
}

func waitForSyncMsg(ch chan syncProgressMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func startSyncGoroutine(conn connector.WritableConnector, database *sql.DB, alias, providerStr string, preset syncPreset, ch chan syncProgressMsg) {
	go func() {
		ctx := context.Background()
		start, end := syncDateRange(preset)
		fetchCh := conn.Fetch(ctx, start, end)

		loaded, inserted := 0, 0
		for result := range fetchCh {
			if result.Err != nil {
				ch <- syncProgressMsg{err: result.Err, done: true}
				return
			}
			isNew, err := db.InsertEmail(database, alias, providerStr, result.Value)
			if err != nil {
				ch <- syncProgressMsg{err: err, done: true}
				return
			}
			loaded++
			if isNew {
				inserted++
			}
			if loaded%50 == 0 {
				ch <- syncProgressMsg{loaded: loaded, inserted: inserted}
			}
		}
		ch <- syncProgressMsg{loaded: loaded, inserted: inserted, done: true}
	}()
}

func syncDateRange(p syncPreset) (start, end time.Time) {
	end = time.Now()
	switch p {
	case preset90Days:
		start = end.AddDate(0, -3, 0)
	case presetYear:
		start = end.AddDate(-1, 0, 0)
	case presetAllTime:
		start = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return
}

func trashAllCmd(conn connector.WritableConnector, senderAddress string) tea.Cmd {
	return func() tea.Msg {
		count, err := conn.TrashAllFrom(context.Background(), senderAddress)
		return actionDoneMsg{
			senderAddress: senderAddress,
			decision:      "deleted",
			count:         count,
			err:           err,
		}
	}
}

func trashSelectedCmd(conn connector.WritableConnector, senderAddress string, ids []string) tea.Cmd {
	return func() tea.Msg {
		err := conn.TrashMessages(context.Background(), ids)
		count := 0
		if err == nil {
			count = len(ids)
		}
		return actionDoneMsg{
			senderAddress: senderAddress,
			decision:      "deleted",
			count:         count,
			err:           err,
		}
	}
}

func sendUnsubEmailCmd(conn connector.WritableConnector, senderAddress, to, subject string) tea.Cmd {
	return func() tea.Msg {
		if subject == "" {
			subject = "Unsubscribe"
		}
		err := conn.SendEmail(context.Background(), to, subject, "Please unsubscribe me from your mailing list.")
		decision := "unsubscribed"
		if err != nil {
			decision = ""
		}
		return actionDoneMsg{
			senderAddress: senderAddress,
			decision:      decision,
			err:           err,
		}
	}
}

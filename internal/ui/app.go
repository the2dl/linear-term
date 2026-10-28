package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/the2dl/linear-term/internal/data"
	"github.com/the2dl/linear-term/internal/linear"
	"github.com/the2dl/linear-term/internal/model"
)

type modalKind int

const (
	modalNone modalKind = iota
	modalDetail
	modalStatePicker
	modalPriorityPicker
	modalTokenInput
	modalHelp
)

type tickMsg time.Time

type boardFetchedMsg struct {
	resp *linear.BoardResponse
	err  error
}

type mutationDoneMsg struct {
	status string
	err    error
}

// Config holds startup options for the TUI.
type Config struct {
	Token        string
	TeamKey      string
	PollInterval time.Duration
	ForceDemo    bool
}

// AppModel is the Bubble Tea model for the Linear terminal Kanban board.
type AppModel struct {
	cfg           Config
	client        *linear.Client
	isDemo        bool
	workspaceName string

	columns   []model.ColumnMeta
	allIssues []model.Issue

	currentView model.BoardView
	hideEmpty   bool

	// Navigation state
	colIdx     int
	rowByCol   map[model.ColumnKind]int
	scrollByCol map[model.ColumnKind]int

	// Filter state
	filtering   bool
	filterQuery string

	// Modal state
	activeModal  modalKind
	detailScroll int
	pickerCursor int
	tokenInput   string

	// Live polling & animation state
	syncing       bool
	lastSync      time.Time
	secondsToPoll int
	simStep       int
	toastMsg      string
	toastUntil    time.Time
	errBanner     string

	width  int
	height int
}

// NewAppModel creates and initializes the board model.
func NewAppModel(cfg Config) AppModel {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 15 * time.Second
	}
	token := linear.ResolveToken(cfg.Token)
	isDemo := cfg.ForceDemo || token == ""

	var client *linear.Client
	if !isDemo {
		client = linear.NewClient(token, cfg.TeamKey)
	}

	var initialIssues []model.Issue
	if isDemo {
		initialIssues = data.SeedIssues()
	}

	m := AppModel{
		cfg:           cfg,
		client:        client,
		isDemo:        isDemo,
		workspaceName: "Acme",
		columns:       model.DefaultColumns(),
		allIssues:     initialIssues,
		currentView:   model.ViewDelivery,
		hideEmpty:     true,
		rowByCol:      make(map[model.ColumnKind]int),
		scrollByCol:   make(map[model.ColumnKind]int),
		syncing:       !isDemo,
		lastSync:      time.Now(),
		secondsToPoll: int(cfg.PollInterval.Seconds()),
		width:         140,
		height:        42,
	}
	return m
}

// SyncOnce performs a synchronous fetch (used by --preview).
func (m *AppModel) SyncOnce() {
	if m.isDemo || m.client == nil {
		return
	}
	msg := fetchBoardCmd(m.client)()
	updated, _ := m.Update(msg)
	if um, ok := updated.(AppModel); ok {
		*m = um
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{tickEverySecond()}
	if !m.isDemo && m.client != nil {
		cmds = append(cmds, fetchBoardCmd(m.client))
	}
	return tea.Batch(cmds...)
}

func tickEverySecond() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchBoardCmd(client *linear.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		resp, err := client.FetchBoard(ctx)
		return boardFetchedMsg{resp: resp, err: err}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		m.secondsToPoll--
		if m.secondsToPoll <= 0 {
			m.secondsToPoll = int(m.cfg.PollInterval.Seconds())
			if !m.isDemo && m.client != nil && !m.syncing {
				m.syncing = true
				return m, tea.Batch(tickEverySecond(), fetchBoardCmd(m.client))
			}
		}
		// Expire toast if past deadline
		if !m.toastUntil.IsZero() && now.After(m.toastUntil) {
			m.toastMsg = ""
		}
		return m, tickEverySecond()

	case boardFetchedMsg:
		m.syncing = false
		if msg.err != nil {
			m.errBanner = msg.err.Error()
			return m, nil
		}
		m.errBanner = ""
		m.lastSync = time.Now()
		if msg.resp != nil {
			if msg.resp.WorkspaceName != "" {
				m.workspaceName = msg.resp.WorkspaceName
			}
			m.columns = msg.resp.Columns
			m.mergeIncomingIssues(msg.resp.Issues)
		}
		m.clampSelection()
		return m, nil

	case mutationDoneMsg:
		if msg.err != nil {
			m.setToast("⚠ API error: " + msg.err.Error())
		} else if msg.status != "" {
			m.setToast(msg.status)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

func (m AppModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.activeModal == modalDetail {
		if msg.Button == tea.MouseButtonWheelDown {
			m.detailScroll += 2
		} else if msg.Button == tea.MouseButtonWheelUp && m.detailScroll > 0 {
			m.detailScroll = max(0, m.detailScroll-2)
		}
		return m, nil
	}
	if m.activeModal != modalNone {
		return m, nil
	}

	visCols := m.visibleColumns()
	if len(visCols) == 0 {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelDown:
		col := visCols[m.colIdx].Kind
		cards := m.issuesForColumn(col)
		if m.rowByCol[col] < len(cards)-1 {
			m.rowByCol[col]++
		}
	case tea.MouseButtonWheelUp:
		col := visCols[m.colIdx].Kind
		if m.rowByCol[col] > 0 {
			m.rowByCol[col]--
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		// Determine which column was clicked
		maxColsOnScreen := max(1, m.width/28)
		startCol := 0
		endCol := len(visCols)
		if len(visCols) > maxColsOnScreen {
			startCol = m.colIdx - maxColsOnScreen/2
			if startCol < 0 {
				startCol = 0
			}
			endCol = startCol + maxColsOnScreen
			if endCol > len(visCols) {
				endCol = len(visCols)
				startCol = max(0, endCol-maxColsOnScreen)
			}
		}
		windowCols := visCols[startCol:endCol]
		if len(windowCols) == 0 {
			return m, nil
		}
		colWidth := max(1, m.width/len(windowCols))
		clickedWinCol := min(len(windowCols)-1, max(0, msg.X/colWidth))
		m.colIdx = startCol + clickedWinCol

		// Determine if a card row was clicked (header + divider + col header + bar = 4 lines)
		if msg.Y >= 4 {
			colKind := visCols[m.colIdx].Kind
			cards := m.issuesForColumn(colKind)
			cardOffset := (msg.Y - 4) / 8
			boardHeight := max(10, m.height-4)
			cardsPerViewport := max(1, (boardHeight-2)/8)
			scrollOffset := computeScrollOffset(m.rowByCol[colKind], len(cards), cardsPerViewport)
			targetIdx := scrollOffset + cardOffset
			if targetIdx >= 0 && targetIdx < len(cards) {
				m.rowByCol[colKind] = targetIdx
			}
		}
	}
	return m, nil
}

func computeScrollOffset(selectedRow, totalCards, cardsPerViewport int) int {
	if totalCards <= cardsPerViewport {
		return 0
	}
	offset := 0
	if selectedRow >= cardsPerViewport {
		offset = selectedRow - cardsPerViewport + 1
	}
	if offset > totalCards-cardsPerViewport {
		offset = totalCards - cardsPerViewport
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

// mergeIncomingIssues diffs old vs new issues so changed or newly arrived issues flash on the board.
func (m *AppModel) mergeIncomingIssues(incoming []model.Issue) {
	oldByID := make(map[string]model.Issue, len(m.allIssues))
	for _, iss := range m.allIssues {
		oldByID[iss.Identifier] = iss
	}
	now := time.Now()
	updatedCount := 0
	for i := range incoming {
		if prev, ok := oldByID[incoming[i].Identifier]; ok {
			if prev.Column != incoming[i].Column {
				incoming[i].FlashUntil = now.Add(8 * time.Second)
				incoming[i].FlashTag = "MOVED"
				updatedCount++
			} else if !prev.UpdatedAt.Equal(incoming[i].UpdatedAt) || prev.Title != incoming[i].Title || prev.Priority != incoming[i].Priority {
				incoming[i].FlashUntil = now.Add(8 * time.Second)
				incoming[i].FlashTag = "UPDATED"
				updatedCount++
			} else if now.Before(prev.FlashUntil) {
				incoming[i].FlashUntil = prev.FlashUntil
				incoming[i].FlashTag = prev.FlashTag
			}
		} else if len(m.allIssues) > 0 {
			incoming[i].FlashUntil = now.Add(8 * time.Second)
			incoming[i].FlashTag = "NEW"
			updatedCount++
		}
	}
	m.allIssues = incoming
	if updatedCount > 0 {
		m.setToast(fmt.Sprintf("⚡ Synced %d live update(s) from Linear", updatedCount))
	}
}

func (m *AppModel) setToast(msg string) {
	m.toastMsg = msg
	m.toastUntil = time.Now().Add(5 * time.Second)
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// 1. Handle active filter input bar
	if m.filtering {
		switch key {
		case "esc", "enter":
			m.filtering = false
			m.clampSelection()
			return m, nil
		case "ctrl+u":
			m.filterQuery = ""
			m.clampSelection()
			return m, nil
		case "backspace":
			if len(m.filterQuery) > 0 {
				m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
				m.clampSelection()
			}
			return m, nil
		default:
			if len(msg.Runes) > 0 {
				m.filterQuery += string(msg.Runes)
				m.clampSelection()
			}
			return m, nil
		}
	}

	// 2. Handle active modal overlays
	if m.activeModal != modalNone {
		return m.handleModalKey(msg)
	}

	// 3. Board-level shortcuts
	visibleCols := m.visibleColumns()
	switch key {
	case "q":
		return m, tea.Quit

	case "h", "left":
		if m.colIdx > 0 {
			m.colIdx--
		}
		m.clampSelection()

	case "l", "right":
		if m.colIdx < len(visibleCols)-1 {
			m.colIdx++
		}
		m.clampSelection()

	case "j", "down":
		if len(visibleCols) > 0 {
			col := visibleCols[m.colIdx].Kind
			cards := m.issuesForColumn(col)
			if m.rowByCol[col] < len(cards)-1 {
				m.rowByCol[col]++
			}
		}

	case "k", "up":
		if len(visibleCols) > 0 {
			col := visibleCols[m.colIdx].Kind
			if m.rowByCol[col] > 0 {
				m.rowByCol[col]--
			}
		}

	case "g":
		if len(visibleCols) > 0 {
			col := visibleCols[m.colIdx].Kind
			m.rowByCol[col] = 0
		}

	case "G":
		if len(visibleCols) > 0 {
			col := visibleCols[m.colIdx].Kind
			cards := m.issuesForColumn(col)
			if len(cards) > 0 {
				m.rowByCol[col] = len(cards) - 1
			}
		}

	case "1":
		m.currentView = model.ViewActive
		m.clampSelection()
	case "2":
		m.currentView = model.ViewBacklog
		m.clampSelection()
	case "3":
		m.currentView = model.ViewAll
		m.clampSelection()
	case "4":
		m.currentView = model.ViewDelivery
		m.clampSelection()
	case "tab":
		for i, v := range model.AllBoardViews {
			if v == m.currentView {
				m.currentView = model.AllBoardViews[(i+1)%len(model.AllBoardViews)]
				break
			}
		}
		m.clampSelection()

	case "e":
		m.hideEmpty = !m.hideEmpty
		m.clampSelection()
		if m.hideEmpty {
			m.setToast("Collapsed empty (0-count) columns")
		} else {
			m.setToast("Showing all 7 columns (including empty)")
		}

	case "/":
		m.filtering = true

	case "esc":
		if m.filterQuery != "" {
			m.filterQuery = ""
			m.clampSelection()
			m.setToast("Cleared search filter")
		}

	case "enter", " ":
		if _, ok := m.selectedIssue(); ok {
			m.activeModal = modalDetail
			m.detailScroll = 0
		}

	case "s":
		if iss, ok := m.selectedIssue(); ok {
			m.activeModal = modalStatePicker
			m.pickerCursor = 0
			for i, k := range model.AllColumnKinds {
				if k == iss.Column {
					m.pickerCursor = i
					break
				}
			}
		}

	case "p":
		if iss, ok := m.selectedIssue(); ok {
			m.activeModal = modalPriorityPicker
			priorities := []model.Priority{
				model.PriorityUrgent,
				model.PriorityHigh,
				model.PriorityMedium,
				model.PriorityLow,
				model.PriorityNone,
			}
			m.pickerCursor = 0
			for i, pr := range priorities {
				if pr == iss.Priority {
					m.pickerCursor = i
					break
				}
			}
		}

	case "o":
		if iss, ok := m.selectedIssue(); ok && iss.URL != "" {
			openURL(iss.URL)
			m.setToast("🌐 Opened " + iss.Identifier + " in browser")
		}

	case "y":
		if iss, ok := m.selectedIssue(); ok {
			copyToClipboard(iss.Identifier)
			m.setToast("📋 Copied issue ID: " + iss.Identifier)
		}

	case "b":
		if iss, ok := m.selectedIssue(); ok {
			branch := iss.BranchName
			if branch == "" {
				branch = strings.ToLower(iss.Identifier)
			}
			copyToClipboard(branch)
			m.setToast("🌿 Copied git branch: " + branch)
		}

	case "r":
		m.secondsToPoll = int(m.cfg.PollInterval.Seconds())
		if !m.isDemo && m.client != nil {
			m.syncing = true
			m.setToast("↻ Refreshing from Linear API...")
			return m, fetchBoardCmd(m.client)
		}
		m.lastSync = time.Now()
		m.setToast("↻ Board refreshed")

	case "L":
		// Trigger live update simulation (great for previewing live transitions & card flashes)
		m.simStep++
		var status string
		m.allIssues, status = data.SimulateLiveEvent(m.allIssues, m.simStep)
		m.lastSync = time.Now()
		m.clampSelection()
		m.setToast(status)

	case "A":
		m.activeModal = modalTokenInput
		m.tokenInput = ""

	case "?":
		m.activeModal = modalHelp
	}

	return m, nil
}

func (m AppModel) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.activeModal {
	case modalHelp:
		if key == "esc" || key == "?" || key == "q" || key == "enter" {
			m.activeModal = modalNone
		}

	case modalTokenInput:
		switch key {
		case "esc":
			m.activeModal = modalNone
		case "ctrl+u":
			m.tokenInput = ""
		case "backspace":
			if len(m.tokenInput) > 0 {
				m.tokenInput = m.tokenInput[:len(m.tokenInput)-1]
			}
		case "enter":
			tok := strings.TrimSpace(m.tokenInput)
			m.activeModal = modalNone
			if tok == "" {
				m.isDemo = true
				m.client = nil
				m.allIssues = data.SeedIssues()
				m.setToast("Switched to Demo (Screenshot) mode")
				return m, nil
			}
			_ = linear.SaveToken(tok)
			m.client = linear.NewClient(tok, m.cfg.TeamKey)
			m.isDemo = false
			m.syncing = true
			m.setToast("🔑 Saved Linear API key — connecting...")
			return m, fetchBoardCmd(m.client)
		default:
			if len(msg.Runes) > 0 {
				m.tokenInput += string(msg.Runes)
			}
		}

	case modalDetail:
		iss, ok := m.selectedIssue()
		if !ok {
			m.activeModal = modalNone
			return m, nil
		}
		switch key {
		case "esc", "q", "enter":
			m.activeModal = modalNone
		case "j", "down":
			m.detailScroll++
		case "k", "up":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		case "s":
			m.activeModal = modalStatePicker
			for i, k := range model.AllColumnKinds {
				if k == iss.Column {
					m.pickerCursor = i
					break
				}
			}
		case "p":
			m.activeModal = modalPriorityPicker
			m.pickerCursor = 0
		case "o":
			if iss.URL != "" {
				openURL(iss.URL)
				m.setToast("🌐 Opened " + iss.Identifier + " in browser")
			}
		case "g":
			if iss.PRs != nil && len(iss.PRs.URLs) > 0 {
				openURL(iss.PRs.URLs[0])
				m.setToast("🌐 Opened PR in browser: " + iss.PRs.URLs[0])
			}
		case "y":
			copyToClipboard(iss.Identifier)
			m.setToast("📋 Copied issue ID: " + iss.Identifier)
		case "b":
			if iss.BranchName != "" {
				copyToClipboard(iss.BranchName)
				m.setToast("🌿 Copied branch: " + iss.BranchName)
			}
		}

	case modalStatePicker:
		switch key {
		case "esc", "q":
			m.activeModal = modalNone
		case "j", "down":
			if m.pickerCursor < len(model.AllColumnKinds)-1 {
				m.pickerCursor++
			}
		case "k", "up":
			if m.pickerCursor > 0 {
				m.pickerCursor--
			}
		case "1", "2", "3", "4", "5", "6", "7":
			idx := int(key[0] - '1')
			if idx >= 0 && idx < len(model.AllColumnKinds) {
				m.pickerCursor = idx
				return m.applyStateSelection(model.AllColumnKinds[idx])
			}
		case "enter":
			if m.pickerCursor >= 0 && m.pickerCursor < len(model.AllColumnKinds) {
				return m.applyStateSelection(model.AllColumnKinds[m.pickerCursor])
			}
		}

	case modalPriorityPicker:
		priorities := []model.Priority{
			model.PriorityUrgent,
			model.PriorityHigh,
			model.PriorityMedium,
			model.PriorityLow,
			model.PriorityNone,
		}
		switch key {
		case "esc", "q":
			m.activeModal = modalNone
		case "j", "down":
			if m.pickerCursor < len(priorities)-1 {
				m.pickerCursor++
			}
		case "k", "up":
			if m.pickerCursor > 0 {
				m.pickerCursor--
			}
		case "1", "2", "3", "4", "5":
			idx := int(key[0] - '1')
			if idx >= 0 && idx < len(priorities) {
				m.pickerCursor = idx
				return m.applyPrioritySelection(priorities[idx])
			}
		case "enter":
			if m.pickerCursor >= 0 && m.pickerCursor < len(priorities) {
				return m.applyPrioritySelection(priorities[m.pickerCursor])
			}
		}
	}

	return m, nil
}

func (m AppModel) applyStateSelection(targetCol model.ColumnKind) (tea.Model, tea.Cmd) {
	m.activeModal = modalNone
	iss, ok := m.selectedIssue()
	if !ok {
		return m, nil
	}
	now := time.Now()
	for i := range m.allIssues {
		if m.allIssues[i].Identifier == iss.Identifier {
			m.allIssues[i].Column = targetCol
			m.allIssues[i].StateName = string(targetCol)
			m.allIssues[i].UpdatedStr = "Just now"
			m.allIssues[i].UpdatedAt = now
			m.allIssues[i].FlashUntil = now.Add(7 * time.Second)
			m.allIssues[i].FlashTag = "MOVED"
			break
		}
	}
	// Focus target column so the user follows the moved card
	vis := m.visibleColumns()
	for i, c := range vis {
		if c.Kind == targetCol {
			m.colIdx = i
			m.rowByCol[targetCol] = max(0, len(m.issuesForColumn(targetCol))-1)
			break
		}
	}
	m.clampSelection()
	m.setToast(fmt.Sprintf("✔ Moved %s → %s", iss.Identifier, targetCol))

	if !m.isDemo && m.client != nil {
		var stateID string
		for _, c := range m.columns {
			if c.Kind == targetCol {
				stateID = c.StateID
				break
			}
		}
		if stateID != "" {
			client := m.client
			issueID := iss.ID
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err := client.UpdateIssueState(ctx, issueID, stateID)
				return mutationDoneMsg{err: err}
			}
		}
	}
	return m, nil
}

func (m AppModel) applyPrioritySelection(prio model.Priority) (tea.Model, tea.Cmd) {
	m.activeModal = modalNone
	iss, ok := m.selectedIssue()
	if !ok {
		return m, nil
	}
	now := time.Now()
	for i := range m.allIssues {
		if m.allIssues[i].Identifier == iss.Identifier {
			m.allIssues[i].Priority = prio
			m.allIssues[i].UpdatedStr = "Just now"
			m.allIssues[i].UpdatedAt = now
			m.allIssues[i].FlashUntil = now.Add(6 * time.Second)
			m.allIssues[i].FlashTag = "PRIORITY"
			break
		}
	}
	m.setToast(fmt.Sprintf("✔ Set %s priority → %s", iss.Identifier, prio.Label()))

	if !m.isDemo && m.client != nil {
		client := m.client
		issueID := iss.ID
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := client.UpdateIssuePriority(ctx, issueID, prio)
			return mutationDoneMsg{err: err}
		}
	}
	return m, nil
}

// visibleColumns returns the columns to display based on the active View and hideEmpty toggle.
func (m AppModel) visibleColumns() []model.ColumnMeta {
	var cols []model.ColumnMeta
	for _, col := range m.columns {
		switch m.currentView {
		case model.ViewBacklog:
			if col.Kind != model.ColBacklog && col.Kind != model.ColTodo {
				continue
			}
		case model.ViewActive:
			if col.Kind != model.ColTodo && col.Kind != model.ColInProgress && col.Kind != model.ColInReview && col.Kind != model.ColRollingOut {
				continue
			}
		}
		count := len(m.issuesForColumn(col.Kind))
		if m.hideEmpty && count == 0 {
			continue
		}
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		// Fallback if filter hid everything
		return m.columns
	}
	return cols
}

// issuesForColumn returns filtered issues belonging to the given column.
func (m AppModel) issuesForColumn(col model.ColumnKind) []model.Issue {
	q := strings.ToLower(strings.TrimSpace(m.filterQuery))
	var out []model.Issue
	for _, iss := range m.allIssues {
		if iss.Column != col {
			continue
		}
		if q != "" && !matchesFilter(iss, q) {
			continue
		}
		out = append(out, iss)
	}
	return out
}

func matchesFilter(iss model.Issue, q string) bool {
	if strings.Contains(strings.ToLower(iss.Identifier), q) ||
		strings.Contains(strings.ToLower(iss.Title), q) ||
		strings.Contains(strings.ToLower(iss.Assignee), q) {
		return true
	}
	if iss.Project != nil && strings.Contains(strings.ToLower(iss.Project.Name), q) {
		return true
	}
	for _, lbl := range iss.Labels {
		if strings.Contains(strings.ToLower(lbl.Name), q) {
			return true
		}
	}
	if iss.PRs != nil && strings.Contains(strings.ToLower(iss.PRs.Number), q) {
		return true
	}
	return false
}

func (m *AppModel) clampSelection() {
	vis := m.visibleColumns()
	if len(vis) == 0 {
		m.colIdx = 0
		return
	}
	if m.colIdx >= len(vis) {
		m.colIdx = len(vis) - 1
	}
	if m.colIdx < 0 {
		m.colIdx = 0
	}
	for _, c := range vis {
		cards := m.issuesForColumn(c.Kind)
		if len(cards) == 0 {
			m.rowByCol[c.Kind] = 0
		} else if m.rowByCol[c.Kind] >= len(cards) {
			m.rowByCol[c.Kind] = len(cards) - 1
		}
	}
}

func (m AppModel) selectedIssue() (model.Issue, bool) {
	vis := m.visibleColumns()
	if len(vis) == 0 || m.colIdx < 0 || m.colIdx >= len(vis) {
		return model.Issue{}, false
	}
	col := vis[m.colIdx].Kind
	cards := m.issuesForColumn(col)
	row := m.rowByCol[col]
	if row < 0 || row >= len(cards) {
		return model.Issue{}, false
	}
	return cards[row], true
}

// View renders the entire Linear terminal dashboard.
func (m AppModel) View() string {
	w := m.width
	h := m.height
	if w < 60 {
		w = 60
	}
	if h < 18 {
		h = 18
	}

	header := m.renderTopHeader(w)
	footer := m.renderFooter(w)

	boardHeight := h - lipgloss.Height(header) - lipgloss.Height(footer)
	if boardHeight < 10 {
		boardHeight = 10
	}

	board := m.renderColumnsBoard(w, boardHeight)
	screen := lipgloss.JoinVertical(lipgloss.Left, header, board, footer)

	// If a modal is open, render it centered on screen
	if m.activeModal != modalNone {
		var modalView string
		switch m.activeModal {
		case modalDetail:
			if iss, ok := m.selectedIssue(); ok {
				modalView = RenderDetailModal(iss, w, h, m.detailScroll)
			}
		case modalStatePicker:
			if iss, ok := m.selectedIssue(); ok {
				modalView = RenderStatePickerModal(iss, m.pickerCursor)
			}
		case modalPriorityPicker:
			if iss, ok := m.selectedIssue(); ok {
				modalView = RenderPriorityPickerModal(iss, m.pickerCursor)
			}
		case modalTokenInput:
			modalView = RenderTokenModal(m.tokenInput)
		case modalHelp:
			modalView = RenderHelpModal()
		}
		if modalView != "" {
			return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalView)
		}
	}

	return screen
}

func (m AppModel) renderTopHeader(width int) string {
	// Left side: Workspace Logo + Breadcrumb + View Tabs
	logo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0D0E11")).
		Background(ColorStatusInReview).
		Bold(true).
		Padding(0, 1).
		Render("▣")

	crumb := lipgloss.NewStyle().Foreground(ColorTextPrimary).Bold(true).Render(" "+m.workspaceName) +
		lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" › ") +
		lipgloss.NewStyle().Foreground(ColorTextSecondary).Bold(true).Render("Issues") +
		lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" ★   ")

	var tabs []string
	for i, v := range model.AllBoardViews {
		numStr := fmt.Sprintf("%d:", i+1)
		prefix := ""
		if v == model.ViewDelivery {
			prefix = "◔ "
		}
		label := numStr + prefix + string(v)
		if v == m.currentView {
			tab := lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				Background(ColorBgCardSelected).
				Bold(true).
				Padding(0, 1).
				Render(label)
			tabs = append(tabs, tab)
		} else {
			tab := lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Padding(0, 1).
				Render(label)
			tabs = append(tabs, tab)
		}
	}
	left := logo + crumb + strings.Join(tabs, " ")

	// Right side: Empty-column indicator, Filter badge, Live poll badge, Mode badge
	var rightParts []string

	hiddenCount := len(m.columns) - len(m.visibleColumns())
	if m.hideEmpty && hiddenCount > 0 {
		emptyBadge := lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Background(ColorBgPill).
			Padding(0, 1).
			Render(fmt.Sprintf("+%d empty [e]", hiddenCount))
		rightParts = append(rightParts, emptyBadge)
	}

	if m.filterQuery != "" {
		fBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0E11")).
			Background(ColorStatusInProgress).
			Bold(true).
			Padding(0, 1).
			Render("🔍 " + m.filterQuery)
		rightParts = append(rightParts, fBadge)
	}

	if m.syncing {
		syncBadge := lipgloss.NewStyle().
			Foreground(ColorStatusRollingOut).
			Bold(true).
			Render("⟳ syncing…")
		rightParts = append(rightParts, syncBadge)
	} else {
		pollTxt := lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Render(fmt.Sprintf("↻ %ds", m.secondsToPoll))
		rightParts = append(rightParts, pollTxt)
	}

	if m.isDemo {
		demoBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0E11")).
			Background(ColorStatusRollingOut).
			Bold(true).
			Padding(0, 1).
			Render("◈ DEMO • L:Sim A:API")
		rightParts = append(rightParts, demoBadge)
	} else {
		liveBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0E11")).
			Background(ColorStatusInReview).
			Bold(true).
			Padding(0, 1).
			Render("● LIVE API")
		rightParts = append(rightParts, liveBadge)
	}

	right := strings.Join(rightParts, "  ")
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	topLine := " " + left + strings.Repeat(" ", gap) + right + " "
	if lipgloss.Width(topLine) > width {
		topLine = ansi.Truncate(topLine, width, "…")
	}

	divider := lipgloss.NewStyle().Foreground(ColorBorderSubtle).Render(strings.Repeat("━", width))
	return topLine + "\n" + divider
}

func (m AppModel) renderColumnsBoard(width, height int) string {
	visCols := m.visibleColumns()
	if len(visCols) == 0 {
		return lipgloss.NewStyle().Foreground(ColorTextMuted).Padding(2, 4).Render("No columns visible.")
	}

	// Determine how many columns fit side by side cleanly (minimum 28 chars per column)
	maxColsOnScreen := max(1, width/28)
	startCol := 0
	endCol := len(visCols)
	if len(visCols) > maxColsOnScreen {
		// Center window around m.colIdx
		startCol = m.colIdx - maxColsOnScreen/2
		if startCol < 0 {
			startCol = 0
		}
		endCol = startCol + maxColsOnScreen
		if endCol > len(visCols) {
			endCol = len(visCols)
			startCol = max(0, endCol-maxColsOnScreen)
		}
	}
	windowCols := visCols[startCol:endCol]
	numCols := len(windowCols)
	colWidth := width / numCols
	extra := width % numCols

	now := time.Now()
	var renderedCols []string

	for i, colMeta := range windowCols {
		actualIdx := startCol + i
		isFocusedCol := actualIdx == m.colIdx
		cw := colWidth
		if i == numCols-1 {
			cw += extra
		}
		renderedCols = append(renderedCols, m.renderSingleColumn(colMeta, cw, height, isFocusedCol, now))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
}

func (m AppModel) renderSingleColumn(colMeta model.ColumnMeta, colWidth, colHeight int, isFocused bool, now time.Time) string {
	cards := m.issuesForColumn(colMeta.Kind)
	selectedRow := m.rowByCol[colMeta.Kind]

	icon, statusColor := ColumnIconAndColor(colMeta.Kind)

	// Column Header matching screenshot: `◔ In Progress  8   ··· +`
	iconRendered := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(icon)
	titleStyle := lipgloss.NewStyle().Foreground(ColorTextPrimary).Bold(true)
	if isFocused {
		titleStyle = titleStyle.Underline(true)
	}
	countStr := lipgloss.NewStyle().Foreground(ColorTextSecondary).Bold(true).Render(fmt.Sprintf(" %d", len(cards)))
	subStr := ""
	if colMeta.Subtitle != "" {
		subStr = lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" " + colMeta.Subtitle)
	}
	leftHdr := " " + iconRendered + " " + titleStyle.Render(colMeta.Title) + countStr + subStr

	rightDots := lipgloss.NewStyle().Foreground(ColorTextMuted).Render("··· + ")

	// Card height is fixed at 8 lines (6 content lines + 2 border lines)
	const cardHeight = 8
	availHeight := colHeight - 2 // subtract column header + accent bar
	cardsPerViewport := max(1, availHeight/cardHeight)

	// Calculate scroll offset so selectedRow is always visible in viewport
	scrollOffset := computeScrollOffset(selectedRow, len(cards), cardsPerViewport)

	if len(cards) > cardsPerViewport {
		above := scrollOffset
		below := max(0, len(cards)-(scrollOffset+cardsPerViewport))
		scrollBadge := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("↑%d ↓%d ", above, below))
		rightDots = scrollBadge + rightDots
	}

	gap := colWidth - lipgloss.Width(leftHdr) - lipgloss.Width(rightDots)
	if gap < 1 {
		gap = 1
	}
	hdrLine := leftHdr + strings.Repeat(" ", gap) + rightDots
	if lipgloss.Width(hdrLine) > colWidth {
		hdrLine = ansi.Truncate(hdrLine, colWidth, "")
	}

	barChar := "─"
	barColor := ColorBorderSubtle
	if isFocused {
		barChar = "━"
		barColor = statusColor
	}
	accentBar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat(barChar, max(1, colWidth-1))) + " "

	var colLines []string
	colLines = append(colLines, hdrLine, accentBar)

	if len(cards) == 0 {
		emptyBox := lipgloss.NewStyle().
			Width(colWidth - 3).
			Foreground(ColorTextMuted).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderMuted).
			Render("+")
		colLines = append(colLines, emptyBox)
	} else {
		endIdx := min(len(cards), scrollOffset+cardsPerViewport)
		for idx := scrollOffset; idx < endIdx; idx++ {
			isSelected := isFocused && idx == selectedRow
			cardStr := RenderCard(cards[idx], colWidth-1, isSelected, now)
			colLines = append(colLines, cardStr)
		}
	}

	joined := strings.Join(colLines, "\n")
	renderedLines := strings.Split(joined, "\n")
	for len(renderedLines) < colHeight {
		renderedLines = append(renderedLines, strings.Repeat(" ", colWidth))
	}
	if len(renderedLines) > colHeight {
		renderedLines = renderedLines[:colHeight]
	}

	return lipgloss.NewStyle().Width(colWidth).Render(strings.Join(renderedLines, "\n"))
}

func (m AppModel) renderFooter(width int) string {
	divider := lipgloss.NewStyle().Foreground(ColorBorderSubtle).Render(strings.Repeat("─", width))

	if m.filtering {
		prompt := lipgloss.NewStyle().Foreground(ColorStatusInProgress).Bold(true).Render(" 🔍 Filter: ") +
			lipgloss.NewStyle().Foreground(ColorTextPrimary).Bold(true).Render(m.filterQuery+"█") +
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render("   (match ID, title, project, label, PR)   [Enter] Keep   [Ctrl+U] Clear   [Esc] Close")
		return divider + "\n" + ansi.Truncate(prompt, width, "…")
	}

	var leftStatus string
	if m.errBanner != "" {
		leftStatus = lipgloss.NewStyle().Foreground(ColorUrgent).Bold(true).Render(" ⚠ " + m.errBanner)
	} else if m.toastMsg != "" {
		leftStatus = lipgloss.NewStyle().Foreground(ColorStatusInProgress).Bold(true).Render(" " + m.toastMsg)
	} else if iss, ok := m.selectedIssue(); ok {
		_, colColor := ColumnIconAndColor(iss.Column)
		idPart := lipgloss.NewStyle().Foreground(colColor).Bold(true).Render(" " + iss.Identifier)
		branchPart := ""
		if iss.BranchName != "" {
			branchPart = lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" ⎇ " + iss.BranchName)
		}
		leftStatus = idPart + branchPart
	} else {
		leftStatus = lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" Ready")
	}

	keyPill := func(k, label string) string {
		return lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true).Render(k) +
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render(":"+label)
	}
	hints := strings.Join([]string{
		keyPill("h/j/k/l", "nav"),
		keyPill("Enter", "details"),
		keyPill("s", "state"),
		keyPill("p", "prio"),
		keyPill("o", "open"),
		keyPill("y", "copy-id"),
		keyPill("/", "filter"),
		keyPill("e", "empty-cols"),
		keyPill("L", "sim-live"),
		keyPill("?", "help"),
		keyPill("q", "quit"),
	}, "  ") + " "

	gap := width - lipgloss.Width(leftStatus) - lipgloss.Width(hints)
	if gap < 1 {
		gap = 1
	}
	bar := leftStatus + strings.Repeat(" ", gap) + hints
	if lipgloss.Width(bar) > width {
		bar = ansi.Truncate(bar, width, "…")
	}
	return divider + "\n" + bar
}

func openURL(url string) {
	for _, bin := range []string{"xdg-open", "open"} {
		if path, err := exec.LookPath(bin); err == nil {
			cmd := exec.Command(path, url)
			_ = cmd.Start()
			return
		}
	}
}

func copyToClipboard(text string) {
	// Try wl-copy (Wayland/Hyprland) or xclip first
	if path, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command(path)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command(path, "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return
		}
	}
	// Fallback OSC52 escape sequence
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	_, _ = fmt.Fprintf(os.Stdout, "\x1b]52;c;%s\x07", b64)
}

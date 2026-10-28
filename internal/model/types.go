package model

import "time"

// Priority represents Linear's priority levels (0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low).
type Priority int

const (
	PriorityNone   Priority = 0
	PriorityUrgent Priority = 1
	PriorityHigh   Priority = 2
	PriorityMedium Priority = 3
	PriorityLow    Priority = 4
)

func (p Priority) Label() string {
	switch p {
	case PriorityUrgent:
		return "Urgent"
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Medium"
	case PriorityLow:
		return "Low"
	default:
		return "No priority"
	}
}

// ColumnKind represents the canonical workflow column on the Delivery board.
type ColumnKind string

const (
	ColBacklog    ColumnKind = "Backlog"
	ColTodo       ColumnKind = "Todo"
	ColInProgress ColumnKind = "In Progress"
	ColInReview   ColumnKind = "In Review"
	ColRollingOut ColumnKind = "Rolling Out"
	ColDone       ColumnKind = "Done"
	ColDuplicate  ColumnKind = "Duplicate"
)

// AllColumnKinds defines the canonical left-to-right column order from the screenshot.
var AllColumnKinds = []ColumnKind{
	ColBacklog,
	ColTodo,
	ColInProgress,
	ColInReview,
	ColRollingOut,
	ColDone,
	ColDuplicate,
}

// BoardView represents the top-bar view filter tabs.
type BoardView string

const (
	ViewActive   BoardView = "Active"
	ViewBacklog  BoardView = "Backlog"
	ViewAll      BoardView = "All issues"
	ViewDelivery BoardView = "Delivery"
)

var AllBoardViews = []BoardView{
	ViewActive,
	ViewBacklog,
	ViewAll,
	ViewDelivery,
}

// Label represents an issue label with its Linear hex color.
type Label struct {
	Name  string
	Color string // Hex color e.g. "#EB5757"
}

// Project represents a Linear project pill on an issue card.
type Project struct {
	Name  string
	Color string // Hex color for project icon
	Icon  string // Optional short glyph
}

// PRInfo represents linked GitHub pull request metadata on an issue card.
type PRInfo struct {
	Count  int      // Number of linked PRs
	Number string   // Specific PR number if single PR (e.g. "#8269", "#3192")
	URLs   []string // PR URLs
	Titles []string // PR titles
}

// SubIssueProgress represents sub-issue completion (e.g., 3/3).
type SubIssueProgress struct {
	Done  int
	Total int
	Items []SubIssueItem
}

type SubIssueItem struct {
	Identifier string
	Title      string
	Done       bool
}

// Comment represents a comment on an issue.
type Comment struct {
	Author    string
	Body      string
	CreatedAt string
}

// Issue represents a Linear issue card on the board.
type Issue struct {
	ID          string // Internal Linear UUID (for mutations)
	Identifier  string // e.g., "LIN-4759"
	Title       string
	Description string
	Column      ColumnKind
	StateID     string // Linear WorkflowState UUID
	StateName   string
	Priority    Priority
	Project     *Project
	Labels      []Label
	PRs         *PRInfo
	SubIssues   *SubIssueProgress
	Assignee    string
	CreatedStr  string // e.g. "Oct 28"
	UpdatedStr  string // e.g. "Oct 28"
	UpdatedAt   time.Time
	URL         string
	BranchName  string
	Comments    []Comment

	// Live UI state
	FlashUntil time.Time // Highlighted when recently updated/moved
	FlashTag   string    // e.g. "UPDATED" or "MOVED"
}

// Column Meta for rendering column headers.
type ColumnMeta struct {
	Kind     ColumnKind
	Title    string
	Subtitle string // e.g., "Past week" for Done and Duplicate
	StateID  string // Linear state ID if mapped from API
}

// DefaultColumns returns the 7 columns matching the Linear Delivery board screenshot.
func DefaultColumns() []ColumnMeta {
	return []ColumnMeta{
		{Kind: ColBacklog, Title: "Backlog"},
		{Kind: ColTodo, Title: "Todo"},
		{Kind: ColInProgress, Title: "In Progress"},
		{Kind: ColInReview, Title: "In Review"},
		{Kind: ColRollingOut, Title: "Rolling Out"},
		{Kind: ColDone, Title: "Done", Subtitle: "Past week"},
		{Kind: ColDuplicate, Title: "Duplicate", Subtitle: "Past week"},
	}
}

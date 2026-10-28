package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/the2dl/linear-term/internal/model"
)

// Linear Dark Palette
var (
	ColorBgCanvas       = lipgloss.Color("#0D0E11")
	ColorBgHeader       = lipgloss.Color("#13151A")
	ColorBgCard         = lipgloss.Color("#16181E")
	ColorBgCardSelected = lipgloss.Color("#1F2330")
	ColorBgPill         = lipgloss.Color("#20232B")
	ColorBorderSubtle   = lipgloss.Color("#262934")
	ColorBorderMuted    = lipgloss.Color("#1D2028")
	ColorBorderActive   = lipgloss.Color("#6E7BF2")
	ColorBorderFlash    = lipgloss.Color("#F2C94C")

	ColorTextPrimary   = lipgloss.Color("#ECEEF2")
	ColorTextSecondary = lipgloss.Color("#9BA1AE")
	ColorTextMuted     = lipgloss.Color("#636875")
	ColorAccentLinear  = lipgloss.Color("#5E6AD2")
	ColorAccentBright  = lipgloss.Color("#818CF8")

	// Workflow Column Status Colors (matching screenshot)
	ColorStatusBacklog    = lipgloss.Color("#6E7480")
	ColorStatusTodo       = lipgloss.Color("#D2D6DF")
	ColorStatusInProgress = lipgloss.Color("#F2C94C") // Warm gold/yellow
	ColorStatusInReview   = lipgloss.Color("#4CB782") // Emerald green
	ColorStatusRollingOut = lipgloss.Color("#38BDF8") // Electric blue/cyan
	ColorStatusDone       = lipgloss.Color("#6366F1") // Linear indigo
	ColorStatusDuplicate  = lipgloss.Color("#6E7480") // Muted grey

	// Priority Colors
	ColorUrgent = lipgloss.Color("#F97316") // Bright orange-red urgent box
	ColorHigh   = lipgloss.Color("#E2E8F0")
	ColorMedium = lipgloss.Color("#94A3B8")
	ColorLow    = lipgloss.Color("#64748B")
)

// ColumnIconAndColor returns the terminal glyph and color for each workflow column.
func ColumnIconAndColor(col model.ColumnKind) (string, lipgloss.Color) {
	switch col {
	case model.ColBacklog:
		return "◌", ColorStatusBacklog
	case model.ColTodo:
		return "○", ColorStatusTodo
	case model.ColInProgress:
		return "◔", ColorStatusInProgress
	case model.ColInReview:
		return "◕", ColorStatusInReview
	case model.ColRollingOut:
		return "➔", ColorStatusRollingOut
	case model.ColDone:
		return "✔", ColorStatusDone
	case model.ColDuplicate:
		return "⊘", ColorStatusDuplicate
	default:
		return "●", ColorTextSecondary
	}
}

// RenderPriorityBadge renders the Linear priority icon pill matching the screenshot:
// Urgent = orange [!] badge, High = ▂▄▆ bars, Medium = ▂▄░, Low = ▂░░, None = ---
func RenderPriorityBadge(p model.Priority) string {
	switch p {
	case model.PriorityUrgent:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111215")).
			Background(ColorUrgent).
			Bold(true).
			Padding(0, 1).
			Render("!")
	case model.PriorityHigh:
		return lipgloss.NewStyle().
			Foreground(ColorHigh).
			Background(ColorBgPill).
			Bold(true).
			Padding(0, 1).
			Render("▂▄▆")
	case model.PriorityMedium:
		return lipgloss.NewStyle().
			Foreground(ColorMedium).
			Background(ColorBgPill).
			Padding(0, 1).
			Render("▂▄░")
	case model.PriorityLow:
		return lipgloss.NewStyle().
			Foreground(ColorLow).
			Background(ColorBgPill).
			Padding(0, 1).
			Render("▂░░")
	default:
		return lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Background(ColorBgPill).
			Padding(0, 1).
			Render("···")
	}
}

// RenderProjectPill renders a project pill like `(❖ Lake engine)` or `(❖ Regions & failover)`.
func RenderProjectPill(proj *model.Project) string {
	if proj == nil || proj.Name == "" {
		return ""
	}
	iconColor := lipgloss.Color(proj.Color)
	if proj.Color == "" {
		iconColor = ColorAccentBright
	}
	icon := lipgloss.NewStyle().Foreground(iconColor).Background(ColorBgPill).Bold(true).Render("❖ ")
	text := lipgloss.NewStyle().Foreground(ColorTextSecondary).Background(ColorBgPill).Render(proj.Name)
	capLeft := lipgloss.NewStyle().Foreground(ColorBorderSubtle).Render("")
	capRight := lipgloss.NewStyle().Foreground(ColorBorderSubtle).Render("")
	_ = capLeft
	_ = capRight
	return lipgloss.NewStyle().
		Background(ColorBgPill).
		Padding(0, 1).
		Render(icon + text)
}

// RenderLabelPill renders a label pill with its colored dot (`● slow`, `● fast`, `● train`, `● Bug`, etc.).
func RenderLabelPill(lbl model.Label) string {
	dotColor := lipgloss.Color(lbl.Color)
	if lbl.Color == "" {
		dotColor = ColorAccentBright
	}
	dot := lipgloss.NewStyle().Foreground(dotColor).Background(ColorBgPill).Bold(true).Render("● ")
	name := lipgloss.NewStyle().Foreground(ColorTextSecondary).Background(ColorBgPill).Render(lbl.Name)
	return lipgloss.NewStyle().
		Background(ColorBgPill).
		Padding(0, 1).
		Render(dot + name)
}

// RenderPRPill renders the GitHub PR badge (`⇅ 4 PRs` or `⇅ #8269`).
func RenderPRPill(pr *model.PRInfo) string {
	if pr == nil || (pr.Count == 0 && pr.Number == "") {
		return ""
	}
	label := fmt.Sprintf("%d PRs", pr.Count)
	if pr.Number != "" {
		label = pr.Number
	} else if pr.Count == 1 {
		label = "1 PR"
	}
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color("#4CB782")).Background(ColorBgPill).Bold(true).Render("⇅ ")
	txt := lipgloss.NewStyle().Foreground(ColorTextSecondary).Background(ColorBgPill).Render(label)
	return lipgloss.NewStyle().
		Background(ColorBgPill).
		Padding(0, 1).
		Render(icon + txt)
}

// RenderSubIssuePill renders sub-issue completion (`◔ 3/3`).
func RenderSubIssuePill(sub *model.SubIssueProgress) string {
	if sub == nil || sub.Total == 0 {
		return ""
	}
	glyph := "◔ "
	col := ColorStatusInProgress
	if sub.Done >= sub.Total {
		glyph = "✔ "
		col = ColorStatusInReview
	}
	icon := lipgloss.NewStyle().Foreground(col).Background(ColorBgPill).Bold(true).Render(glyph)
	txt := lipgloss.NewStyle().Foreground(ColorTextSecondary).Background(ColorBgPill).Render(fmt.Sprintf("%d/%d", sub.Done, sub.Total))
	return lipgloss.NewStyle().
		Background(ColorBgPill).
		Padding(0, 1).
		Render(icon + txt)
}

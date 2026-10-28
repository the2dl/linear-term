package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/the2dl/linear-term/internal/model"
)

// RenderDetailModal renders the full issue details modal overlay.
func RenderDetailModal(issue model.Issue, width, height int, scroll int) string {
	modalW := min(width-6, 96)
	if modalW < 44 {
		modalW = width - 2
	}
	innerW := modalW - 6

	statusIcon, statusColor := ColumnIconAndColor(issue.Column)
	statusBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0D0E11")).
		Background(statusColor).
		Bold(true).
		Padding(0, 1).
		Render(statusIcon + " " + string(issue.Column))

	idBadge := lipgloss.NewStyle().
		Foreground(ColorAccentBright).
		Bold(true).
		Render(issue.Identifier)

	prioBadge := RenderPriorityBadge(issue.Priority) + " " +
		lipgloss.NewStyle().Foreground(ColorTextSecondary).Render(issue.Priority.Label())

	topBar := fmt.Sprintf("%s   %s   %s", idBadge, statusBadge, prioBadge)
	if issue.Assignee != "" {
		topBar += "   " + lipgloss.NewStyle().Foreground(ColorTextSecondary).Render("Assignee: @"+issue.Assignee)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorTextPrimary).
		Bold(true).
		Width(innerW)

	var metaPills []string
	if issue.Project != nil {
		metaPills = append(metaPills, RenderProjectPill(issue.Project))
	}
	for _, lbl := range issue.Labels {
		metaPills = append(metaPills, RenderLabelPill(lbl))
	}
	if issue.PRs != nil {
		metaPills = append(metaPills, RenderPRPill(issue.PRs))
	}
	if issue.SubIssues != nil {
		metaPills = append(metaPills, RenderSubIssuePill(issue.SubIssues))
	}
	pillsLine := strings.Join(metaPills, " ")

	divider := lipgloss.NewStyle().Foreground(ColorBorderSubtle).Render(strings.Repeat("─", innerW))

	var sections []string
	sections = append(sections, topBar)
	sections = append(sections, "")
	sections = append(sections, titleStyle.Render(issue.Title))
	if pillsLine != "" {
		sections = append(sections, "")
		sections = append(sections, pillsLine)
	}

	// Branch & Dates info
	metaInfo := fmt.Sprintf("Created %s  •  Updated %s", issue.CreatedStr, issue.UpdatedStr)
	if issue.BranchName != "" {
		metaInfo += fmt.Sprintf("  •  Branch: %s", lipgloss.NewStyle().Foreground(ColorStatusRollingOut).Render(issue.BranchName))
	}
	sections = append(sections, lipgloss.NewStyle().Foreground(ColorTextMuted).Render(metaInfo))
	sections = append(sections, divider)

	// Description
	secTitle := lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true)
	sections = append(sections, secTitle.Render("DESCRIPTION"))
	desc := strings.TrimSpace(issue.Description)
	if desc == "" {
		desc = "No description provided."
	}
	sections = append(sections, RenderMarkdown(desc, innerW, ""))

	// Linked PRs
	if issue.PRs != nil && (issue.PRs.Count > 0 || len(issue.PRs.URLs) > 0) {
		sections = append(sections, "")
		sections = append(sections, secTitle.Render(fmt.Sprintf("LINKED PULL REQUESTS (%d)", max(issue.PRs.Count, len(issue.PRs.URLs)))))
		if len(issue.PRs.URLs) > 0 {
			for _, u := range issue.PRs.URLs {
				sections = append(sections, lipgloss.NewStyle().Foreground(ColorStatusInReview).Render("  ⇅ ")+
					lipgloss.NewStyle().Foreground(ColorTextPrimary).Underline(true).Render(u))
			}
		} else if issue.PRs.Number != "" {
			sections = append(sections, lipgloss.NewStyle().Foreground(ColorStatusInReview).Render("  ⇅ PR "+issue.PRs.Number))
		}
	}

	// Sub-issues
	if issue.SubIssues != nil && len(issue.SubIssues.Items) > 0 {
		sections = append(sections, "")
		sections = append(sections, secTitle.Render(fmt.Sprintf("SUB-ISSUES (%d/%d)", issue.SubIssues.Done, issue.SubIssues.Total)))
		for _, item := range issue.SubIssues.Items {
			box := "[ ]"
			col := ColorTextSecondary
			if item.Done {
				box = "[✔]"
				col = ColorStatusInReview
			}
			sections = append(sections,
				lipgloss.NewStyle().Foreground(col).Bold(true).Render("  "+box+" "+item.Identifier+" ")+
					lipgloss.NewStyle().Foreground(ColorTextPrimary).Render(item.Title))
		}
	}

	// Comments
	if len(issue.Comments) > 0 {
		sections = append(sections, "")
		sections = append(sections, secTitle.Render(fmt.Sprintf("COMMENTS (%d)", len(issue.Comments))))
		for i, c := range issue.Comments {
			if i > 0 {
				sections = append(sections, "")
			}
			hdr := lipgloss.NewStyle().Foreground(ColorStatusInProgress).Bold(true).Render("  @"+c.Author) +
				lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" • "+c.CreatedAt)
			body := RenderMarkdown(strings.TrimSpace(c.Body), innerW, "    ")
			sections = append(sections, hdr, body)
		}
	}

	fullContent := strings.Join(sections, "\n")
	lines := strings.Split(fullContent, "\n")

	maxBodyLines := max(height-12, 10)
	if scroll > max(0, len(lines)-maxBodyLines) {
		scroll = max(0, len(lines)-maxBodyLines)
	}
	if scroll < 0 {
		scroll = 0
	}
	end := min(len(lines), scroll+maxBodyLines)
	visibleLines := lines[scroll:end]

	footerHints := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(
		"[j/k] Scroll  [s] State  [p] Priority  [o] Open Linear  [g] Open PR  [y] Copy ID  [b] Copy Branch  [Esc] Close",
	)
	if lipgloss.Width(footerHints) > innerW {
		footerHints = ansi.Truncate(footerHints, innerW, "…")
	}

	boxContent := strings.Join(visibleLines, "\n") + "\n" + divider + "\n" + footerHints

	modalStyle := lipgloss.NewStyle().
		Width(modalW - 2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccentBright)

	return modalStyle.Render(boxContent)
}

// RenderStatePickerModal renders the quick workflow state switcher modal (`s` key).
func RenderStatePickerModal(issue model.Issue, cursor int) string {
	var rows []string
	hdr := lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true).Render(
		fmt.Sprintf("Move %s to Workflow State", issue.Identifier),
	)
	rows = append(rows, hdr, "")

	for i, kind := range model.AllColumnKinds {
		icon, col := ColumnIconAndColor(kind)
		num := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("[%d] ", i+1))
		iconRendered := lipgloss.NewStyle().Foreground(col).Bold(true).Render(icon + " ")
		label := string(kind)
		if kind == issue.Column {
			label += " (current)"
		}
		if i == cursor {
			line := lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				Background(ColorBgCardSelected).
				Bold(true).
				Padding(0, 1).
				Render("▸ " + num + iconRendered + label)
			rows = append(rows, line)
		} else {
			line := lipgloss.NewStyle().
				Foreground(ColorTextSecondary).
				Padding(0, 1).
				Render("  " + num + iconRendered + label)
			rows = append(rows, line)
		}
	}

	rows = append(rows, "", lipgloss.NewStyle().Foreground(ColorTextMuted).Render("[j/k or 1-7] Select  [Enter] Apply  [Esc] Cancel"))

	return lipgloss.NewStyle().
		Width(46).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(ColorAccentBright).
		Render(strings.Join(rows, "\n"))
}

// RenderPriorityPickerModal renders the priority switcher modal (`p` key).
func RenderPriorityPickerModal(issue model.Issue, cursor int) string {
	priorities := []model.Priority{
		model.PriorityUrgent,
		model.PriorityHigh,
		model.PriorityMedium,
		model.PriorityLow,
		model.PriorityNone,
	}
	var rows []string
	hdr := lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true).Render(
		fmt.Sprintf("Set Priority for %s", issue.Identifier),
	)
	rows = append(rows, hdr, "")

	for i, p := range priorities {
		num := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("[%d] ", i+1))
		badge := RenderPriorityBadge(p)
		label := " " + p.Label()
		if p == issue.Priority {
			label += " (current)"
		}
		if i == cursor {
			line := lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				Background(ColorBgCardSelected).
				Bold(true).
				Padding(0, 1).
				Render("▸ " + num + badge + label)
			rows = append(rows, line)
		} else {
			line := lipgloss.NewStyle().
				Foreground(ColorTextSecondary).
				Padding(0, 1).
				Render("  " + num + badge + label)
			rows = append(rows, line)
		}
	}

	rows = append(rows, "", lipgloss.NewStyle().Foreground(ColorTextMuted).Render("[j/k or 1-5] Select  [Enter] Apply  [Esc] Cancel"))

	return lipgloss.NewStyle().
		Width(44).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(ColorAccentBright).
		Render(strings.Join(rows, "\n"))
}

// RenderTokenModal renders an interactive input prompt to configure LINEAR_API_KEY.
func RenderTokenModal(input string) string {
	masked := input
	if len(input) > 12 {
		masked = input[:8] + strings.Repeat("•", len(input)-12) + input[len(input)-4:]
	}
	var rows []string
	rows = append(rows,
		lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true).Render("🔑 Configure Linear API Key"),
		"",
		lipgloss.NewStyle().Foreground(ColorTextSecondary).Render("Paste your Personal API Key (lin_api_...) to switch from Demo to Live API:"),
		"",
		lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Background(ColorBgPill).
			Padding(0, 1).
			Width(54).
			Render(masked+"█"),
		"",
		lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Saves to ~/.config/linear-term/token (0600)"),
		lipgloss.NewStyle().Foreground(ColorTextMuted).Render("[Enter] Save & Connect   [Ctrl+U] Clear   [Esc] Cancel"),
	)
	return lipgloss.NewStyle().
		Width(62).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(ColorAccentBright).
		Render(strings.Join(rows, "\n"))
}

// RenderHelpModal renders the keyboard shortcut reference overlay (`?` key).
func RenderHelpModal() string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorStatusInProgress).Bold(true).Width(14)
	descStyle := lipgloss.NewStyle().Foreground(ColorTextSecondary)
	secStyle := lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true)

	row := func(k, d string) string {
		return "  " + keyStyle.Render(k) + descStyle.Render(d)
	}

	lines := []string{
		secStyle.Render("NAVIGATION & VIEWS"),
		row("h / l, ← / →", "Move across workflow columns"),
		row("j / k, ↓ / ↑", "Move up/down issue cards in column"),
		row("g / G", "Jump to top / bottom card in column"),
		row("1 / 2 / 3 / 4", "Switch view: Active, Backlog, All issues, Delivery"),
		row("e", "Toggle hiding empty (0-count) columns (Backlog, Todo, Duplicate)"),
		row("/", "Filter cards by ID, title, project, or label"),
		"",
		secStyle.Render("ISSUE ACTIONS"),
		row("Enter / Space", "Open full Issue Detail modal (description, PRs, sub-issues, comments)"),
		row("s", "Change workflow state (move card across columns)"),
		row("p", "Change issue priority (Urgent, High, Medium, Low)"),
		row("o", "Open selected issue in browser (linear.app)"),
		row("g (in modal)", "Open linked GitHub PR in browser"),
		row("y", "Copy issue ID (e.g. LIN-4759) to clipboard"),
		row("b", "Copy issue git branch name to clipboard"),
		"",
		secStyle.Render("LIVE UPDATES & API"),
		row("r", "Force immediate refresh from Linear API"),
		row("L", "Simulate live incoming event / card transition (Demo mode)"),
		row("A", "Configure / update Linear API Key (LINEAR_API_KEY)"),
		row("q / Ctrl+C", "Quit linear-term"),
		"",
		lipgloss.NewStyle().Foreground(ColorTextMuted).Render("Press [Esc] or [?] to close help"),
	}

	return lipgloss.NewStyle().
		Width(76).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(ColorAccentBright).
		Render(strings.Join(lines, "\n"))
}

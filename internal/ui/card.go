package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/the2dl/linear-term/internal/model"
)

// RenderCard renders a single Linear issue card of total width `cardWidth`.
func RenderCard(issue model.Issue, cardWidth int, selected bool, now time.Time) string {
	if cardWidth < 24 {
		cardWidth = 24
	}
	innerWidth := cardWidth - 4 // 2 border chars + 2 horizontal padding chars

	statusIcon, statusColor := ColumnIconAndColor(issue.Column)
	isFlashing := !issue.FlashUntil.IsZero() && now.Before(issue.FlashUntil)

	// 1. Header row: Identifier on left, Assignee / Flash tag on right
	idStyle := lipgloss.NewStyle().Foreground(ColorTextMuted).Bold(selected)
	if selected {
		idStyle = idStyle.Foreground(ColorTextSecondary)
	}
	leftID := idStyle.Render(issue.Identifier)

	var rightParts []string
	if isFlashing {
		tag := issue.FlashTag
		if tag == "" {
			tag = "LIVE"
		}
		flashBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111215")).
			Background(ColorBorderFlash).
			Bold(true).
			Padding(0, 1).
			Render("⚡ " + tag)
		rightParts = append(rightParts, flashBadge)
	}
	if issue.Assignee != "" {
		avatarGlyph := "●"
		avatarCol := lipgloss.Color("#F2994A")
		if strings.Contains(strings.ToLower(issue.Assignee), "codex") || strings.Contains(strings.ToLower(issue.Assignee), "agent") {
			avatarGlyph = "◈"
			avatarCol = lipgloss.Color("#38BDF8")
		}
		av := lipgloss.NewStyle().Foreground(avatarCol).Bold(true).Render(avatarGlyph) +
			lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" "+issue.Assignee)
		rightParts = append(rightParts, av)
	}
	rightHeader := strings.Join(rightParts, " ")
	gap := innerWidth - lipgloss.Width(leftID) - lipgloss.Width(rightHeader)
	if gap < 1 {
		gap = 1
	}
	headerLine := leftID + strings.Repeat(" ", gap) + rightHeader

	// 2. Title rows (2 lines, prefixed with status icon on line 1)
	iconStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusIcon)
	titleStyle := lipgloss.NewStyle().Foreground(ColorTextPrimary).Bold(selected)
	line1Budget := innerWidth - 2 // "◔ " takes 2 cells
	line2Budget := innerWidth

	l1, l2 := wrapTwoLines(issue.Title, line1Budget, line2Budget)
	titleLine1 := iconStyled + " " + titleStyle.Render(l1)
	titleLine2 := titleStyle.Render(l2)

	// 3. Pills (Priority, Project, Labels, PRs, Sub-issues) packed into rows that fit innerWidth
	var pills []string
	pills = append(pills, RenderPriorityBadge(issue.Priority))
	if issue.Project != nil {
		if p := RenderProjectPill(issue.Project); p != "" {
			pills = append(pills, p)
		}
	}
	for _, lbl := range issue.Labels {
		pills = append(pills, RenderLabelPill(lbl))
	}
	if issue.PRs != nil {
		if pr := RenderPRPill(issue.PRs); pr != "" {
			pills = append(pills, pr)
		}
	}
	if issue.SubIssues != nil {
		if sub := RenderSubIssuePill(issue.SubIssues); sub != "" {
			pills = append(pills, sub)
		}
	}

	pillLines := packPills(pills, innerWidth, 2)

	// 4. Footer row: Created Oct 28   Updated Oct 28
	createdTxt := "Created " + issue.CreatedStr
	updatedTxt := "Updated " + issue.UpdatedStr
	footerStyle := lipgloss.NewStyle().Foreground(ColorTextMuted)
	if isFlashing {
		updatedTxt = lipgloss.NewStyle().Foreground(ColorBorderFlash).Bold(true).Render("Updated " + issue.UpdatedStr)
	} else {
		updatedTxt = footerStyle.Render(updatedTxt)
	}
	footerLine := footerStyle.Render(createdTxt) + "  " + updatedTxt
	if lipgloss.Width(footerLine) > innerWidth {
		footerLine = ansi.Truncate(footerLine, innerWidth, "…")
	}

	// Assemble card body lines
	var bodyLines []string
	bodyLines = append(bodyLines, headerLine)
	bodyLines = append(bodyLines, titleLine1)
	if strings.TrimSpace(l2) != "" {
		bodyLines = append(bodyLines, titleLine2)
	} else {
		bodyLines = append(bodyLines, "")
	}
	bodyLines = append(bodyLines, pillLines...)
	bodyLines = append(bodyLines, footerLine)

	body := strings.Join(bodyLines, "\n")

	border := lipgloss.RoundedBorder()
	borderColor := ColorBorderSubtle
	if isFlashing {
		borderColor = ColorBorderFlash
		border = lipgloss.ThickBorder()
	} else if selected {
		borderColor = statusColor
		border = lipgloss.ThickBorder()
	}

	cardStyle := lipgloss.NewStyle().
		Width(cardWidth - 2). // lipgloss Width includes padding, excludes border
		Padding(0, 1).
		Border(border).
		BorderForeground(borderColor)

	return cardStyle.Render(body)
}

// wrapTwoLines splits text cleanly across at most two lines of given widths, truncating with "…" if needed.
func wrapTwoLines(text string, w1, w2 int) (string, string) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "", ""
	}
	var line1Words []string
	curLen := 0
	idx := 0
	for idx < len(words) {
		wLen := lipgloss.Width(words[idx])
		sep := 0
		if len(line1Words) > 0 {
			sep = 1
		}
		if curLen+sep+wLen <= w1 {
			line1Words = append(line1Words, words[idx])
			curLen += sep + wLen
			idx++
		} else {
			break
		}
	}
	if len(line1Words) == 0 {
		// First word itself is longer than w1
		first := ansi.Truncate(words[0], w1, "…")
		idx = 1
		rest := strings.Join(words[idx:], " ")
		return first, ansi.Truncate(rest, w2, "…")
	}
	line1 := strings.Join(line1Words, " ")
	if idx >= len(words) {
		return line1, ""
	}
	rest := strings.Join(words[idx:], " ")
	line2 := ansi.Truncate(rest, w2, "…")
	return line1, line2
}

// packPills packs rendered pill badges into at most `maxRows` lines fitting within `maxWidth`.
func packPills(pills []string, maxWidth int, maxRows int) []string {
	if len(pills) == 0 {
		return []string{""}
	}
	var rows []string
	var curRow []string
	curWidth := 0

	for i, p := range pills {
		pw := lipgloss.Width(p)
		sep := 0
		if len(curRow) > 0 {
			sep = 1
		}
		if curWidth+sep+pw <= maxWidth {
			curRow = append(curRow, p)
			curWidth += sep + pw
		} else {
			if len(curRow) > 0 {
				rows = append(rows, strings.Join(curRow, " "))
			}
			if len(rows) >= maxRows-1 {
				// Last allowed row: pack remaining as much as fits, then "+N" indicator
				var lastRow []string
				lw := 0
				remaining := len(pills) - i
				for j := i; j < len(pills); j++ {
					w := lipgloss.Width(pills[j])
					s := 0
					if len(lastRow) > 0 {
						s = 1
					}
					if lw+s+w <= maxWidth-4 {
						lastRow = append(lastRow, pills[j])
						lw += s + w
						remaining--
					} else {
						break
					}
				}
				if remaining > 0 {
					more := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(fmt.Sprintf("+%d", remaining))
					lastRow = append(lastRow, more)
				}
				rows = append(rows, strings.Join(lastRow, " "))
				curRow = nil
				break
			}
			curRow = []string{p}
			curWidth = pw
		}
	}
	if len(curRow) > 0 && len(rows) < maxRows {
		rows = append(rows, strings.Join(curRow, " "))
	}
	for len(rows) < maxRows {
		rows = append(rows, "")
	}
	return rows
}

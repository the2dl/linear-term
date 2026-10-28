package ui

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type spanKind int

const (
	spanPlain spanKind = iota
	spanBold
	spanItalic
	spanBoldItalic
	spanCode
	spanLink
	spanStrike
)

type inlineSpan struct {
	text string
	kind spanKind
}

var (
	orderedListRe = regexp.MustCompile(`^(\s*)(\d+[.)])\s+(.*)$`)
	ghNumberRe    = regexp.MustCompile(`/(?:issues|pull)/(\d+)(?:[/?#]|$)`)
)

// RenderMarkdown converts a markdown string into terminal-styled lines wrapped to width,
// with baseIndent prepended to every rendered line.
func RenderMarkdown(md string, width int, baseIndent string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	rawLines := strings.Split(md, "\n")

	indentW := ansi.StringWidth(baseIndent)
	availW := max(20, width-indentW)

	var out []string
	inCodeBlock := false

	plainColor := ColorTextSecondary
	bulletPlainColor := ColorTextPrimary

	for _, rawLine := range rawLines {
		trimmed := strings.TrimSpace(rawLine)

		// 1. Fenced code block toggle (``` or ~~~)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			bar := lipgloss.NewStyle().Foreground(ColorBorderActive).Render("│")
			firstPfx := baseIndent + "  " + bar + " "
			restPfx := baseIndent + "  " + bar + " "
			codeW := max(12, availW-4)
			codeStyle := lipgloss.NewStyle().Foreground(ColorStatusRollingOut)
			wrapped := wrapPlainText(rawLine, codeW)
			for i, wl := range wrapped {
				if i == 0 {
					out = append(out, firstPfx+codeStyle.Render(wl))
				} else {
					out = append(out, restPfx+codeStyle.Render(wl))
				}
			}
			continue
		}

		// 2. Blank line
		if trimmed == "" {
			out = append(out, "")
			continue
		}

		// 3. Horizontal rule (---, ***, ___)
		if isHorizontalRule(trimmed) {
			hr := lipgloss.NewStyle().Foreground(ColorBorderSubtle).Render(strings.Repeat("─", max(10, availW)))
			out = append(out, baseIndent+hr)
			continue
		}

		// 4. Headings (# through ######)
		if level, headingText, ok := parseHeading(trimmed); ok {
			spans := parseInlineSpans(headingText)
			var pfx string
			var hColor lipgloss.Color
			switch level {
			case 1:
				pfx = "◆ "
				hColor = ColorAccentBright
			case 2, 3:
				pfx = "▸ "
				hColor = ColorStatusInProgress
			default:
				pfx = "▹ "
				hColor = ColorTextPrimary
			}
			styledPfx := baseIndent + lipgloss.NewStyle().Foreground(hColor).Bold(true).Render(pfx)
			restPfx := baseIndent + strings.Repeat(" ", ansi.StringWidth(pfx))
			out = append(out, wrapAndRenderSpans(spans, max(12, availW-ansi.StringWidth(pfx)), styledPfx, restPfx, hColor, true)...)
			continue
		}

		// 5. Blockquotes (> ...)
		if strings.HasPrefix(trimmed, ">") {
			qText := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			bar := lipgloss.NewStyle().Foreground(ColorAccentLinear).Render("▎")
			pfx := baseIndent + "  " + bar + " "
			spans := parseInlineSpans(qText)
			out = append(out, wrapAndRenderSpans(spans, max(12, availW-4), pfx, pfx, ColorTextMuted, false)...)
			continue
		}

		// 6. Unordered lists (- , * , + ) and task lists (- [ ] , - [x] )
		if leadSpaces, itemText, isTask, taskDone, ok := parseBulletItem(rawLine); ok {
			nestLevel := leadSpaces / 2
			nestPad := strings.Repeat("  ", nestLevel)
			var bulletStr string
			var pfxWidth int
			if isTask {
				if taskDone {
					bulletStr = lipgloss.NewStyle().Foreground(ColorStatusInReview).Bold(true).Render("[✔]")
				} else {
					bulletStr = lipgloss.NewStyle().Foreground(ColorTextMuted).Bold(true).Render("[ ]")
				}
				pfxWidth = 2 + len(nestPad) + 4
			} else {
				glyph := "•"
				col := ColorAccentBright
				if nestLevel > 0 {
					glyph = "◦"
					col = ColorTextMuted
				}
				bulletStr = lipgloss.NewStyle().Foreground(col).Bold(true).Render(glyph)
				pfxWidth = 2 + len(nestPad) + 2
			}
			firstPfx := baseIndent + "  " + nestPad + bulletStr + " "
			restPfx := baseIndent + strings.Repeat(" ", pfxWidth)
			spans := parseInlineSpans(itemText)
			out = append(out, wrapAndRenderSpans(spans, max(12, availW-pfxWidth), firstPfx, restPfx, bulletPlainColor, false)...)
			continue
		}

		// 7. Ordered lists (1. , 2. , etc.)
		if m := orderedListRe.FindStringSubmatch(rawLine); len(m) == 4 {
			nestLevel := len(m[1]) / 2
			nestPad := strings.Repeat("  ", nestLevel)
			numToken := m[2]
			itemText := m[3]
			styledNum := lipgloss.NewStyle().Foreground(ColorAccentBright).Bold(true).Render(numToken)
			pfxWidth := 2 + len(nestPad) + len(numToken) + 1
			firstPfx := baseIndent + "  " + nestPad + styledNum + " "
			restPfx := baseIndent + strings.Repeat(" ", pfxWidth)
			spans := parseInlineSpans(itemText)
			out = append(out, wrapAndRenderSpans(spans, max(12, availW-pfxWidth), firstPfx, restPfx, bulletPlainColor, false)...)
			continue
		}

		// 8. Normal paragraph line
		spans := parseInlineSpans(trimmed)
		out = append(out, wrapAndRenderSpans(spans, availW, baseIndent, baseIndent, plainColor, false)...)
	}

	return strings.Join(out, "\n")
}

func isHorizontalRule(trimmed string) bool {
	if len(trimmed) < 3 {
		return false
	}
	ch := trimmed[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != ch && trimmed[i] != ' ' {
			return false
		}
	}
	return strings.Count(trimmed, string(ch)) >= 3
}

func parseHeading(trimmed string) (int, string, bool) {
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	text := strings.TrimSpace(trimmed[level+1:])
	if text == "" {
		return 0, "", false
	}
	return level, text, true
}

func parseBulletItem(rawLine string) (leadSpaces int, itemText string, isTask bool, taskDone bool, ok bool) {
	for leadSpaces < len(rawLine) && (rawLine[leadSpaces] == ' ' || rawLine[leadSpaces] == '\t') {
		if rawLine[leadSpaces] == '\t' {
			leadSpaces += 2
		} else {
			leadSpaces++
		}
	}
	rest := strings.TrimLeft(rawLine, " \t")
	if len(rest) < 2 {
		return 0, "", false, false, false
	}
	if (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		body := strings.TrimSpace(rest[2:])
		if strings.HasPrefix(body, "[ ] ") {
			return leadSpaces, strings.TrimSpace(body[4:]), true, false, true
		}
		if strings.HasPrefix(body, "[x] ") || strings.HasPrefix(body, "[X] ") {
			return leadSpaces, strings.TrimSpace(body[4:]), true, true, true
		}
		return leadSpaces, body, false, false, true
	}
	return 0, "", false, false, false
}

// parseInlineSpans parses inline markdown constructs:
// - backslash escapes (\~, \*, \_, etc.)
// - inline code (`...`)
// - links ([label](url))
// - bold+italic (***...***)
// - bold (**...** or __...__)
// - strikethrough (~~...~~)
// - italic (*...* or word-bounded _..._)
func parseInlineSpans(s string) []inlineSpan {
	var spans []inlineSpan
	var plainBuf strings.Builder

	flushPlain := func() {
		if plainBuf.Len() > 0 {
			spans = append(spans, inlineSpan{text: plainBuf.String(), kind: spanPlain})
			plainBuf.Reset()
		}
	}

	i := 0
	n := len(s)
	for i < n {
		// 1. Backslash escape: \~ \* \_ \` \[ \] \\ etc.
		if s[i] == '\\' && i+1 < n && isEscapablePunct(s[i+1]) {
			plainBuf.WriteByte(s[i+1])
			i += 2
			continue
		}

		// 2. Inline code (`...` or ``...``)
		if s[i] == '`' {
			tickCount := 1
			if i+1 < n && s[i+1] == '`' {
				tickCount = 2
			}
			delim := s[i : i+tickCount]
			endIdx := strings.Index(s[i+tickCount:], delim)
			if endIdx >= 0 {
				flushPlain()
				codeContent := s[i+tickCount : i+tickCount+endIdx]
				spans = append(spans, inlineSpan{text: codeContent, kind: spanCode})
				i += tickCount + endIdx + tickCount
				continue
			}
		}

		// 3. Markdown link: [label](url)
		if s[i] == '[' {
			if closeBracket := findClosingBracket(s, i); closeBracket > i {
				if closeBracket+1 < n && s[closeBracket+1] == '(' {
					if closeParen := findClosingParen(s, closeBracket+1); closeParen > closeBracket+1 {
						label := unescapeMarkdown(s[i+1 : closeBracket])
						url := strings.TrimSpace(s[closeBracket+2 : closeParen])
						display := formatLinkDisplay(label, url)
						flushPlain()
						spans = append(spans, inlineSpan{text: display, kind: spanLink})
						i = closeParen + 1
						continue
					}
				}
			}
		}

		// 4. Bold + Italic: ***...***
		if strings.HasPrefix(s[i:], "***") {
			if endIdx := strings.Index(s[i+3:], "***"); endIdx > 0 {
				flushPlain()
				content := unescapeMarkdown(s[i+3 : i+3+endIdx])
				spans = append(spans, inlineSpan{text: content, kind: spanBoldItalic})
				i += 3 + endIdx + 3
				continue
			}
		}

		// 5. Bold: **...** or __...__
		if strings.HasPrefix(s[i:], "**") || strings.HasPrefix(s[i:], "__") {
			delim := s[i : i+2]
			if endIdx := strings.Index(s[i+2:], delim); endIdx > 0 {
				flushPlain()
				content := unescapeMarkdown(s[i+2 : i+2+endIdx])
				spans = append(spans, inlineSpan{text: content, kind: spanBold})
				i += 2 + endIdx + 2
				continue
			}
		}

		// 6. Strikethrough: ~~...~~
		if strings.HasPrefix(s[i:], "~~") {
			if endIdx := strings.Index(s[i+2:], "~~"); endIdx > 0 {
				flushPlain()
				content := unescapeMarkdown(s[i+2 : i+2+endIdx])
				spans = append(spans, inlineSpan{text: content, kind: spanStrike})
				i += 2 + endIdx + 2
				continue
			}
		}

		// 7. Italic: *...* (not followed by space) or word-bounded _..._
		if s[i] == '*' && i+1 < n && s[i+1] != ' ' && s[i+1] != '*' {
			if endIdx := findClosingSingleDelim(s[i+1:], '*'); endIdx > 0 {
				flushPlain()
				content := unescapeMarkdown(s[i+1 : i+1+endIdx])
				spans = append(spans, inlineSpan{text: content, kind: spanItalic})
				i += 1 + endIdx + 1
				continue
			}
		}
		if s[i] == '_' && i+1 < n && s[i+1] != ' ' && s[i+1] != '_' && isWordBoundaryBefore(s, i) {
			if endIdx := findClosingUnderscore(s[i+1:]); endIdx > 0 {
				flushPlain()
				content := unescapeMarkdown(s[i+1 : i+1+endIdx])
				spans = append(spans, inlineSpan{text: content, kind: spanItalic})
				i += 1 + endIdx + 1
				continue
			}
		}

		plainBuf.WriteByte(s[i])
		i++
	}

	flushPlain()
	return spans
}

func isEscapablePunct(b byte) bool {
	switch b {
	case '\\', '`', '*', '_', '{', '}', '[', ']', '(', ')', '#', '+', '-', '.', '!', '~', '|', '>':
		return true
	default:
		return false
	}
}

func unescapeMarkdown(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && isEscapablePunct(s[i+1]) {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func findClosingBracket(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '[' {
			depth++
		} else if s[i] == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findClosingParen(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findClosingSingleDelim(sub string, delim byte) int {
	for i := 0; i < len(sub); i++ {
		if sub[i] == '\\' {
			i++
			continue
		}
		if sub[i] == delim && i > 0 && sub[i-1] != ' ' {
			return i
		}
	}
	return -1
}

func isWordBoundaryBefore(s string, idx int) bool {
	if idx == 0 {
		return true
	}
	prev := rune(s[idx-1])
	return !unicode.IsLetter(prev) && !unicode.IsDigit(prev)
}

func findClosingUnderscore(sub string) int {
	for i := 0; i < len(sub); i++ {
		if sub[i] == '\\' {
			i++
			continue
		}
		if sub[i] == '_' && i > 0 && sub[i-1] != ' ' {
			// Check word boundary after closing underscore
			if i+1 == len(sub) {
				return i
			}
			next := rune(sub[i+1])
			if !unicode.IsLetter(next) && !unicode.IsDigit(next) {
				return i
			}
		}
	}
	return -1
}

func formatLinkDisplay(label, url string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return url
	}
	// If URL points to a GitHub issue or PR (e.g. .../issues/8267) and label doesn't include #8267, append (#8267)
	if m := ghNumberRe.FindStringSubmatch(url); len(m) == 2 {
		num := m[1]
		if !strings.Contains(label, num) {
			return label + " (#" + num + ")"
		}
	}
	return label
}

type wordPiece struct {
	text string
	kind spanKind
}

func splitIntoPieces(spans []inlineSpan) []wordPiece {
	var pieces []wordPiece
	for _, sp := range spans {
		s := sp.text
		start := 0
		for i := 0; i < len(s); i++ {
			isSpace := s[i] == ' ' || s[i] == '\t'
			if isSpace {
				if i > start {
					pieces = append(pieces, wordPiece{text: s[start:i], kind: sp.kind})
				}
				j := i
				for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
					j++
				}
				pieces = append(pieces, wordPiece{text: " ", kind: sp.kind})
				i = j - 1
				start = j
			} else if (s[i] == '/' || s[i] == '-') && i > start && (i-start >= 4 || sp.kind == spanCode) {
				// Allow breaking long paths or hyphenated tokens after '/' or '-'
				pieces = append(pieces, wordPiece{text: s[start : i+1], kind: sp.kind})
				start = i + 1
			}
		}
		if start < len(s) {
			pieces = append(pieces, wordPiece{text: s[start:], kind: sp.kind})
		}
	}
	return pieces
}

func wrapAndRenderSpans(spans []inlineSpan, width int, firstPfx, restPfx string, baseColor lipgloss.Color, forceBold bool) []string {
	if len(spans) == 0 {
		return []string{firstPfx}
	}
	pieces := splitIntoPieces(spans)
	var lines [][]inlineSpan
	var curLine []inlineSpan
	curW := 0

	appendToCurLine := func(p wordPiece) {
		if len(curLine) > 0 && curLine[len(curLine)-1].kind == p.kind {
			curLine[len(curLine)-1].text += p.text
		} else {
			curLine = append(curLine, inlineSpan(p))
		}
		curW += ansi.StringWidth(p.text)
	}

	flushLine := func() {
		// Trim trailing space on flushed line
		if len(curLine) > 0 {
			last := &curLine[len(curLine)-1]
			last.text = strings.TrimRight(last.text, " ")
		}
		lines = append(lines, curLine)
		curLine = nil
		curW = 0
	}

	for _, p := range pieces {
		pw := ansi.StringWidth(p.text)
		if p.text == " " {
			if curW == 0 {
				continue
			}
			if curW+pw > width {
				flushLine()
				continue
			}
			appendToCurLine(p)
			continue
		}

		if curW > 0 && curW+pw > width {
			flushLine()
		}

		// If a single unbroken token still exceeds width, hard-wrap it
		for ansi.StringWidth(p.text) > width && width > 4 {
			chunk, rem := splitAtCellWidth(p.text, width-curW)
			if chunk == "" {
				break
			}
			appendToCurLine(wordPiece{text: chunk, kind: p.kind})
			flushLine()
			p.text = rem
		}
		if p.text != "" {
			appendToCurLine(p)
		}
	}
	if len(curLine) > 0 || len(lines) == 0 {
		flushLine()
	}

	out := make([]string, 0, len(lines))
	for idx, ln := range lines {
		var b strings.Builder
		if idx == 0 {
			b.WriteString(firstPfx)
		} else {
			b.WriteString(restPfx)
		}
		for _, sp := range ln {
			b.WriteString(renderInlineSpan(sp, baseColor, forceBold))
		}
		out = append(out, b.String())
	}
	return out
}

func splitAtCellWidth(s string, maxW int) (string, string) {
	if maxW <= 0 {
		return "", s
	}
	w := 0
	for i, r := range s {
		rw := ansi.StringWidth(string(r))
		if w+rw > maxW && i > 0 {
			return s[:i], s[i:]
		}
		w += rw
	}
	return s, ""
}

func renderInlineSpan(sp inlineSpan, baseColor lipgloss.Color, forceBold bool) string {
	switch sp.kind {
	case spanBold:
		return lipgloss.NewStyle().Foreground(ColorTextPrimary).Bold(true).Render(sp.text)
	case spanItalic:
		st := lipgloss.NewStyle().Foreground(ColorTextPrimary).Italic(true)
		if forceBold {
			st = st.Bold(true)
		}
		return st.Render(sp.text)
	case spanBoldItalic:
		return lipgloss.NewStyle().Foreground(ColorTextPrimary).Bold(true).Italic(true).Render(sp.text)
	case spanCode:
		return lipgloss.NewStyle().
			Foreground(ColorStatusRollingOut).
			Background(ColorBgPill).
			Render(sp.text)
	case spanLink:
		return lipgloss.NewStyle().
			Foreground(ColorAccentBright).
			Underline(true).
			Bold(forceBold).
			Render(sp.text)
	case spanStrike:
		return lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Strikethrough(true).
			Render(sp.text)
	default:
		st := lipgloss.NewStyle().Foreground(baseColor)
		if forceBold {
			st = st.Bold(true)
		}
		return st.Render(sp.text)
	}
}

func wrapPlainText(line string, width int) []string {
	if ansi.StringWidth(line) <= width {
		return []string{line}
	}
	var out []string
	rem := line
	for len(rem) > 0 {
		chunk, next := splitAtCellWidth(rem, width)
		out = append(out, chunk)
		rem = next
	}
	return out
}

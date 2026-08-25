package render

import (
	"image/color"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

// MDStyle selects face/color for a markdown line.
type MDStyle int

const (
	MDBody MDStyle = iota
	MDBold
	MDH1
	MDH2
	MDList
	MDStamp // COMM [T+mm:ss] prefix
)

// MDLine is one wrapped display line after markdown layout.
type MDLine struct {
	Text  string
	Style MDStyle
}

// ParseMarkdown lays out a lightweight markdown subset into wrapped lines:
//
//	# H1 / ## H2 — larger + amber
//	- / * / 1. lists — phosphor bullet indent
//	**bold** — brighter body (markers stripped)
//	blank lines — paragraph gaps (empty MDLine)
func ParseMarkdown(text string, maxW int, smallBody bool) []MDLine {
	if maxW < 40 {
		maxW = 40
	}
	var out []MDLine
	rawLines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, raw := range rawLines {
		line := strings.TrimRightFunc(raw, unicode.IsSpace)
		if strings.TrimSpace(line) == "" {
			out = append(out, MDLine{})
			continue
		}
		style := MDBody
		content := line
		switch {
		case strings.HasPrefix(content, "### "):
			style = MDH2
			content = strings.TrimSpace(content[4:])
		case strings.HasPrefix(content, "## "):
			style = MDH2
			content = strings.TrimSpace(content[3:])
		case strings.HasPrefix(content, "# "):
			style = MDH1
			content = strings.TrimSpace(content[2:])
		case isMDListLine(content):
			style = MDList
			content = stripMDListMarker(content)
		}
		content = stripInlineBold(content)
		prefix := ""
		cont := ""
		if style == MDList {
			prefix = "• "
			cont = "  "
		}
		wrapped := wrapMD(content, maxW, style, smallBody, prefix, cont)
		if len(wrapped) == 0 {
			out = append(out, MDLine{Style: style})
			continue
		}
		for _, w := range wrapped {
			out = append(out, MDLine{Text: w, Style: style})
		}
	}
	return out
}

func isMDListLine(s string) bool {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") {
		return true
	}
	// "1. " / "12. "
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i > 0 && i+2 <= len(s) && s[i] == '.' && s[i+1] == ' '
}

func stripMDListMarker(s string) string {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") {
		return strings.TrimSpace(s[2:])
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i+2 <= len(s) && s[i] == '.' && s[i+1] == ' ' {
		return strings.TrimSpace(s[i+2:])
	}
	return s
}

// stripInlineBold removes ** markers (text kept; drawn as body/phosphor).
func stripInlineBold(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func mdMeasure(s string, style MDStyle, smallBody bool) int {
	switch style {
	case MDH1:
		return LabelWidth(s) // medium+ visually; Draw uses Large
	case MDH2:
		return LabelWidth(s)
	default:
		if smallBody {
			return SmallLabelWidth(s)
		}
		return LabelWidth(s)
	}
}

func wrapMD(s string, maxW int, style MDStyle, smallBody bool, firstPrefix, contPrefix string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	words := strings.Fields(s)
	var out []string
	prefix := firstPrefix
	lineBody := ""
	flush := func() {
		if lineBody == "" {
			return
		}
		out = append(out, prefix+lineBody)
		prefix = contPrefix
		lineBody = ""
	}
	for _, w := range words {
		cand := w
		if lineBody != "" {
			cand = lineBody + " " + w
		}
		if mdMeasure(prefix+cand, style, smallBody) > maxW && lineBody != "" {
			flush()
			lineBody = w
			continue
		}
		lineBody = cand
	}
	flush()
	return out
}

func mdColor(style MDStyle) color.Color {
	switch style {
	case MDH1, MDH2, MDStamp:
		return ColorAmber
	case MDList, MDBold:
		return ColorPhosphor
	default:
		return ColorDim
	}
}

func mdLineHeight(style MDStyle, smallBody bool) int {
	switch style {
	case MDStamp:
		return 16
	case MDH1:
		if smallBody {
			return 18
		}
		return 28
	case MDH2:
		if smallBody {
			return 16
		}
		return 22
	default:
		if smallBody {
			return 14
		}
		return 18
	}
}

// DrawMarkdown draws ParseMarkdown output. Returns Y after the last line.
// maxY <= 0 means no vertical clip.
func DrawMarkdown(screen *ebiten.Image, text string, x, y, maxW, maxY int, smallBody bool) int {
	lines := ParseMarkdown(text, maxW, smallBody)
	yy := y
	for _, ln := range lines {
		h := mdLineHeight(ln.Style, smallBody)
		if ln.Text == "" {
			yy += h / 2
			if maxY > 0 && yy > maxY {
				break
			}
			continue
		}
		if maxY > 0 && yy+h > maxY {
			break
		}
		drawMDLine(screen, ln, x, yy, smallBody)
		yy += h
	}
	return yy
}

// MarkdownLinesForCOMM builds scrollable COMM lines: amber timestamp, then markdown body.
func MarkdownLinesForCOMM(timeLabel, body string, maxW int) []MDLine {
	var out []MDLine
	if timeLabel != "" {
		out = append(out, MDLine{Text: timeLabel, Style: MDStamp})
		out = append(out, MDLine{}) // keep T+ stamp clear of following H1 baseline
	}
	out = append(out, ParseMarkdown(body, maxW, true)...)
	out = append(out, MDLine{}) // gap between messages
	return out
}

func drawMDLine(screen *ebiten.Image, ln MDLine, x, y int, smallBody bool) {
	clr := mdColor(ln.Style)
	switch ln.Style {
	case MDH1:
		if smallBody {
			DrawText(screen, ln.Text, x, y, clr, false)
		} else {
			DrawTextLarge(screen, ln.Text, x, y, clr)
		}
	case MDH2, MDStamp:
		DrawText(screen, ln.Text, x, y, clr, false)
	default:
		DrawText(screen, ln.Text, x, y, clr, smallBody)
	}
}

// DrawMDLines draws pre-parsed lines (COMM scroll window).
func DrawMDLines(screen *ebiten.Image, lines []MDLine, start, end, x, y int, smallBody bool) int {
	yy := y
	for i := start; i < end && i < len(lines); i++ {
		ln := lines[i]
		h := mdLineHeight(ln.Style, smallBody)
		if ln.Text == "" {
			yy += h / 2
			continue
		}
		drawMDLine(screen, ln, x, yy, smallBody)
		yy += h
	}
	return yy
}

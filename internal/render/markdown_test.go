package render

import (
	"strings"
	"testing"
)

func TestParseMarkdownHeadersAndLists(t *testing.T) {
	src := "# Title\n\nIntro paragraph.\n\n## Orders\n\n- First item\n- Second item\n\n1. Numbered\n2. Next\n"
	lines := ParseMarkdown(src, 400, true)
	var styles []MDStyle
	var texts []string
	for _, ln := range lines {
		if ln.Text == "" {
			continue
		}
		styles = append(styles, ln.Style)
		texts = append(texts, ln.Text)
	}
	if len(styles) < 5 {
		t.Fatalf("got %d content lines: %#v", len(styles), texts)
	}
	if styles[0] != MDH1 || texts[0] != "Title" {
		t.Fatalf("h1: %v %q", styles[0], texts[0])
	}
	foundList := false
	for i, s := range styles {
		if s == MDList {
			foundList = true
			if !stringsHasPrefixBullet(texts[i]) {
				t.Fatalf("list line missing bullet: %q", texts[i])
			}
		}
	}
	if !foundList {
		t.Fatal("expected list lines")
	}
}

func stringsHasPrefixBullet(s string) bool {
	return strings.HasPrefix(s, "• ") || strings.HasPrefix(s, "  ")
}

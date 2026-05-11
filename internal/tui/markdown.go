package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

var nordMarkdownStyle = ansi.StyleConfig{
	Document: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#D8DEE9"), BackgroundColor: mdString("#2E3440")}},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: mdString("#D8DEE9")},
		Indent:         mdUint(1),
		IndentToken:    mdString("│ "),
	},
	Paragraph: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#D8DEE9")}},
	List:      ansi.StyleList{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#D8DEE9")}}, LevelIndent: 2},
	Heading: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		BlockSuffix: "\n",
		Color:       mdString("#8FBCBB"),
		Bold:        mdBool(true),
	}},
	H1:     ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#8FBCBB"), Bold: mdBool(true)}},
	H2:     ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#88C0D0"), Bold: mdBool(true)}},
	H3:     ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#81A1C1"), Bold: mdBool(true)}},
	H4:     ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#81A1C1"), Bold: mdBool(true)}},
	H5:     ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#81A1C1"), Bold: mdBool(true)}},
	H6:     ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: mdString("#81A1C1"), Bold: mdBool(true)}},
	Emph:   ansi.StylePrimitive{Color: mdString("#EBCB8B"), Italic: mdBool(true)},
	Strong: ansi.StylePrimitive{Color: mdString("#ECEFF4"), Bold: mdBool(true)},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: mdBool(true),
	},
	HorizontalRule: ansi.StylePrimitive{Color: mdString("#4C566A"), Format: "\n────────\n"},
	Item:           ansi.StylePrimitive{BlockPrefix: "• "},
	Enumeration:    ansi.StylePrimitive{BlockPrefix: ". "},
	Task:           ansi.StyleTask{Ticked: "[✓] ", Unticked: "[ ] "},
	Link:           ansi.StylePrimitive{Color: mdString("#88C0D0"), Underline: mdBool(true)},
	LinkText:       ansi.StylePrimitive{Color: mdString("#88C0D0"), Underline: mdBool(true)},
	Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix:          " ",
		Suffix:          " ",
		Color:           mdString("#EBCB8B"),
		BackgroundColor: mdString("#3B4252"),
	}},
	CodeBlock: ansi.StyleCodeBlock{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color:           mdString("#D8DEE9"),
		BackgroundColor: mdString("#3B4252"),
	}}},
	Table: ansi.StyleTable{
		CenterSeparator: mdString("┼"),
		ColumnSeparator: mdString("│"),
		RowSeparator:    mdString("─"),
	},
}

func mdString(value string) *string { return &value }
func mdBool(value bool) *bool       { return &value }
func mdUint(value uint) *uint       { return &value }

func renderMarkdownPreview(document string, showFrontmatter bool, width, limit int) []string {
	if limit <= 0 {
		return nil
	}
	frontmatter, body, hasFrontmatter := splitFrontmatter(document)
	if width < 20 {
		width = 20
	}

	lines := make([]string, 0, limit)
	if showFrontmatter && hasFrontmatter {
		lines = append(lines, "---")
		if strings.TrimSpace(frontmatter) != "" {
			lines = append(lines, strings.Split(frontmatter, "\n")...)
		}
		lines = append(lines, "---", "")
	}
	lines = append(lines, renderMarkdownLines(body, width)...)
	return limitPreviewLines(lines, limit)
}

func splitFrontmatter(document string) (frontmatter, body string, ok bool) {
	document = strings.ReplaceAll(document, "\r\n", "\n")
	lines := strings.Split(document, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", document, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), true
		}
	}
	return "", document, false
}

func renderMarkdownLines(markdown string, width int) []string {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return []string{"No content."}
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(nordMarkdownStyle),
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return fallbackMarkdownLines(markdown)
	}
	rendered, err := renderer.Render(markdown)
	if err != nil {
		return fallbackMarkdownLines(markdown)
	}
	lines := trimOuterBlankLines(strings.Split(strings.ReplaceAll(rendered, "\r\n", "\n"), "\n"))
	if len(lines) == 0 {
		return fallbackMarkdownLines(markdown)
	}
	return lines
}

func fallbackMarkdownLines(markdown string) []string {
	return strings.Split(strings.TrimSpace(markdown), "\n")
}

func trimOuterBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func limitPreviewLines(lines []string, limit int) []string {
	if len(lines) == 0 {
		return []string{""}
	}
	if len(lines) > limit {
		return lines[:limit]
	}
	return lines
}

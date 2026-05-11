package tui

import (
	"context"
	"testing"

	"feedctl/internal/app"
	"feedctl/internal/testutil"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIKeybindingsAndItemActions(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"), testutil.DefaultItem("guid-2", "Second", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}

	m := NewModel(a)
	if len(m.items) != 2 {
		t.Fatalf("items=%d", len(m.items))
	}
	m = updateKey(t, m, "j")
	if m.cursor != 1 {
		t.Fatalf("cursor after j=%d", m.cursor)
	}
	m = updateKey(t, m, "g")
	if m.cursor != 0 {
		t.Fatalf("cursor after g=%d", m.cursor)
	}
	m = updateKey(t, m, "?")
	if !m.help {
		t.Fatal("help not opened")
	}
	m = updateKey(t, m, "esc")
	if m.help {
		t.Fatal("help not closed")
	}
	firstID := m.items[m.cursor].ID
	m = updateKey(t, m, " ")
	item, err := a.Item(firstID)
	if err != nil {
		t.Fatal(err)
	}
	if item.ReadAt == "" {
		t.Fatal("space did not mark read")
	}
	m = updateKey(t, m, "s")
	item, _ = a.Item(firstID)
	if !item.Starred {
		t.Fatal("s did not star")
	}
	m = updateKey(t, m, "a")
	item, _ = a.Item(firstID)
	if item.ArchivedAt == "" {
		t.Fatal("a did not archive")
	}
	m = updateKey(t, m, "2")
	if m.section != 1 {
		t.Fatalf("section=%d", m.section)
	}
	m = updateKey(t, m, "A")
	if !m.showRemoved {
		t.Fatal("A did not toggle removed visibility")
	}
}

func updateKey(t *testing.T, m Model, key string) Model {
	t.Helper()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == " " {
		msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	if key == "esc" {
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	model, _ := m.Update(msg)
	return model.(Model)
}

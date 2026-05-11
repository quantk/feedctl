package tui

import (
	"context"

	"feedctl/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultViewWidth  = 100
	defaultViewHeight = 30
	minViewWidth      = 20
	minViewHeight     = 8
	wideViewWidth     = 92
	selectedMarker    = "┃"
	multiSelectMarker = "✓"
)

var sections = []string{"Inbox", "Unread", "Starred", "Sources", "Removed Sources", "All Items"}

func Run(ctx context.Context) error {
	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	m := NewModelWithContext(ctx, a)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func helpView() string {
	m := Model{width: defaultViewWidth, height: defaultViewHeight, help: true}
	return m.renderHelp(defaultViewWidth, defaultViewHeight)
}

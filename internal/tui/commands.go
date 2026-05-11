package tui

import (
	"context"
	"strings"
	"time"

	"feedctl/internal/app"
	feedSync "feedctl/internal/sync"

	tea "github.com/charmbracelet/bubbletea"
)

type syncMsg struct {
	results []feedSync.Result
}

type tickMsg struct{}

func syncCmd(ctx context.Context, a *app.App) tea.Cmd {
	return syncSourcesCmd(ctx, a, []string{""})
}

func syncSourcesCmd(ctx context.Context, a *app.App, sourceIDs []string) tea.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		results := make([]feedSync.Result, 0, len(sourceIDs))
		for _, sourceID := range sourceIDs {
			results = append(results, a.Sync(ctx, sourceID))
		}
		return syncMsg{results: results}
	}
}

func (m syncMsg) ok() bool {
	for _, result := range m.results {
		if !result.OK {
			return false
		}
	}
	return true
}

func (m syncMsg) errorText() string {
	var parts []string
	for _, result := range m.results {
		parts = append(parts, result.Errors...)
		for _, source := range result.Sources {
			parts = append(parts, source.Errors...)
		}
	}
	if len(parts) == 0 {
		return "unknown error"
	}
	return strings.Join(parts, "; ")
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

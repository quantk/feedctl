package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	nord0  = lipgloss.Color("#2E3440")
	nord1  = lipgloss.Color("#3B4252")
	nord2  = lipgloss.Color("#434C5E")
	nord3  = lipgloss.Color("#4C566A")
	nord4  = lipgloss.Color("#D8DEE9")
	nord5  = lipgloss.Color("#E5E9F0")
	nord6  = lipgloss.Color("#ECEFF4")
	nord7  = lipgloss.Color("#8FBCBB")
	nord8  = lipgloss.Color("#88C0D0")
	nord9  = lipgloss.Color("#81A1C1")
	nord10 = lipgloss.Color("#5E81AC")
	nord11 = lipgloss.Color("#BF616A")
	nord13 = lipgloss.Color("#EBCB8B")
	nord14 = lipgloss.Color("#A3BE8C")
	nord15 = lipgloss.Color("#B48EAD")

	appStyle = lipgloss.NewStyle().Foreground(nord4).Background(nord0)

	headerStyle = lipgloss.NewStyle().Foreground(nord6).Background(nord1).Bold(true)
	tabsStyle   = lipgloss.NewStyle().Foreground(nord9).Background(nord0)
	bodyStyle   = lipgloss.NewStyle().Foreground(nord4).Background(nord0)
	panelStyle  = lipgloss.NewStyle().Foreground(nord4).Background(nord0)
	mutedStyle  = lipgloss.NewStyle().Foreground(nord3).Background(nord0)
	borderStyle = lipgloss.NewStyle().Foreground(nord3).Background(nord0)

	selectedRowStyle    = lipgloss.NewStyle().Foreground(nord6).Background(nord2).Bold(true)
	selectedMarkerStyle = lipgloss.NewStyle().Foreground(nord8).Background(nord2).Bold(true)
	unreadStyle         = lipgloss.NewStyle().Foreground(nord8).Background(nord0).Bold(true)
	selectedUnreadStyle = lipgloss.NewStyle().Foreground(nord8).Background(nord2).Bold(true)
	readStyle           = lipgloss.NewStyle().Foreground(nord3).Background(nord0)
	selectedReadStyle   = lipgloss.NewStyle().Foreground(nord4).Background(nord2)
	starStyle           = lipgloss.NewStyle().Foreground(nord13).Background(nord0).Bold(true)
	selectedStarStyle   = lipgloss.NewStyle().Foreground(nord13).Background(nord2).Bold(true)
	successStyle        = lipgloss.NewStyle().Foreground(nord14).Background(nord0).Bold(true)
	warningStyle        = lipgloss.NewStyle().Foreground(nord13).Background(nord0).Bold(true)
	errorStyle          = lipgloss.NewStyle().Foreground(nord11).Background(nord0).Bold(true)
	accentStyle         = lipgloss.NewStyle().Foreground(nord7).Background(nord0).Bold(true)
	selectedAccentStyle = lipgloss.NewStyle().Foreground(nord7).Background(nord2).Bold(true)
	selectedMutedStyle  = lipgloss.NewStyle().Foreground(nord9).Background(nord2)
	helpTitleStyle      = lipgloss.NewStyle().Foreground(nord8).Background(nord0).Bold(true)
)

func isErrorMessage(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "failed") || strings.Contains(lower, "error")
}

func syncStateStyle(status string, selected ...bool) lipgloss.Style {
	background := nord0
	if len(selected) > 0 && selected[0] {
		background = nord2
	}
	s := strings.ToLower(status)
	switch {
	case strings.Contains(s, "syncing") || strings.Contains(s, "running"):
		return lipgloss.NewStyle().Foreground(nord13).Background(background).Bold(true)
	case strings.Contains(s, "fail") || strings.Contains(s, "error"):
		return lipgloss.NewStyle().Foreground(nord11).Background(background).Bold(true)
	case strings.Contains(s, "ok") || strings.Contains(s, "success"):
		return lipgloss.NewStyle().Foreground(nord14).Background(background).Bold(true)
	case strings.Contains(s, "removed") || strings.Contains(s, "disabled"):
		return lipgloss.NewStyle().Foreground(nord13).Background(background).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(nord3).Background(background)
	}
}

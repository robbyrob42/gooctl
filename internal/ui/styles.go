package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	Primary   = lipgloss.Color("#4285F4") // Google Blue
	Secondary = lipgloss.Color("#34A853") // Google Green
	Warning   = lipgloss.Color("#FBBC04") // Google Yellow
	Error     = lipgloss.Color("#EA4335") // Google Red
	Subtle    = lipgloss.Color("#6B7280")
	Highlight = lipgloss.Color("#F3F4F6")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Error)

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Warning)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(Subtle)

	LabelStyle = lipgloss.NewStyle().
			Bold(true).
			Width(12)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				PaddingLeft(2)
)

// Helper functions for common output patterns
func Success(msg string) string {
	return SuccessStyle.Render("✓ " + msg)
}

func Error_(msg string) string {
	return ErrorStyle.Render("✗ " + msg)
}

func Warning_(msg string) string {
	return WarningStyle.Render("⚠ " + msg)
}

func Info(msg string) string {
	return SubtleStyle.Render("ℹ " + msg)
}

func Title(msg string) string {
	return TitleStyle.Render(msg)
}

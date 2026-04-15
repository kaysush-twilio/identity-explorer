package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	warningColor   = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(1)

	// Input styles
	InputLabelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(0, 1)

	// Selection styles
	SelectedStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	UnselectedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(mutedColor)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	// Highlight style for selected items in lists
	HighlightStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
)

package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7C3AED")
	ColorSecondary = lipgloss.Color("#10B981")
	ColorWarning   = lipgloss.Color("#F59E0B")
	ColorDanger    = lipgloss.Color("#EF4444")
	ColorMuted     = lipgloss.Color("#6B7280")
	ColorText      = lipgloss.Color("#E5E7EB")
	ColorBg        = lipgloss.Color("#1F2937")
	ColorBgLight   = lipgloss.Color("#374151")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorText).
			Padding(0, 2).
			Bold(true)

	AppStyle = lipgloss.NewStyle().
			Background(ColorBg)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBgLight).
			Padding(1, 2)

	InputLabelStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	ButtonStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorText).
			Padding(0, 3).
			Bold(true)

	ButtonInactiveStyle = lipgloss.NewStyle().
				Background(ColorBgLight).
				Foreground(ColorMuted).
				Padding(0, 3)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBgLight).
			Foreground(ColorMuted).
			Padding(0, 1)

	LogStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	LogSuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	LogErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	StarStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ResultTableStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBgLight)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	ErrorBoxStyle = lipgloss.NewStyle().
			Background(ColorDanger).
			Foreground(ColorText).
			Padding(0, 2)

	ProgressBarEmpty = lipgloss.NewStyle().
			Background(ColorBgLight)

	ProgressBarFill = lipgloss.NewStyle().
			Background(ColorSecondary)

	NoticeStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	RadioSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	RadioUnselectedStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

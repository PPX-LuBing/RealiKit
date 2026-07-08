package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"reality-scanner-rev/internal/tui"
)

func main() {
	m := tui.NewModel()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

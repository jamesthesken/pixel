package tui

import (
	"pixel/tui/constants"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	m.viewport.Style = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("212")).Width(m.viewport.Width)

	// channel pane
	left := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("212")).Height(m.viewport.Height).Width(m.viewport.Width / 7).Padding(1).Render(m.list.View())
	right := m.viewport.View()
	bottomRight := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Height(1).Width(m.viewport.Width).BorderForeground(lipgloss.Color("212")).Padding(1).Render(m.textarea.View())

	// chat window and input
	rightPane := lipgloss.JoinVertical(lipgloss.Center, right, bottomRight)

	formatted := lipgloss.JoinHorizontal(lipgloss.Left, left, rightPane)

	return constants.DocStyle.Render(formatted)
}

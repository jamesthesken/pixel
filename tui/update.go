package tui

import (
	"fmt"
	"pixel/tui/constants"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		liCmd tea.Cmd
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	// Implement different tea messages sent by the client.
	// I.e., constants.Message message data sent in a Matrix room.
	case constants.Message:
		m.updateViewport()
	case constants.Room:
		m.list.InsertItem(-1, item(msg.Name))
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - msg.Width/4
		m.viewport.Height = msg.Height - msg.Height/4
		m.updateViewport()
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case key.Matches(msg, constants.Keymap.Tab):
			m.toggleBox()
		case key.Matches(msg, constants.Keymap.Enter):
			if m.textarea.Focused() {
				if m.textarea.Value() != "" {
					timeStamp := time.Now()
					m.messages = append(m.messages, m.senderStyle.Render(timeStamp.Format("3:04PM"+" < You > "))+m.textarea.Value())
					m.textarea.Reset()
					m.setContent(strings.Join(m.messages, "\n"))
					m.viewport.GotoBottom()
				}
			} else {
				m.updateViewport()
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.list, liCmd = m.list.Update(msg)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, liCmd)
}

// setContent performs text wrapping before setting the content in the viewport
func (m *Model) setContent(text string) {
	wrap := lipgloss.NewStyle().Width(m.viewport.Width)
	m.viewport.SetContent(wrap.Render(text))
}

// toggleBox toggles between the message entry and room list
func (m *Model) toggleBox() {
	m.mode = (m.mode + 1) % 2
	if m.mode == 0 {
		m.textarea.Blur()
	} else {
		m.textarea.Focus()
	}
}

// updateViewport sets the displayed messages based on which room is selected.
func (m *Model) updateViewport() {
	i, ok := m.list.SelectedItem().(item)
	if ok {
		roomId := m.rooms[string(i)]
		m.setContent(strings.Join(m.msgMap[roomId], "\n"))
		m.viewport.GotoBottom()
	}
}

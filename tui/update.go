package tui

import (
	"fmt"
	"pixel/tui/constants"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"maunium.net/go/mautrix/id"
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
		case key.Matches(msg, constants.Keymap.ListNav):
			if !m.textarea.Focused() {
				// prevents having to press the arrow key twice for updates
				if msg.String() == "down" {
					m.list.CursorDown()
				}
				if msg.String() == "up" {
					m.list.CursorUp()
				}
				m.updateViewport()
			}

		case key.Matches(msg, constants.Keymap.Enter):
			if m.textarea.Focused() {
				if m.textarea.Value() != "" {
					// get selected room from the list
					i, _ := m.list.SelectedItem().(item)
					// send text, there's other options too for later (i.e., images)
					m.client.SendText(id.RoomID(m.rooms[string(i)]), m.textarea.Value())
					m.textarea.SetValue("")
				}
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
		m.list.KeyMap.CursorUp.SetEnabled(true)
		m.list.KeyMap.CursorDown.SetEnabled(true)
	} else {
		m.textarea.Focus()
		m.list.KeyMap.CursorUp.SetEnabled(false)
		m.list.KeyMap.CursorDown.SetEnabled(false)
	}
}

// updateViewport sets the displayed messages based on which room is selected.
func (m *Model) updateViewport() {
	if len(m.list.Items()) > 0 {

		// get the current position of the cursuor and use that to access the message map
		idx := m.list.Cursor()
		rooms := m.list.Items()
		id := rooms[idx].(item)
		roomId := m.rooms[string(id)]

		// set content based on selected room
		m.setContent(strings.Join(m.msgMap[roomId], "\n"))
		m.viewport.GotoBottom()
	}
}

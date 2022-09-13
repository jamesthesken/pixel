package tui

import (
	"fmt"
	"pixel/tui/constants"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
	// Implement different tea messages sent by the clients.
	// i.e., Message interface
	case constants.Message:
		nick := fmt.Sprintf("< %s >", msg.Nick)
		m.messages = append(m.messages, m.senderStyle.Render(msg.Time)+" "+m.senderStyle.Render(nick+" ")+" "+msg.Content)
		m.setContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - msg.Width/4
		m.viewport.Height = msg.Height - msg.Height/4
		m.setContent(strings.Join(m.messages, "\n"))
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
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	channels := channelsToItems(m.channels)
	m.list.SetItems(channels)

	m.list, liCmd = m.list.Update(msg)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, liCmd)
}

func (m *Model) setContent(text string) {
	// Perform text wrapping before setting the content in the viewport
	wrap := lipgloss.NewStyle().Width(m.viewport.Width)
	m.viewport.SetContent(wrap.Render(text))
}

// toggleBox toggles between the message entry and channels list
func (m *Model) toggleBox() {
	m.mode = (m.mode + 1) % 2
	if m.mode == 0 {
		m.textarea.Blur()
	} else {
		m.textarea.Focus()
	}
}

func channelsToItems(channels []string) []list.Item {
	items := make([]list.Item, len(channels))

	for i := range channels {
		items[i] = item(channels[i])
		i++
	}

	return items
}

func removeChannel(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

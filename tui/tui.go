// Copyright (C) 2017 Tulir Asokan
// Copyright (C) 2018-2020 Luca Weiss
// Copyright (c) 2022 James Thesken
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package tui

/*
TODO:
- Set viewport content based on selected channel in list
- Set message target based on selected channel in list
*/

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"pixel/tui/constants"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var homeserver = flag.String("homeserver", "", "Matrix homeserver")
var username = flag.String("username", "", "Matrix username localpart")
var password = flag.String("password", "", "Matrix password")

func StartTea() {

	m := initialModel()

	p := *tea.NewProgram(m,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	// connect to Matrix
	flag.Parse()
	if *username == "" || *password == "" || *homeserver == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Println("Logging into", *homeserver, "as", *username)
	client, err := mautrix.NewClient(*homeserver, "", "")
	if err != nil {
		panic(err)
	}
	_, err = client.Login(&mautrix.ReqLogin{
		Type:             "m.login.password",
		Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: *username},
		Password:         *password,
		StoreCredentials: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Login successful")

	syncer := client.Syncer.(*mautrix.DefaultSyncer)

	// todo: improve timestamp parsing (it's not currently the user's local TZ), also understand when messages
	// are synced (right now it doesnt sync new messages in real-time)
	syncer.OnEventType(event.EventMessage, func(source mautrix.EventSource, evt *event.Event) {
		msgRcv := constants.Message{
			Time:    time.Unix(evt.Timestamp, 0).Format("3:04PM"),
			Nick:    evt.Sender.String(),
			Content: evt.Content.AsMessage().Body,
			Channel: evt.Content.AsRoomName().Name,
		}
		p.Send(msgRcv)
	})

	// todo: sync when a user leaves a room - right now it doesn't?
	syncer.OnEventType(event.StateRoomName, func(source mautrix.EventSource, evt *event.Event) {
		channel := constants.Channel{
			Name: evt.Content.AsRoomName().Name,
		}
		p.Send(channel)
	})

	go func() {
		for {
			if err := client.Sync(); err != nil {
				fmt.Println("Sync() returned ", err)
			}
		}
	}()

	m.client = client

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

type errMsg error
type item string
type itemDelegate struct{}
type mode int

func (i item) FilterValue() string { return "" }

const (
	nav mode = iota
	msgMode
)

type Model struct {
	mode        mode
	viewport    viewport.Model
	textarea    textarea.Model
	list        list.Model
	senderStyle lipgloss.Style
	notifStyle  lipgloss.Style
	client      *mautrix.Client
	messages    []string
	err         error
}

const listHeight = 14

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s", i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprintf(w, fn(str))
}

func initialModel() *Model {

	// text area
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(2)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	// viewport
	vp := viewport.New(5, 2)
	vp.SetContent(`Welcome to Matrix!`)
	// TODO - apply a new list of keybindings ...
	vp.KeyMap.PageDown.SetEnabled(false)
	vp.KeyMap.PageUp.SetEnabled(false)
	vp.KeyMap.HalfPageDown.SetEnabled(false)
	vp.KeyMap.HalfPageUp.SetEnabled(false)
	vp.KeyMap.Up.SetEnabled(false)
	vp.KeyMap.Down.SetEnabled(false)

	// channel list
	items := []list.Item{}
	const defaultWidth = 20

	list := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	list.SetFilteringEnabled(false)
	list.DisableQuitKeybindings()
	list.Title = "Rooms"
	list.SetStatusBarItemName("Room", "Rooms")

	return &Model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		notifStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		list:        list,
		err:         nil,
	}
}

// Init() is the first function called by BubbleTea.
func (m Model) Init() tea.Cmd {
	return nil
}

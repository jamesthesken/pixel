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

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"pixel/tui/constants"

	"github.com/caarlos0/env/v6"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type config struct {
	Homeserver string `env:"HOMESERVER"`
	Username   string `env:"USERNAME"`
	Password   string `env:"PASSWORD"`
}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

const listHeight = 14

func StartTea() {
	m := initialModel()

	m.msgMap = make(map[string][]string)
	m.rooms = make(map[string]string)

	p := *tea.NewProgram(m,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	// Read environment variables
	var cfg config
	cfgOpts := env.Options{Prefix: "PIXEL_"}
	if err := env.Parse(&cfg, cfgOpts); err != nil {
		fmt.Println("error reading environment:", err)
		os.Exit(1)
	}

	// Flags take priority over environment variables
	{
		var (
			homeserver = flag.String("homeserver", "", "Matrix homeserver")
			username   = flag.String("username", "", "Matrix username localpart")
			password   = flag.String("password", "", "Matrix password")
		)

		flag.Parse()
		if *homeserver != "" {
			cfg.Homeserver = *homeserver
		}
		if *username != "" {
			cfg.Username = *username
		}
		if *password != "" {
			cfg.Password = *password
		}
	}

	if cfg.Username == "" || cfg.Password == "" || cfg.Homeserver == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Connect to Matrix server
	fmt.Println("Logging into", cfg.Homeserver, "as", cfg.Username)
	client, err := mautrix.NewClient(cfg.Homeserver, "", "")
	if err != nil {
		panic(err)
	}
	_, err = client.Login(&mautrix.ReqLogin{
		Type:             "m.login.password",
		Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: cfg.Username},
		Password:         cfg.Password,
		StoreCredentials: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Login successful")

	syncer := client.Syncer.(*mautrix.DefaultSyncer)

	/*
		todo:
			improve timestamp parsing (it's not currently the user's local TZ)
			understand how message syncing works. currently on start up, sometimes the most recent message is not received.
	*/
	syncer.OnEventType(event.EventMessage, func(source mautrix.EventSource, evt *event.Event) {
		msgRcv := constants.Message{
			Time:    time.Unix(evt.Timestamp, 0).Format("3:04PM"),
			Nick:    evt.Sender.String(),
			Content: evt.Content.AsMessage().Body,
			Channel: string(evt.Content.AsCanonicalAlias().Alias),
		}

		// this could definitely be improved! this maps a slice of messages in a room to their respective RoomID.
		m.msgMap[evt.RoomID.String()] = append(m.msgMap[evt.RoomID.String()], m.senderStyle.Render(msgRcv.Time)+" "+m.senderStyle.Render(msgRcv.Nick+" ")+" "+msgRcv.Content)
		p.Send(msgRcv)
	})

	// todo: update the ui when a user leaves the room
	syncer.OnEventType(event.StateRoomName, func(source mautrix.EventSource, evt *event.Event) {
		channel := constants.Room{
			Name: evt.Content.AsRoomName().Name,
			Id:   evt.RoomID.String(),
		}
		m.rooms[evt.Content.AsRoomName().Name] = evt.RoomID.String()
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

type (
	errMsg       error
	item         string
	itemDelegate struct{}
	mode         int
)

func (i item) FilterValue() string { return "" }

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

// initialModel sets the defaults for each Bubble Tea component and constructs the model
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

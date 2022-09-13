package constants

/*
	messages.go implements message types that are sent to the TUI
	The TUI utilizes a switch statement
	based on which messages are received.
*/

type Message struct {
	Content      string
	Nick         string
	Channel      string
	Time         string
	Notification string
}

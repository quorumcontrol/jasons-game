package stats

import "github.com/AsynkronIT/protoactor-go/eventstream"

// UserMessage is for any stat that can be shown to a user
// this is what's called in the UI
type UserMessage interface {
	Humanize() string
}

// Stream is a global eventstream to use for app stats -
// it will probably go away
var Stream = new(eventstream.EventStream)

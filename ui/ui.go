package ui

import (
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// the UIs event for when a new command has been typed
type EventCommand string

func (ec EventCommand) String() string {
	return string(ec)
}

// JasonsGameUI is the basic 3 box UI for a text game
type JasonsGameUI struct {
	app         *tview.Application
	inputField  *textGameInputField
	txtField    *tview.TextView
	EventStream *eventstream.EventStream
}

func (jsgui *JasonsGameUI) Write(msg string) (n int, err error) {
	return jsgui.txtField.Write([]byte(msg + "\n"))
}

func (jsgui *JasonsGameUI) Run() error {
	return jsgui.app.Run()
}

type Event struct {
	EventType uint8
	Payload   interface{}
}

type textGameInputField struct {
	inputField  *tview.InputField
	inputBuffer []byte
}

func newTextGameInputField(stream *eventstream.EventStream) *textGameInputField {
	var buffer []byte

	inputField := tview.NewInputField().
		SetLabel("What to do?").
		SetFieldWidth(128)

	inputField.SetDoneFunc(func(key tcell.Key) {
		stream.Publish(EventCommand(inputField.GetText()))
		inputField.SetText("")
	})

	return &textGameInputField{
		inputField:  inputField,
		inputBuffer: buffer,
	}
}

// run the UI
func New() (*JasonsGameUI, error) {
	app := tview.NewApplication()
	events := &eventstream.EventStream{}

	mainFlex := tview.NewFlex()
	todoReplaceBox := tview.NewBox().SetBorder(true).SetTitle("Your Stuff").SetBorderPadding(10, 5, 5, 5)

	txtFlex := tview.NewFlex()
	txtFlex.SetDirection(tview.FlexRow)

	txt := tview.NewTextView()
	txt.SetBorder(true).SetTitle("Hello, dog!")
	txt.SetChangedFunc(func() {
		app.Draw()
	})

	inptField := newTextGameInputField(events)

	txtFlex.AddItem(txt, 0, 90, false)
	txtFlex.AddItem(inptField.inputField, 10, 10, true)

	mainFlex.AddItem(txtFlex, 0, 75, true)
	mainFlex.AddItem(todoReplaceBox, 0, 25, false)
	app.SetRoot(mainFlex, true)
	return &JasonsGameUI{
		app:         app,
		inputField:  inptField,
		txtField:    txt,
		EventStream: events,
	}, nil
}

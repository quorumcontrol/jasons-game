package ui

import (
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	eventTypeCommand uint8 = iota
)

// JasonsGameUI is the basic 3 box UI for a text game
type JasonsGameUI struct {
	app         *tview.Application
	inputField  *textGameInputField
	txtField    *tview.TextView
	EventStream *eventstream.EventStream
}

func (jsgui *JasonsGameUI) Write(bits []byte) (n int, err error) {
	return jsgui.txtField.Write(bits)
}

func (jsgui *JasonsGameUI) Run() error {
	return jsgui.app.Run()
}

type event struct {
	eventType uint8
	payload   interface{}
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
		stream.Publish(&event{
			eventType: eventTypeCommand,
			payload:   inputField.GetText(),
		})
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
	txt.Write([]byte("hi hi\n"))

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

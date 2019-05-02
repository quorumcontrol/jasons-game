package ui

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/gdamore/tcell"
	"github.com/quorumcontrol/jasons-game/navigator"
	"github.com/quorumcontrol/jasons-game/stats"
	"github.com/rivo/tview"
)

const (
	gameOutputLabel   = "locationOutput"
	commandInputField = "commandInputField"
	gameStatsOut      = "gameStatsOut"
)

type Exit struct{}

// UserInput is the event outputted when a user interacts with the UI
type UserInput struct {
	Message string
}

// MessageToUser is used to communicate directly to the game output
// outside of normal messages like Location
type MessageToUser struct {
	Message string
}

// Subscribe is used to get and receive UI events
type Subscribe struct{}

type elementMap map[string]tview.Primitive

type jasonsGameUI struct {
	app               *tview.Application
	elements          elementMap
	subscriber        *actor.PID
	statsSubscription *eventstream.Subscription
}

// NewUIProps returns the actor props necessary to spin up a new UI
func NewUIProps() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &jasonsGameUI{
			elements: make(elementMap),
		}
	})
}

func (jsgui *jasonsGameUI) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		if err := jsgui.initialize(actorCtx); err != nil {
			panic(fmt.Errorf("error initializing: %v", err))
		}
	case *actor.Stopping:
		stats.Stream.Unsubscribe(jsgui.statsSubscription)
		jsgui.app.Stop()
	case *Exit:
		actorCtx.Self().Poison()
	case *Subscribe:
		jsgui.subscriber = actorCtx.Sender()
	case *navigator.Location:
		jsgui.handleLocation(msg)
	case *MessageToUser:
		jsgui.handleMessageToUser(msg)
	case stats.UserMessage:
		jsgui.handleStatsMessage(msg)
	}
}

func (jsgui *jasonsGameUI) handleStatsMessage(msg stats.UserMessage) {
	jsgui.elements[gameStatsOut].(*tview.TextView).Write([]byte(msg.Humanize() + "\n"))
}

func (jsgui *jasonsGameUI) handleLocation(loc *navigator.Location) {
	jsgui.elements[gameOutputLabel].(*tview.TextView).Write([]byte(loc.Did + fmt.Sprintf(" (%d,%d)\n", loc.X, loc.Y) + loc.Description + "\n"))
}

func (jsgui *jasonsGameUI) handleMessageToUser(msg *MessageToUser) {
	jsgui.elements[gameOutputLabel].(*tview.TextView).Write([]byte(msg.Message + "\n"))
}

// run the UI
func (jsgui *jasonsGameUI) initialize(actorCtx actor.Context) error {
	app := tview.NewApplication()

	mainFlex := tview.NewFlex()

	txtFlex := tview.NewFlex()
	txtFlex.SetDirection(tview.FlexRow)

	txt := tview.NewTextView()
	txt.SetBorder(true).SetTitle("Hello, dog!")
	txt.SetChangedFunc(func() {
		app.Draw()
	})

	inputField := tview.NewInputField().
		SetLabel("What to do?").
		SetFieldWidth(128)

	inputField.SetDoneFunc(func(key tcell.Key) {
		actor.EmptyRootContext.Send(jsgui.subscriber, &UserInput{
			Message: inputField.GetText(),
		})
		inputField.SetText("")
	})

	txtFlex.AddItem(txt, 0, 90, false)
	txtFlex.AddItem(inputField, 10, 10, true)

	mainFlex.AddItem(txtFlex, 0, 75, true)

	statTxt := tview.NewTextView()
	statTxt.SetChangedFunc(func() {
		statTxt.ScrollToEnd()
		app.Draw()
	})
	statTxt.SetBorder(true).SetTitle("Stats")

	mainFlex.AddItem(statTxt, 0, 25, false)
	app.SetRoot(mainFlex, true)
	jsgui.app = app

	jsgui.elements[commandInputField] = inputField
	jsgui.elements[gameOutputLabel] = txt
	jsgui.elements[gameStatsOut] = statTxt

	self := actorCtx.Self()
	jsgui.statsSubscription = stats.Stream.Subscribe(func(evt interface{}) {
		msg, ok := evt.(stats.UserMessage)
		if ok {
			actor.EmptyRootContext.Send(self, msg)
		}
	})

	go app.Run()
	return nil
}

package ui

import "github.com/AsynkronIT/protoactor-go/actor"

type simulatedUI struct {
	subscriber *actor.PID
	events     []interface{}
}

type GetEventsFromSimulator struct{}

// NewUIPNewSimulatedUIProps returns an actor that just stores
// the latest events sent to it and can retreive them, you can generate
// UI events that will also be sent to the subscriber
func NewSimulatedUIProps() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &simulatedUI{}
	})
}

func (sui *simulatedUI) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started, *actor.Stopping, *actor.Stopped:
		// do nothing
	case *Subscribe:
		sui.subscriber = actorCtx.Sender()
	case *UserInput:
		actorCtx.Send(sui.subscriber, msg)
	case *GetEventsFromSimulator:
		actorCtx.Respond(sui.events)
	default:
		sui.events = append(sui.events, msg)
	}
}

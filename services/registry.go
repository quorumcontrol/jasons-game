package services

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/jasons-game/handlers"

)

type HandlerRegistry struct {
	byPid        map[*actor.PID][]string
	byMessage    map[string][]*actor.PID
	uniqMessages []string
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		byPid:     make(map[*actor.PID][]string),
		byMessage: make(map[string][]*actor.PID),
	}
}

func (r *HandlerRegistry) Add(pid *actor.PID) error {
	response, err := actor.EmptyRootContext.RequestFuture(pid, &handlers.GetSupportedMessages{}, 5*time.Second).Result()
	if err != nil {
		return err
	}
	supportedMsgs := response.([]string)
	r.byPid[pid] = supportedMsgs

	for _, msg := range supportedMsgs {
		r.byMessage[msg] = append(r.byMessage[msg], pid)
	}

	uniqMessages := make([]string, len(r.byMessage))
	i := 0
	for msg := range r.byMessage {
		uniqMessages[i] = msg
		i++
	}
	r.uniqMessages = uniqMessages
	return nil
}

func (r *HandlerRegistry) AllMessages() []string {
	return r.uniqMessages
}

func (r *HandlerRegistry) ForMessage(messageType string) []*actor.PID {
	return r.byMessage[messageType]
}

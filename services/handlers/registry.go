package handlers

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
)

type Registry struct {
	byPid        map[*actor.PID][]string
	byMessage    map[string][]*actor.PID
	uniqMessages []string
}

func NewRegistry() *Registry {
	return &Registry{
		byPid:     make(map[*actor.PID][]string),
		byMessage: make(map[string][]*actor.PID),
	}
}

func (r *Registry) Add(pid *actor.PID) error {
	response, err := actor.EmptyRootContext.RequestFuture(pid, &GetSupportedMessages{}, 5*time.Second).Result()
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

func (r *Registry) AllMessages() []string {
	return r.uniqMessages
}

func (r *Registry) ForMessage(messageType string) []*actor.PID {
	return r.byMessage[messageType]
}

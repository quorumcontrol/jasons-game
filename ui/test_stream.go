package ui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/stretchr/testify/require"
)

type TestStream struct {
	t        *testing.T
	messages []*jasonsgame.UserInterfaceMessage
	events   *eventstream.EventStream
	wg       sync.WaitGroup
}

func NewTestStream(t *testing.T) *TestStream {
	return &TestStream{
		t:      t,
		events: new(eventstream.EventStream),
		wg:     sync.WaitGroup{},
	}
}

func (ts *TestStream) Send(msg *jasonsgame.UserInterfaceMessage) error {
	ts.messages = append(ts.messages, msg)
	ts.events.Publish(msg)
	return nil
}

func (ts *TestStream) GetMessages() []*jasonsgame.UserInterfaceMessage {
	return ts.messages
}

func (ts *TestStream) ClearMessages() error {
	ts.messages = NewTestStream(ts.t).messages
	return nil
}

func (ts *TestStream) Wait() {
	ts.wg.Wait()
	// Just a slight extra sleep to make sure ui refreshes
	time.Sleep(25 * time.Millisecond)
}

func (ts *TestStream) ExpectMessage(msg string, timeout time.Duration) {
	found := make(chan string, 1)

	subscription := ts.subscribe(func(evt interface{}) {
		switch eMsg := evt.(type) {
		case *jasonsgame.UserInterfaceMessage:
			if userMsg := eMsg.GetUserMessage(); userMsg != nil {
				if strings.Contains(userMsg.GetMessage(), msg) {
					found <- userMsg.GetMessage()
				}
			}
		}
	})

	go func() {
		defer ts.unsubscribe(subscription)

		for {
			select {
			case foundMsg := <-found:
				require.Contains(ts.t, foundMsg, msg)
				return
			case <-time.After(timeout):
				require.Fail(ts.t, fmt.Sprintf("timeout waiting for user message: %s", msg))
				return
			}
		}
	}()
}

func (ts *TestStream) subscribe(fn func(evt interface{})) *eventstream.Subscription {
	ts.wg.Add(1)
	return ts.events.Subscribe(fn)
}

func (ts *TestStream) unsubscribe(sub *eventstream.Subscription) {
	ts.events.Unsubscribe(sub)
	ts.wg.Done()
}

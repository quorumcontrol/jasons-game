package server

import (
	"context"
	"log"
	"os"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

const statePath = "/tmp/jasonsgame"

type GameServer struct {
	sessions map[string]*actor.PID
	network  network.Network
}

func NewGameServer(ctx context.Context) *GameServer {
	group, err := setupNotaryGroup(ctx)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	os.RemoveAll(statePath)
	os.MkdirAll(statePath, 0755)
	defer os.RemoveAll(statePath)

	net, err := network.NewRemoteNetwork(ctx, group, statePath)
	if err != nil {
		panic(errors.Wrap(err, "setting up notary group"))
	}

	return &GameServer{
		sessions: make(map[string]*actor.PID),
		network:  net,
	}
}

func (gs *GameServer) SendCommand(ctx context.Context, inp *jasonsgame.UserInput) (*jasonsgame.CommandReceived, error) {
	log.Printf("received: %v", inp)
	act, ok := gs.sessions[inp.Session.Uuid]
	if !ok {
		log.Printf("creating actor")
		act = actor.EmptyRootContext.Spawn(NewUIProps(gs.network))
		gs.sessions[inp.Session.Uuid] = act
	}

	actor.EmptyRootContext.Send(act, inp)
	return &jasonsgame.CommandReceived{}, nil
}

func (gs *GameServer) ReceiveUserMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUserMessagesServer) error {
	log.Printf("receive user messages %v", sess)
	act, ok := gs.sessions[sess.Uuid]
	if !ok {
		log.Printf("creating actor")
		act = actor.EmptyRootContext.Spawn(NewUIProps(gs.network))
		gs.sessions[sess.Uuid] = act
	}

	actor.EmptyRootContext.Send(act, &subscribeStream{stream: stream})

	return nil
}

func (gs *GameServer) ReceiveStatMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveStatMessagesServer) error {
	return nil
}

// func (s *BookServer) QueryBooks(bookQuery *books.QueryBooksRequest, stream books.BookService_QueryBooksServer) error {
// 	stream.SendHeader(metadata.Pairs("Pre-Response-Metadata", "Is-sent-as-headers-stream"))
// 	// for _, book := range books {
// 	// 	if strings.HasPrefix(book.Author, bookQuery.AuthorPrefix) {
// 	// 		stream.Send(book)
// 	// 	}
// 	// }
// 	stream.SetTrailer(metadata.Pairs("Post-Response-Metadata", "Is-sent-as-trailers-stream"))
// 	return nil
// }

package server

import (
	"context"
	"log"

	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

type GameServer struct{}

func (gs *GameServer) SendCommand(ctx context.Context, inp *jasonsgame.UserInput) (*jasonsgame.CommandReceived, error) {
	log.Printf("received: %v", inp)
	return &jasonsgame.CommandReceived{}, nil
}

func (gs *GameServer) ReceiveUserMessages(sess *jasonsgame.Session, stream jasonsgame.GameService_ReceiveUserMessagesServer) error {
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

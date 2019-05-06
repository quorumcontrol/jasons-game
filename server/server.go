package server

import (
	"context"

	"github.com/quorumcontrol/jasons-game/pb/books"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type BookServer struct {
}

func (s *BookServer) GetBook(ctx context.Context, bookQuery *books.GetBookRequest) (*books.Book, error) {
	grpc.SendHeader(ctx, metadata.Pairs("Pre-Response-Metadata", "Is-sent-as-headers-unary"))
	grpc.SetTrailer(ctx, metadata.Pairs("Post-Response-Metadata", "Is-sent-as-trailers-unary"))

	return &books.Book{Isbn: 1234, Title: "Rnadom book"}, nil

	// for _, book := range books {
	// 	if book.Isbn == bookQuery.Isbn {
	// 		return book, nil
	// 	}
	// }

	return nil, grpc.Errorf(codes.NotFound, "Book could not be found")
}

func (s *BookServer) QueryBooks(bookQuery *books.QueryBooksRequest, stream books.BookService_QueryBooksServer) error {
	stream.SendHeader(metadata.Pairs("Pre-Response-Metadata", "Is-sent-as-headers-stream"))
	// for _, book := range books {
	// 	if strings.HasPrefix(book.Author, bookQuery.AuthorPrefix) {
	// 		stream.Send(book)
	// 	}
	// }
	stream.SetTrailer(metadata.Pairs("Post-Response-Metadata", "Is-sent-as-trailers-stream"))
	return nil
}

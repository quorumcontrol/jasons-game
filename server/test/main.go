package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pkg/errors"
	logging "github.com/ipfs/go-log"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func mustSetLogLevel(name, level string) {
	err := logging.SetLogLevel(name, level)
	if err != nil {
		panic(errors.Wrap(err, "error setting log level"))
	}
}

func main() {
	mustSetLogLevel("*", "INFO")
	mustSetLogLevel("swarm2", "error")
	mustSetLogLevel("relay", "error")
	mustSetLogLevel("autonat", "error")
	mustSetLogLevel("uiserver", "debug")
	mustSetLogLevel("game", "debug")
	mustSetLogLevel("gameserver", "debug")

	port := 8080
	grpcServer := grpc.NewServer()
	fmt.Println("Starting Jasons Game server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server.NewGameServer(ctx)

	jasonsgame.RegisterGameServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	fmt.Println("Listening on port", port)

	wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(_origin string) bool { return true }))

	serv := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		MaxHeaderBytes: 1 << 20,
	}

	serv.Handler = http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsGrpcWebRequest(req) {
			log.Printf("grpc request")
			wrappedGrpc.ServeHTTP(resp, req)
			return
		}

		if wrappedGrpc.IsAcceptableGrpcCorsRequest(req) {
			log.Printf("options request")
			headers := resp.Header()
			headers.Add("Access-Control-Allow-Origin", "*")
			headers.Add("Access-Control-Allow-Headers", "*")
			headers.Add("Access-Control-Allow-Methods", "GET, POST,OPTIONS")
			resp.WriteHeader(http.StatusOK)
			return
		}

		//TODO: this is a good place to stick in the UI
		log.Printf("unkown route: %v", req)

	})

	log.Fatal(serv.ListenAndServe())
}

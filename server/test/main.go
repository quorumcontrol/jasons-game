package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := 8080
	grpcServer := grpc.NewServer()
	fmt.Println("Starting Tupelo RPC server")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// // By providing port 0 to net.Listen, we get a randomized one
	// if port <= 0 {
	// 	port = 0
	// }
	// listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	// if err != nil {
	// 	log.Printf("Failed to open listener: %s", err)
	// 	panic(err)
	// }

	// if port == 0 {
	// 	comps := strings.Split(listener.Addr().String(), ":")
	// 	port, err = strconv.Atoi(comps[len(comps)-1])
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	s := server.NewGameServer(ctx)

	jasonsgame.RegisterGameServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	fmt.Println("Listening on port", port)

	wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(_origin string) bool { return true }))

	serv := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
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

	// err = grpcServer.Serve(listener)
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/quorumcontrol/jasons-game/build"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/server"
	"github.com/quorumcontrol/jasons-game/ui"
)

var inkDID string // set by an ldflag at compile time (e.g. go build -ldflags "-X main.inkDID=did:tupelo:blahblah")

func main() {

	if os.Getenv("PPROF_ENABLED") == "true" {
		go func() {
			debugR := mux.NewRouter()
			debugR.HandleFunc("/debug/pprof/", pprof.Index)
			debugR.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			debugR.HandleFunc("/debug/pprof/profile", pprof.Profile)
			debugR.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			debugR.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			debugR.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			debugR.Handle("/debug/pprof/block", pprof.Handler("block"))
			debugR.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
			err := http.ListenAndServe(":8081", debugR)
			if err != nil {
				fmt.Println(err.Error())
			}
		}()
	}

	config.MustSetLogLevel("*", "INFO")
	config.MustSetLogLevel("swarm2", "error")
	config.MustSetLogLevel("pubsub", "error")
	config.MustSetLogLevel("relay", "error")
	config.MustSetLogLevel("autonat", "info")
	config.MustSetLogLevel("dht", "error")
	config.MustSetLogLevel("uiserver", "debug")
	config.MustSetLogLevel("game", "debug")
	config.MustSetLogLevel("gameserver", "debug")
	config.MustSetLogLevel("gamenetwork", "debug")
	config.MustSetLogLevel("invites", "debug")

	port := 8080
	grpcServer := grpc.NewServer()
	fmt.Println("Starting Jasons Game server")

	fmt.Println(build.BuildLabel, "build")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localnet := ui.SetOptions()

	if inkDID == "" {
		inkDID = os.Getenv("INK_DID")
	}

	gsCfg := server.GameServerConfig{
		LocalNet: *localnet,
		InkDID:   inkDID,
	}
	s := server.NewGameServer(ctx, gsCfg)

	jasonsgame.RegisterGameServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	fmt.Println("Listening on port", port)

	wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(_origin string) bool { return true }))

	serv := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		MaxHeaderBytes: 1 << 20,
	}

	box := packr.New("Frontend", "./frontend/jasons-game/public")

	fs := http.FileServer(box)

	fsWithHeaders := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		headers := resp.Header()
		headers.Set("Content-Security-Policy", "default-src 'self'; " +
			"object-src 'none'; " +
			"style-src 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com; " +
			"font-src https://cdn.jsdelivr.net https://fonts.gstatic.com data:; " +
			"script-src 'self' 'sha256-eBU0yMA10wlS8+IouTT5knu2DQmVTyICn7YbqhWu3fw='")

		fs.ServeHTTP(resp, req)
	})

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
			headers.Add("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			resp.WriteHeader(http.StatusOK)
			return
		}

		fsWithHeaders.ServeHTTP(resp, req)

	})

	fmt.Println("listen and serve")
	go func() {
		log.Fatal(serv.ListenAndServe())
	}()

	<-make(chan struct{})
}

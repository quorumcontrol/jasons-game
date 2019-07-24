package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/gorilla/mux"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"

	"github.com/gobuffalo/packr/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/quorumcontrol/jasons-game/build"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
	"github.com/quorumcontrol/jasons-game/server"
	"github.com/quorumcontrol/jasons-game/ui"
)

func mustSetLogLevel(name, level string) {
	err := logging.SetLogLevel(name, level)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error setting log level (%s %s)", name, level)))
	}
}

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

	mustSetLogLevel("*", "INFO")
	mustSetLogLevel("swarm2", "error")
	mustSetLogLevel("pubsub", "error")
	mustSetLogLevel("relay", "error")
	mustSetLogLevel("autonat", "info")
	mustSetLogLevel("dht", "error")
	mustSetLogLevel("uiserver", "debug")
	mustSetLogLevel("game", "debug")
	mustSetLogLevel("gameserver", "debug")
	mustSetLogLevel("gamenetwork", "debug")

	port := 8080
	grpcServer := grpc.NewServer()
	fmt.Println("Starting Jasons Game server")

	fmt.Println(build.BuildLabel, "build")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	disableWebView, localnet := ui.SetOptions()

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

		fs.ServeHTTP(resp, req)

	})

	if *disableWebView {
		fmt.Println("webview disabled")
		log.Fatal(serv.ListenAndServe())
		return
	}

	fmt.Println("listen and serve")
	go func() {
		log.Fatal(serv.ListenAndServe())
	}()
	fmt.Println("opening webview")
	ui.OpenWebView()
}

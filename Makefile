VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

# This is important to export until we're on Go 1.13+ or packr can break
export GO111MODULE = on

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

jsmodules = ./frontend/jasons-game/node_modules
generated = pb/jasonsgame/jasonsgame.pb.go frontend/jasons-game/src/js/frontend/remote/*_pb.* game/messages_gen.go game/messages_gen_test.go messages/messages_gen.go messages/messages_gen_test.go
packr = packrd/packed-packr.go main-packr.go

all: frontend-build $(packr) build
	$(FIRSTGOPATH)/bin/packr2 clean

$(FIRSTGOPATH)/src/github.com/gogo/protobuf/proto:
	go get github.com/gogo/protobuf/proto

$(FIRSTGOPATH)/src/github.com/gogo/protobuf/gogoproto:
	go get github.com/gogo/protobuf/gogoproto

$(FIRSTGOPATH)/bin/protoc-gen-gogofaster: $(FIRSTGOPATH)/src/github.com/gogo/protobuf/proto $(FIRSTGOPATH)/src/github.com/gogo/protobuf/gogoproto
	go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

$(generated): $(FIRSTGOPATH)/bin/protoc-gen-gogofaster $(jsmodules) game/messages.go
	./scripts/protogen.sh
	cd game && go generate
	cd messages && go generate
	
$(jsmodules):
	cd frontend/jasons-game && npm install

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

$(FIRSTGOPATH)/bin/msgp:
	go get github.com/tinylib/msgp

bin/jasonsgame: $(generated) go.mod go.sum
	mkdir -p bin
	go build -tags=desktop -o ./bin/jasonsgame

build: bin/jasonsgame

lint: $(FIRSTGOPATH)/bin/golangci-lint $(generated)
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags integration

test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum

generate: game/messages.go $(FIRSTGOPATH)/bin/protoc-gen-gogofaster $(FIRSTGOPATH)/bin/msgp
	scripts/protogen.sh && cd game && go generate

integration-test: $(generated) go.mod go.sum
ifdef testpackage
	TEST_PACKAGE=${testpackage} docker-compose -f docker-compose-dev.yml run --rm integration
else
	docker-compose -f docker-compose-dev.yml run --rm integration
endif

localnet: $(generated) go.mod go.sum
	docker-compose -f docker-compose-localnet.yml up --force-recreate

game-server: $(generated) go.mod go.sum
ifdef testnet
	docker-compose -f docker-compose-dev.yml run --rm --service-ports game-testnet
else
	docker-compose -f docker-compose-dev.yml run --rm --service-ports game
endif

game2: $(generated) go.mod go.sum
	docker-compose -f docker-compose-dev.yml run --rm --service-ports game2

jason: $(generated) go.mod go.sum
	docker-compose -f docker-compose-dev.yml up --force-recreate jason

frontend-build: $(generated) $(jsmodules)
	cd frontend/jasons-game && ./node_modules/.bin/shadow-cljs release app

frontend-dev: $(generated) $(jsmodules)
	cd frontend/jasons-game && ./node_modules/.bin/shadow-cljs watch app

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

$(FIRSTGOPATH)/bin/packr2:
	go get -u github.com/gobuffalo/packr/v2/packr2

$(packr): $(FIRSTGOPATH)/bin/packr2 main.go
	$(FIRSTGOPATH)/bin/packr2

clean: $(FIRSTGOPATH)/bin/packr2
	$(FIRSTGOPATH)/bin/packr2 clean
	go clean ./...
	rm -rf vendor
	rm -rf bin

.PHONY: all build test integration-test localnet clean lint game-server jason game2

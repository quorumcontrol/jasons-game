VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

jsmodules = ./frontend/jasons-game/node_modules
generated = pb/jasonsgame/jasonsgame.pb.go frontend/jasons-game/src/js/frontend/remote/*_pb.*

all: build jsmodules

$(FIRSTGOPATH)/src/github.com/gogo/protobuf/proto:
	go get github.com/gogo/protobuf/proto

$(FIRSTGOPATH)/src/github.com/gogo/protobuf/gogoproto:
	go get github.com/gogo/protobuf/gogoproto

$(FIRSTGOPATH)/bin/protoc-gen-gogofaster: $(FIRSTGOPATH)/src/github.com/gogo/protobuf/proto $(FIRSTGOPATH)/src/github.com/gogo/protobuf/gogoproto
	go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

$(generated): $(FIRSTGOPATH)/bin/protoc-gen-gogofaster $(jsmodules)
	scripts/protogen.sh

$(jsmodules):
	cd frontend/jasons-game && npm install

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

build: $(generated) go.mod go.sum
	go build ./...

lint: $(FIRSTGOPATH)/bin/golangci-lint $(generated)
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags integration

test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum

integration-test: $(generated) go.mod go.sum
ifdef testpackage
	TEST_PACKAGE=${testpackage} docker-compose -f docker-compose-dev.yml run --rm integration
else
	docker-compose -f docker-compose-dev.yml run --rm integration
endif

localnet: $(generated) go.mod go.sum
	docker-compose -f docker-compose-localnet.yml up --force-recreate

game-server: $(generated) go.mod go.sum
	docker-compose -f docker-compose-dev.yml run --rm --service-ports game

frontend-dev: $(generated) $(jsmodules)
	cd frontend/jasons-game && shadow-cljs watch app

clean:
	go clean ./...
	rm -rf vendor

.PHONY: all build test integration-test localnet clean lint game-server

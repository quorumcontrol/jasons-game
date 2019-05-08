VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)
generated = pb/jasonsgame/jasonsgame.pb.go frontend/jasons-game/src/js/frontend/remote/*_pb.*

all: build

$(FIRSTGOPATH)/src/github.com/gogo/protobuf/proto:
	go get github.com/gogo/protobuf/proto

$(FIRSTGOPATH)/src/github.com/gogo/protobuf/gogoproto:
	go get github.com/gogo/protobuf/gogoproto

$(FIRSTGOPATH)/bin/protoc-gen-gogofaster: $(FIRSTGOPATH)/src/github.com/gogo/protobuf/proto $(FIRSTGOPATH)/src/github.com/gogo/protobuf/gogoproto
	go get -u github.com/gogo/protobuf/protoc-gen-gogofaster

$(generated): $(FIRSTGOPATH)/bin/protoc-gen-gogofaster
	scripts/protogen.sh

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

build: $(gosources) $(generated) go.mod go.sum
	go build ./...

lint: $(FIRSTGOPATH)/bin/golangci-lint $(generated)
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags integration

test: $(gosources) $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum

integration-test: $(gosources) $(generated) go.mod go.sum
ifdef testpackage
	TEST_PACKAGE=${testpackage} docker-compose -f docker-compose-dev.yml run --rm integration
else
	docker-compose -f docker-compose-dev.yml run --rm integration
endif

localnet: $(gosources) $(generated) go.mod go.sum
	docker-compose -f docker-compose-localnet.yml up --force-recreate

game-server: $(gosources) $(generated) go.mod go.sum
	docker-compose -f docker-compose-dev.yml run --rm --service-ports game

clean:
	go clean ./...
	rm -rf vendor

.PHONY: all build test integration-test localnet clean lint game-server

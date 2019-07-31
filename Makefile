VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

BUILD ?= public

# This is important to export until we're on Go 1.13+ or packr can break
export GO111MODULE = on

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

jsmodules = frontend/jasons-game/node_modules
generated = network/messages.pb.go game/types.pb.go pb/jasonsgame/jasonsgame.pb.go \
            inkfaucet/inkfaucet/messages.pb.go \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb.d.ts \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb.js \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.d.ts \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.js

packr = packrd/packed-packr.go main-packr.go
gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: frontend-build $(packr) build

# Turn off go mod so that this will install to $GOPATH/src instead of $GOPATH/pkg/mod
${FIRSTGOPATH}/src/github.com/gogo/protobuf/protobuf:
	env GO111MODULE=off go get github.com/gogo/protobuf/...

%.pb.go %_pb.d.ts %_pb_service.d.ts %_pb.js %_pb_service.js: %.proto $(FIRSTGOPATH)/src/github.com/gogo/protobuf/protobuf $(jsmodules)
	./scripts/protogen.sh

generated: $(generated)
	
$(jsmodules):
	cd frontend/jasons-game && npm install

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

bin/jasonsgame-$(BUILD): $(gosources) $(generated) go.mod go.sum
	mkdir -p bin
	go build -tags='desktop $(BUILD)' -ldflags="-X main.inkDID=${INK_DID}" -o ./bin/jasonsgame-$(BUILD)

build: bin/jasonsgame-$(BUILD)

JasonsGame-$(BUILD).app/Contents/MacOS/jasonsgame: $(generated) go.mod go.sum frontend-build $(packr)
	mkdir -p JasonsGame-$(BUILD).app/Contents/MacOS
	go build -tags='desktop macos_app_bundle $(BUILD)' -o JasonsGame-$(BUILD).app/Contents/MacOS/jasonsgame

JasonsGame-$(BUILD).app: JasonsGame-$(BUILD).app/Contents/MacOS/jasonsgame

mac-app: JasonsGame-$(BUILD).app

lint: $(FIRSTGOPATH)/bin/golangci-lint $(generated)
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags 'integration public'
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags 'integration internal'

test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum
	gotestsum -- -tags=internal ./...

ci-test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	mkdir -p test_results/tests
	gotestsum --junitfile=test_results/tests/results.xml -- -mod=readonly ./...

integration-test: $(generated) go.mod go.sum
ifdef testpackage
	env TEST_PACKAGE=${testpackage} docker-compose -p jasons-game -f docker-compose-dev.yml run --rm integration
else
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm integration
endif

localnet: $(generated) go.mod go.sum
	docker-compose -p jasons-game -f docker-compose-localnet.yml pull --quiet
	docker-compose -p jasons-game -f docker-compose-localnet.yml up --force-recreate

game-server: $(generated) go.mod go.sum
ifdef testnet
	env INK_DID=$(INK_DID) docker-compose -p jasons-game -f docker-compose-dev.yml run --rm --service-ports game-testnet
else
	env INK_DID=$(INK_DID) docker-compose -p jasons-game -f docker-compose-dev.yml run --rm --service-ports game
endif

importer: $(generated) go.mod go.sum
ifdef testnet
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm importer-testnet
else
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm importer
endif

game2: $(generated) go.mod go.sum
	env INK_DID=$(INK_DID) docker-compose -p jasons-game -f docker-compose-dev.yml run --rm --service-ports game2

jason: $(generated) go.mod go.sum
	docker-compose -p jasons-game -f docker-compose-dev.yml up --force-recreate jason

inkfaucet: $(generated) go.mod go.sum
	env TOKEN_PAYLOAD=$(TOKEN_PAYLOAD) INK_FAUCET_KEY=${INK_FAUCET_KEY} docker-compose -f docker-compose-dev.yml run --rm inkfaucet

inkfaucetdid:
	docker-compose -f docker-compose-dev.yml run --rm inkfaucetdid

devink: $(generated) go.mod go.sum
	env INK_FAUCET_KEY=$(INK_FAUCET_KEY) docker-compose -f docker-compose-dev.yml run --rm devink

invite: $(generated) go.mod go.sum
	env INK_DID=$(INK_DID) docker-compose -f docker-compose-dev.yml run --rm invite

dev:
	scripts/start-dev.sh

down:
	docker-compose -f docker-compose-dev.yml down
	docker-compose -f docker-compose-localnet.yml down

frontend-build: $(generated) $(jsmodules)
	cd frontend/jasons-game && ./node_modules/.bin/shadow-cljs release app

frontend-dev: $(generated) $(jsmodules)
	cd frontend/jasons-game && ./node_modules/.bin/shadow-cljs watch app

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

prepare: $(gosources) $(generated) $(packr) $(vendor)

$(FIRSTGOPATH)/bin/packr2:
	go get -u github.com/gobuffalo/packr/v2@662c20c19dde

$(packr): $(FIRSTGOPATH)/bin/packr2 main.go
	$(FIRSTGOPATH)/bin/packr2

clean: $(FIRSTGOPATH)/bin/packr2
	$(FIRSTGOPATH)/bin/packr2 clean
	go clean -tags='internal public' ./...
	rm -rf vendor
	rm -rf bin
	rm -rf JasonsGame.app-$(BUILD)/Contents/MacOS
	rm -f $(generated)

.PHONY: all build test integration-test localnet clean lint game-server importer jason inkfaucet devink game2 mac-app prepare generated dev down

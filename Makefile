VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

# This is important to export until we're on Go 1.13+ or packr can break
export GO111MODULE = on

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

jsmodules = frontend/jasons-game/node_modules
generated = network/messages.pb.go game/types.pb.go pb/jasonsgame/jasonsgame.pb.go \
            inkwell/inkwell/messages.pb.go \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb.d.ts \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb.js \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.d.ts \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.js

packr = packrd/packed-packr.go main-packr.go
gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: frontend-build $(packr) build

${FIRSTGOPATH}/src/github.com/gogo/protobuf/protobuf:
	go get github.com/gogo/protobuf/...

%.pb.go %_pb.d.ts %_pb_service.d.ts %_pb.js %_pb_service.js: %.proto $(FIRSTGOPATH)/src/github.com/gogo/protobuf/protobuf $(jsmodules)
	./scripts/protogen.sh

generated: $(generated)
	
$(jsmodules):
	cd frontend/jasons-game && npm install

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

bin/jasonsgame: $(gosources) $(generated) go.mod go.sum
	mkdir -p bin
	go build -tags=desktop -o ./bin/jasonsgame

build: bin/jasonsgame

JasonsGame.app/Contents/MacOS/jasonsgame: $(generated) go.mod go.sum frontend-build $(packr)
	mkdir -p JasonsGame.app/Contents/MacOS
	go build -tags='desktop macos_app_bundle' -o JasonsGame.app/Contents/MacOS/jasonsgame

JasonsGame.app: JasonsGame.app/Contents/MacOS/jasonsgame

mac-app: JasonsGame.app

lint: $(FIRSTGOPATH)/bin/golangci-lint $(generated)
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags integration

test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum

ci-test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	mkdir -p test_results/tests
	gotestsum --junitfile=test_results/tests/results.xml -- -mod=readonly ./...

integration-test: $(generated) go.mod go.sum
ifdef testpackage
	TEST_PACKAGE=${testpackage} docker-compose -p jasons-game -f docker-compose-dev.yml run --rm integration
else
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm integration
endif

localnet: $(generated) go.mod go.sum
	docker-compose -p jasons-game -f docker-compose-localnet.yml pull --quiet
	docker-compose -p jasons-game -f docker-compose-localnet.yml up --force-recreate

game-server: $(generated) go.mod go.sum
ifdef testnet
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm --service-ports game-testnet
else
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm --service-ports game
endif

game2: $(generated) go.mod go.sum
	docker-compose -p jasons-game -f docker-compose-dev.yml run --rm --service-ports game2

jason: $(generated) go.mod go.sum
	docker-compose -p jasons-game -f docker-compose-dev.yml up --force-recreate jason

inkwell: $(generated) go.mod go.sum
	env TOKEN_PAYLOAD=$(TOKEN_PAYLOAD) docker-compose -f docker-compose-dev.yml run --rm inkwell

inkwelldid:
	docker-compose -f docker-compose-dev.yml run --rm inkwelldid

devink: $(generated) go.mod go.sum
	env INKWELL_DID=$(INKWELL_DID) docker-compose -f docker-compose-dev.yml run --rm devink

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
	get -u github.com/gobuffalo/packr/v2@662c20c19dde

$(packr): $(FIRSTGOPATH)/bin/packr2 main.go
	$(FIRSTGOPATH)/bin/packr2

clean: $(FIRSTGOPATH)/bin/packr2
	$(FIRSTGOPATH)/bin/packr2 clean
	go clean ./...
	rm -rf vendor
	rm -rf bin
	rm -rf JasonsGame.app/Contents/MacOS

.PHONY: all build test integration-test localnet clean lint game-server jason inkwell devink game2 mac-app prepare generated dev down

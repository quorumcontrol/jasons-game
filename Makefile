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

generated = network/messages.pb.go game/types.pb.go pb/jasonsgame/jasonsgame.pb.go \
            inkfaucet/inkfaucet/messages.pb.go \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb.d.ts \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb.js \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.d.ts \
            frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.js

packr = packrd/packed-packr.go main-packr.go
gosources = $(shell find . -path "./vendor/*" -prune -o -path "./dist/*" -prune -o -type f -name "*.go" -print)

jsmodules = node_modules frontend/jasons-game/node_modules
cljssources = $(shell find ./frontend/jasons-game/src -type f -name "*.cljs" -print)
jsgenerated = frontend/jasons-game/public/js/compiled/base.js

all: frontend-build $(packr) build

# OS detection boilerplate

ifeq ($(OS), Windows_NT)
  PLATFORM ?= win32
  EXE_SUFFIX=.exe
else
  UNAME := $(shell uname -s)
  ifeq ($(UNAME), Linux)
    PLATFORM ?= linux
  endif
  ifeq ($(UNAME), Darwin)
    PLATFORM ?= darwin
  endif
endif

# end OS detection boilerplate

# Turn off go mod so that this will install to $GOPATH/src instead of $GOPATH/pkg/mod
${FIRSTGOPATH}/src/github.com/gogo/protobuf/protobuf:
	env GO111MODULE=off go get -u github.com/gogo/protobuf/...

%.pb.go %_pb.d.ts %_pb_service.d.ts %_pb.js %_pb_service.js: %.proto $(FIRSTGOPATH)/src/github.com/gogo/protobuf/protobuf $(jsmodules)
	./scripts/protogen.sh

generated: $(generated)
	
$(jsmodules): package.json package-lock.json frontend/jasons-game/package.json frontend/jasons-game/package-lock.json
	# touch node_modules dirs so make doesn't reinstall node modules every time
	npm install && touch node_modules && touch frontend/jasons-game/node_modules

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	env GO111MODULE=off go get gotest.tools/gotestsum

$(FIRSTGOPATH)/bin/xgo:
	env GO111MODULE=off go get src.techknowlogick.com/xgo

bin/jasonsgame-darwin-$(BUILD): $(gosources) $(generated) go.mod go.sum vendor $(FIRSTGOPATH)/bin/xgo
	mkdir -p bin
	$(FIRSTGOPATH)/bin/xgo -go go-1.13.4 -tags='public' -targets='darwin-10.10/amd64,' -ldflags="-X main.inkDID=${INK_DID}" ./
	mv github.com/quorumcontrol/jasons-game-darwin-* bin/jasonsgame-darwin-$(BUILD)

bin/jasonsgame-win32-$(BUILD).exe: $(gosources) $(generated) go.mod go.sum vendor $(FIRSTGOPATH)/bin/xgo
	mkdir -p bin
	$(FIRSTGOPATH)/bin/xgo -go go-1.13.4 -tags='public' -targets='windows-6.0/amd64,' -ldflags="-X main.inkDID=${INK_DID}" ./
	mv github.com/quorumcontrol/jasons-game-windows-* bin/jasonsgame-win32-$(BUILD).exe

bin/jasonsgame-linux-$(BUILD): $(gosources) $(generated) go.mod go.sum vendor $(FIRSTGOPATH)/bin/xgo
	mkdir -p bin
	$(FIRSTGOPATH)/bin/xgo -go go-1.13.4 -tags='public' -targets='linux/amd64,' -ldflags="-X main.inkDID=${INK_DID}" ./
	mv github.com/quorumcontrol/jasons-game-linux-* bin/jasonsgame-linux-$(BUILD)

out/make/zip/darwin: bin/jasonsgame-darwin-$(BUILD) frontend/main.js forge.config.js
	npm run make-darwin

out/make/zip/linux: bin/jasonsgame-linux-$(BUILD) frontend/main.js forge.config.js
	npm run make-linux

out/make/squirrel.windows: bin/jasonsgame-win32-$(BUILD).exe frontend/main.js forge.config.js
	npm run make-win32

all-go-binaries: bin/jasonsgame-win32-$(BUILD).exe bin/jasonsgame-linux-$(BUILD) bin/jasonsgame-darwin-$(BUILD)

all-platforms: out/make/zip/darwin out/make/zip/linux out/make/squirrel.windows

ifeq ($(PLATFORM), all)
  build: $(jsmodules) bin/jasonsgame-win32-$(BUILD).exe bin/jasonsgame-darwin-$(BUILD) bin/jasonsgame-linux-$(BUILD) build-all
else
  ifeq ($(PLATFORM), win32)
    TARGET=squirrel.windows
    EXE_SUFFIX=.exe
  else
    TARGET=zip/$(PLATFORM)
  endif
  build: $(jsmodules) bin/jasonsgame-$(PLATFORM)-$(BUILD)$(EXE_SUFFIX) out/make/$(TARGET)
endif

release: $(jsmodules) all-go-binaries all-platforms
	npm run publish-win32
	npm run publish-darwin
	npm run publish-linux

lint: $(FIRSTGOPATH)/bin/golangci-lint $(generated)
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags 'integration public'
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags 'integration internal'

test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum
	gotestsum -- -tags=internal ./...

ci-test: $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	mkdir -p test_results/tests
	gotestsum --junitfile=test_results/tests/results.xml -- ./...

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

frontend/jasons-game/node_modules/.bin/shadow-cljs:
	cd frontend/jasons-game && npm i

$(jsgenerated): $(jsmodules) $(generated) $(cljssources) frontend/jasons-game/externs/app.txt frontend/jasons-game/shadow-cljs.edn frontend/jasons-game/node_modules/.bin/shadow-cljs
	cd frontend/jasons-game && ./node_modules/.bin/shadow-cljs release app

frontend-build: frontend/jasons-game/public/js/compiled/base.js

frontend-dev: $(generated) $(jsmodules) frontend/jasons-game/node_modules/.bin/shadow-cljs
	cd frontend/jasons-game && ./node_modules/.bin/shadow-cljs watch app

fetchdids = ssh ec2-user@54.149.56.75 'for cid in $$(docker ps -q); do docker logs $$cid 2>&1 | grep -o -P "did:tupelo:0x[a-fA-F0-9]+"; done | sort | uniq' > $@

dids.txt:
	$(fetchdids)

cache/dids.txt:
	$(fetchdids)

cache/out/nodes.gz: cache/dids.txt
	go run cache/cmd/main.go

node-cache: cache/out/nodes.gz

bin/benchmark: $(generated) $(gosources) dids.txt
	go build -o $@ ./benchmark/cmd/cli/main.go

benchmark/lambda/benchmark.zip: $(generated) $(gosources) dids.txt
	mkdir -p benchmark/lambda
	go build -o benchmark/lambda/benchmark ./benchmark/cmd/lambda/main.go
	cp dids.txt benchmark/lambda/
	zip -D $@ $(@D)/*

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

prepare: $(gosources) $(generated) $(packr) vendor

$(FIRSTGOPATH)/bin/packr2:
	env GO111MODULE=off go get -u github.com/gobuffalo/packr/v2/packr2

$(packr): $(FIRSTGOPATH)/bin/packr2 main.go $(jsgenerated) frontend/jasons-game/public/index.html
	$(FIRSTGOPATH)/bin/packr2

clean: $(FIRSTGOPATH)/bin/packr2
	$(FIRSTGOPATH)/bin/packr2 clean
	go clean -tags='internal public' ./...
	rm -rf vendor
	rm -rf bin
	rm -f $(generated)
	rm -rf $(jsmodules)
	rm -rf frontend/jasons-game/public/js/compiled
	rm -rf out/*
	rm -f courts/yml.tar.gz

encrypt: courts/yml/
	tar -zcf courts/yml.tar.gz courts/yml/
	git secret add courts/yml.tar.gz
	git secret hide
	rm -f courts/yml.tar.gz
	git add courts/yml.tar.gz.secret .gitsecret/paths/mapping.cfg

decrypt: courts/yml.tar.gz.secret
	git secret cat courts/yml.tar.gz.secret | tar -zxf -

.PHONY: all build build-all test integration-test localnet clean lint game-server importer jason inkfaucet devink game2 mac-app prepare generated dev down encrypt decrypt node-cache

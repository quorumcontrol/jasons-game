VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: build

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

build: $(gosources) $(generated) go.mod go.sum
	go build ./...

lint: $(FIRSTGOPATH)/bin/golangci-lint
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

docker-image: vendor $(gosources) $(generated) Dockerfile .dockerignore
	docker build -t quorumcontrol/jasons-game:$(TAG) .

game-server: $(gosources) $(generated) go.mod go.sum
	docker-compose run --rm --service-ports game

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

clean:
	go clean ./...
	rm -rf vendor

.PHONY: all build test integration-test localnet docker-image clean

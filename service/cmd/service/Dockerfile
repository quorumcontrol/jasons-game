FROM golang:1.12.5-alpine3.9 AS build

RUN apk add --no-cache --update build-base git

RUN	go get -u github.com/gobuffalo/packr/v2/packr2

WORKDIR /src
# NOTE: make runs this Dockerfile in the context of the root
# project directory
COPY . .

# This is important until we're on Go 1.13+ or packr can break
ENV GO111MODULE on

RUN packr2 && cd /src/service/cmd/service && packr2

RUN go build -o jason-listener-service -mod=vendor -v -a /src/service/cmd/service && cp ./jason-listener-service /go/bin/jason-listener-service

FROM alpine:3.9
LABEL maintainer="dev@quorumcontrol.com"

COPY --from=build /go/bin/jason-listener-service /usr/bin/jason-listener-service

ENTRYPOINT ["/usr/bin/jason-listener-service"]

FROM golang:1.12.5-alpine3.9 AS build

WORKDIR /app

RUN apk add --no-cache --update build-base git

COPY . .

# This is important until we're on Go 1.13+ or packr can break
ENV GO111MODULE on

RUN go install -mod=vendor -v -a -gcflags=-trimpath="${PWD}" -asmflags=-trimpath="${PWD}"

FROM alpine:3.9
LABEL maintainer="dev@quorumcontrol.com"

COPY --from=build /go/bin/jasons-game /usr/bin/jasons-game

ENTRYPOINT ["/usr/bin/jason"]

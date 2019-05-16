FROM golang:1.12.4-alpine3.9 AS build

WORKDIR /app

RUN apk add --no-cache --update build-base

COPY . .

RUN go install -mod=vendor -v -a -gcflags=-trimpath="${PWD}" -asmflags=-trimpath="${PWD}"
RUN go install -mod=vendor -v -a ./jason

FROM alpine:3.9
LABEL maintainer="dev@quorumcontrol.com"

COPY --from=build /go/bin/jasons-game /usr/bin/jasons-game
COPY --from=build /go/bin/jason /usr/bin/jason

ENTRYPOINT ["/usr/bin/tupelo"]

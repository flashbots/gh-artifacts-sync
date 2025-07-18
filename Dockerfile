# stage: build ---------------------------------------------------------

FROM golang:1.22-alpine as build

RUN apk add --no-cache gcc musl-dev linux-headers

WORKDIR /go/src/github.com/flashbots/gh-artifacts-sync

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o bin/gh-artifacts-sync -ldflags "-s -w" github.com/flashbots/gh-artifacts-sync/cmd

# stage: run -----------------------------------------------------------

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /go/src/github.com/flashbots/gh-artifacts-sync/bin/gh-artifacts-sync ./gh-artifacts-sync

ENTRYPOINT ["/app/gh-artifacts-sync"]

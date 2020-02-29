FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/github.com/mridul-sahu/greedy_auction
COPY auctioneer/ auctioneer/
COPY models/ models/
WORKDIR $GOPATH/src/github.com/mridul-sahu/greedy_auction/auctioneer/cmd
# Fetch dependencies.
# Using go get.
RUN go get -d -v
# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/auctioneer

FROM scratch
# Copy our static executable.
COPY --from=builder /go/bin/auctioneer /go/bin/auctioneer

ENTRYPOINT ["/go/bin/auctioneer"]
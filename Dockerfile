# Build Secret in a stock Go builder container
FROM golang:1.15-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-secret
RUN cd /go-secret && make secret

# Pull Secret into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/secret /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["secret"]

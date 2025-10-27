FROM golang:1.25-alpine AS builder

RUN apk update && apk add --no-cache git mercurial openssh make

ARG REPOSITORY_PRIVATE_KEY
ARG SSH_PRIVATE_KEY

ARG GOOS=linux
ENV GO111MODULE=on
ENV GOPRIVATE=tcb-odds/matching-engine

WORKDIR $GOPATH/src/tcb-odds/matching-engine

COPY . .

RUN make build

FROM alpine:3.11

WORKDIR /app

COPY --from=builder /go/src/tcb-odds/matching-engine/build/app /app/app

ENTRYPOINT ["/app/app"]
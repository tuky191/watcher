FROM golang:1.18.2-alpine3.15 AS go-builder
#ARG arch=x86_64


# this comes from standard alpine nightly file
#  https://github.com/rust-lang/docker-rust-nightly/blob/master/alpine3.12/Dockerfile
# with some changes to support our toolchain, etc
RUN set -eux; apk add --no-cache ca-certificates build-base;


WORKDIR /code
COPY . /code/

#Build watcher
RUN go build -o build/watcherd cmd/rpcwatcher/main.go
#Build sync
RUN go build -o build/syncd cmd/sync/main.go

FROM alpine:3.15.4

RUN addgroup watcher \
    && adduser -G watcher -D -h /watcher watcher

WORKDIR /watcher

COPY --from=go-builder /code/build/watcherd /usr/local/bin/watcherd
COPY --from=go-builder /code/build/syncd /usr/local/bin/syncd
USER watcher


CMD ["/usr/local/bin/watcherd"]
FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG SMALLWEB_VERSION=dev
RUN go build -ldflags="-s -w -X github.com/pomdtr/smallweb/build.Version=${SMALLWEB_VERSION}" -o smallweb

FROM denoland/deno:2.1.2
COPY --from=builder /build/smallweb /usr/local/bin/smallweb

ENV SMALLWEB_DIR=/smallweb
VOLUME ["$SMALLWEB_DIR"]

EXPOSE 7777 2222
ENTRYPOINT ["/usr/local/bin/smallweb", "up", "--cron", "--addr", ":7777", "--ssh-addr", ":2222"]

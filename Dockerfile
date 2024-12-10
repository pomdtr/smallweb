FROM golang:1.23-alpine AS builder

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

# Set entrypoint
ENV SMALLWEB_ADDR=0.0.0.0:7777
EXPOSE 7777
ENTRYPOINT ["/usr/local/bin/smallweb", "up", "--cron"]

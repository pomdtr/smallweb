FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG SMALLWEB_VERSION=dev
RUN go build -ldflags="-s -w -X github.com/pomdtr/smallweb/build.Version=${SMALLWEB_VERSION}" -o smallweb

FROM denoland/deno:2.1.2
COPY --from=builder /build/smallweb /usr/local/bin/smallweb

ARG UNAME=smallweb
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID -o $UNAME
RUN useradd --create-home -u $UID -g $GID --shell $(which bash) $UNAME

ENV SMALLWEB_DIR=/smallweb
RUN mkdir -p $SMALLWEB_DIR && chown -R smallweb:smallweb $SMALLWEB_DIR

# Switch to non-root user
USER smallweb
ENV HOME=/home/smallweb
WORKDIR $HOME

EXPOSE 7777
VOLUME ["$SMALLWEB_DIR"]

# Set entrypoint
ENV SMALLWEB_ADDR=0.0.0.0:7777
ENTRYPOINT ["/usr/local/bin/smallweb", "up", "--cron"]

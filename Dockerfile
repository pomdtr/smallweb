FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG SMALLWEB_VERSION=dev
RUN go build -ldflags="-s -w -X github.com/pomdtr/smallweb/build.Version=${SMALLWEB_VERSION}" -o smallweb

FROM debian:bookworm-slim
COPY --from=builder /build/smallweb /usr/local/bin/smallweb

RUN apt update && apt install -y git unzip curl && rm -rf /var/lib/apt/lists/*

ARG DENO_VERSION=v2.2.2
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local/deno sh -s "$DENO_VERSION"
ENV PATH="/usr/local/deno/bin:$PATH"

ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID smallweb && useradd -m -s /bin/bash -u $UID -g $GID smallweb
RUN mkdir /smallweb && chown smallweb:smallweb /smallweb

USER smallweb
VOLUME /smallweb
WORKDIR /smallweb
EXPOSE 7777 2222
ENTRYPOINT ["/usr/local/bin/smallweb"]
CMD [  "up", "--enable-crons", "--addr", ":7777", "--ssh-addr", ":2222"]

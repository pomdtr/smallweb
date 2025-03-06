FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SMALLWEB_VERSION=dev
RUN go build -ldflags="-s -w -X github.com/pomdtr/smallweb/build.Version=${SMALLWEB_VERSION}" -o smallweb

FROM debian:bookworm-slim
COPY --from=builder /build/smallweb /usr/local/bin/smallweb

# Install required packages
RUN apt update && apt install -y git unzip curl gosu && rm -rf /var/lib/apt/lists/*

# Install Deno
ARG DENO_VERSION=v2.2.2
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local/deno sh -s "$DENO_VERSION"
ENV PATH="/usr/local/deno/bin:$PATH"

# Set up default user with ID 1000
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID smallweb && useradd -m -s /bin/bash -u $GID -g $1000 smallweb

# Create app directory
RUN mkdir -p /smallweb && chown smallweb:smallweb /smallweb

# Add entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

VOLUME /smallweb
WORKDIR /smallweb
EXPOSE 7777 2222

ENTRYPOINT ["/entrypoint.sh"]
CMD ["up", "--enable-crons", "--addr", ":7777", "--ssh-addr", ":2222"]

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
ARG DENO_VERSION=v2.2.4
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local/deno sh -s "$DENO_VERSION"
ENV PATH="/usr/local/deno/bin:$PATH"

# Set up default user with ID 1000
ARG UID=1000
ARG GID=1000
RUN groupadd -g $GID smallweb && useradd -m -s /bin/bash -u $UID -g $GID smallweb


RUN mkdir -p /home/smallweb/.cache/deno
RUN chown -R smallweb:smallweb /home/smallweb/.cache/deno
VOLUME [ "/home/smallweb/.cache/deno" ]

# Create app directory
RUN mkdir -p /smallweb && chown smallweb:smallweb /smallweb
VOLUME /smallweb
WORKDIR /smallweb
ENV SMALLWEB_DIR /smallweb

EXPOSE 7777

# Add entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["up", "--enable-crons", "--addr", ":7777"]

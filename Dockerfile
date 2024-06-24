FROM golang:1.22.3 as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY cmd/ ./cmd/
COPY worker ./worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /smallweb

FROM denoland/deno:1.44.1
COPY --from=builder /smallweb /usr/local/bin/smallweb

WORKDIR /www
ENV SMALLWEB_ROOT /www

EXPOSE 7777
ENTRYPOINT [ "/usr/local/bin/smallweb" ]
CMD [ "up" ]

FROM golang:1.22.3 as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY cmd/ ./cmd/
COPY proxy ./proxy
COPY client ./client
RUN CGO_ENABLED=0 GOOS=linux go build -o /smallweb

FROM alpine:3.20 as proxy
COPY --from=builder /smallweb /usr/local/bin/smallweb
EXPOSE 8000
ENTRYPOINT [ "/usr/local/bin/smallweb" ]
CMD [ "proxy" ]

FROM denoland/deno:1.44.1
COPY --from=builder /smallweb /usr/local/bin/smallweb

USER deno
WORKDIR /home/deno/www
ENV SMALLWEB_ROOT /www

EXPOSE 8000
ENTRYPOINT [ "/usr/local/bin/smallweb" ]
CMD [ "serve" ]

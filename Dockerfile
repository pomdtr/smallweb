FROM golang:1.22.3 as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY cmd/ ./cmd/
COPY server ./server
COPY client ./client
RUN CGO_ENABLED=0 GOOS=linux go build -o /smallweb

FROM alpine:3.20
COPY --from=builder /smallweb /usr/local/bin/smallweb
ENTRYPOINT [ "/usr/local/bin/smallweb" ]
CMD [ "server" ]

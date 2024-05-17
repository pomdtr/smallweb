FROM golang:1.22.0 as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY deno/sandbox.ts ./deno/sandbox.ts
RUN CGO_ENABLED=0 GOOS=linux go build -o /smallweb

FROM denoland/deno:1.43.4
COPY --from=builder /smallweb /usr/local/bin/smallweb
ENTRYPOINT [ "/usr/local/bin/smallweb" ]

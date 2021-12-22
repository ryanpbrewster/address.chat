FROM golang:1.17-buster AS builder
RUN mkdir /workspace
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build ./cmd/api_server.go

FROM debian:buster AS runner
RUN mkdir /app
COPY --from=builder /workspace/api_server /app/api_server
ENTRYPOINT ["/app/api_server"]

FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -o /flowengine-server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /flowengine-server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]

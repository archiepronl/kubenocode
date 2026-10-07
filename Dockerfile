FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY api ./api
COPY core ./core
COPY cmd ./cmd
COPY docs ./docs
COPY internal ./internal
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' -o /out/flowengine-api ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/flowengine-api /flowengine-api
EXPOSE 8080
ENTRYPOINT ["/flowengine-api"]

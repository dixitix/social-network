FROM golang:1.25.0 AS build
WORKDIR /app

ENV GOCACHE=/tmp/go-build \
    GOMODCACHE=/go/pkg/mod

RUN mkdir -p /tmp/go-build

COPY stats-service/go.mod stats-service/go.sum ./stats-service/
COPY posts-service/go.mod posts-service/go.sum ./posts-service/

WORKDIR /app/stats-service
RUN go mod download

COPY stats-service/ ./
COPY posts-service/ ../posts-service/

RUN go build -o stats-service ./cmd/stats-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/stats-service/stats-service /app/stats-service
EXPOSE 8081 9090
ENTRYPOINT ["/app/stats-service"]

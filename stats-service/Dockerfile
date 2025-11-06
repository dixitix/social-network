FROM golang:1.25.0 AS build
WORKDIR /app
COPY stats-service/go.mod stats-service/go.sum ./stats-service/
WORKDIR /app/stats-service
RUN go mod download
COPY stats-service/ ./
RUN go build -o stats-service ./cmd/stats-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/stats-service/stats-service /app/stats-service
EXPOSE 8081 9090
ENTRYPOINT ["/app/stats-service"]

FROM golang:1.25.0 AS build
WORKDIR /app
COPY . .
RUN go build -o stats-service ./cmd/stats-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/stats-service /app/stats-service
EXPOSE 8081
ENTRYPOINT ["/app/stats-service"]

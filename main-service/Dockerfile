FROM golang:1.25.0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main-service/ ./main-service/

WORKDIR /app/main-service
RUN go build -o /app/server ./cmd/api

EXPOSE 8080
CMD ["/app/server"]

FROM golang:1.25.0

WORKDIR /app

COPY go.work ./
COPY main-service/go.mod main-service/go.sum ./main-service/
COPY posts-service/go.mod posts-service/go.sum ./posts-service/

WORKDIR /app/main-service
RUN go mod download
COPY main-service/ ./
COPY posts-service/proto ../posts-service/proto
RUN go build -o /app/server ./cmd/api

EXPOSE 8080
CMD ["/app/server"]

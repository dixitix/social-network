FROM golang:1.25.0

WORKDIR /app

COPY posts-service/go.mod posts-service/go.sum ./
RUN go mod download

COPY posts-service/ ./
RUN go build -o /app/posts-service ./cmd/posts-service

EXPOSE 50051
CMD ["/app/posts-service"]

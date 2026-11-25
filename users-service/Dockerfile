FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o users-service ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/users-service .
CMD ["./users-service"]
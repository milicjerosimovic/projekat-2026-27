FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o files-service .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/files-service .
RUN mkdir -p /data/storage
CMD ["./files-service"]
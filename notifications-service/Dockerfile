
FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o notifications-service .


FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/notifications-service .
CMD ["./notifications-service"]
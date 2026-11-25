FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api-gateway .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/api-gateway .
CMD ["./api-gateway"]
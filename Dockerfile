FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sketch-talk ./cmd/sketch-talk

FROM alpine:latest
RUN apk add --no-cache poppler-utils ca-certificates
COPY --from=builder /app/sketch-talk /usr/local/bin/sketch-talk
EXPOSE 8080
ENTRYPOINT ["sketch-talk"]

FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/statsman ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/statsman /app/statsman

ENV LISTEN_ADDR=:8080
ENV DB_PATH=/app/data/statsman.json
EXPOSE 8080
VOLUME ["/app/data"]

CMD ["/app/statsman"]

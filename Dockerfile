FROM golang:1.26-alpine as builder

WORKDIR /app
COPY . .

RUN go build -o service ./cmd/tema/...

FROM alpine

COPY --from=builder /app/service /service

CMD ["/service"]
EXPOSE 8080
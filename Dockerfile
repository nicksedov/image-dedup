FROM golang:1.26.2-alpine3.23 AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 5170

CMD ["./server"]
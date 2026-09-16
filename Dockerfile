FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o yasirbin -ldflags="-s -w" .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/yasirbin .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

RUN mkdir -p /app/data

ENV PORT=3000
ENV DB_PATH=/app/data/yasirbin.db

EXPOSE 3000

CMD ["./yasirbin"]

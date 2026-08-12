FROM golang:1.24-alpine AS builder

WORKDIR /app

# (required for HTTPS calls to Anthropic)
RUN apk --no-cache add ca-certificates

COPY go.mod ./

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o gateway .

FROM scratch

WORKDIR /

# SSL certificates from the builder to make HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/gateway /gateway

EXPOSE 8080

ENTRYPOINT ["/gateway"]
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sample-app main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

# Copy the yamlet config fetcher script
COPY yamlet-fetch-config.sh /usr/local/bin/yamlet-fetch-config
RUN chmod +x /usr/local/bin/yamlet-fetch-config

# Copy the binary
COPY --from=builder /app/sample-app .

EXPOSE 8080

# Default environment variables
ENV PORT=8080

# Use the script to fetch config and start the app
CMD ["sh", "-c", "/usr/local/bin/yamlet-fetch-config && ./sample-app"]

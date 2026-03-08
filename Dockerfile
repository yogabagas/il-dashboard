# Stage 1: Build Go Binary
FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary (CGO disabled, fully static)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main main.go

# Stage 2: Minimal image with Alpine
FROM alpine:3.19

# Install optional CA certs if needed (for HTTPS etc.)
RUN apk --no-cache add ca-certificates

WORKDIR /app

RUN adduser -D -u 10001 appuser
RUN mkdir -p /app/logs && chown appuser:appuser /app/logs && chmod 755 /app/logs
RUN mkdir -p /app/files/image && chown appuser:appuser /app/files/image && chmod 755 /app/files/image
RUN mkdir -p /app/files/video && chown appuser:appuser /app/files/video && chmod 755 /app/files/video
RUN mkdir -p /app/files/pdf && chown appuser:appuser /app/files/pdf && chmod 755 /app/files/pdf
RUN mkdir -p /app/dir && chown appuser:appuser /app/dir && chmod 755 /app/dir

COPY --from=builder /app/main .

# Copy configuration files
COPY config.properties .
COPY config-local.properties .
COPY config-dev.properties .
COPY config-prod.properties .
COPY secret ./secret

USER appuser

EXPOSE 9601
CMD ["/app/main"]

#docker network create --driver overlay kilat
#docker stack deploy -c docker-compose.yml am-switching
#docker swarm leave --force
#docker tag ariandin1411/am-switching-be:0.1 ariandin1411/am-switching-be:latest
#docker service update --image ariandin1411/am-switching-be:latest am-switching_am-switching
#docker build --platform linux/amd64 --tag ariandin1411/am-switching-be .

#docker run -d \
#  --name il-user-mgt-be \
#  --network kilatnetwork \
#  -p 9601:9601 \
#  --memory=200m \
#  --cpus="0.5" \
#  --restart=always \
#  ariandin1411/il-user-mgt-be:latest

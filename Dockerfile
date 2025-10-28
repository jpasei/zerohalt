FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o zerohalt \
    ./cmd/zerohalt


FROM nginx:alpine AS testapp

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /build/zerohalt /usr/local/bin/zerohalt

EXPOSE 80 8888

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8888/health || exit 1

CMD ["/usr/local/bin/zerohalt", "nginx", "-g", "daemon off;"]

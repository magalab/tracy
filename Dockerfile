FROM node:24-alpine AS web-builder

WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --include=dev
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /src/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/tracy-server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && mkdir -p /data

COPY --from=go-builder /out/tracy-server /usr/local/bin/tracy-server

ENV TRACY_ADDR=:8080 \
    TRACY_META_DB=/data/meta.db \
    TRACY_TRACE_DB=/data/traces.db

WORKDIR /data
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/tracy-server"]

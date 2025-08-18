# --- Build stage ---
FROM golang:1.22-alpine AS builder

# go-sqlite3 needs CGO + gcc + sqlite dev headers at build time
RUN apk add --no-cache build-base sqlite-dev ca-certificates

WORKDIR /app

# Pre-fetch modules
COPY go.mod ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build with CGO on; omit load_extension for safety on Alpine
ENV CGO_ENABLED=1
RUN go build -tags "sqlite_omit_load_extension" -o server ./cmd/server

# --- Runtime stage ---
FROM alpine:3.20

# Only the runtime libs are needed in the final image
RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

ENV PORT=8080
ENV DB_PATH=/app/data/forum.db
ENV SESSION_TTL_HOURS=24

# Copy binary and required assets/templates
COPY --from=builder /app/server ./server
COPY assets ./assets
COPY internal/views/templates ./internal/views/templates
COPY sql ./sql
COPY README.md ./README.md

EXPOSE 8080
CMD ["./server"]

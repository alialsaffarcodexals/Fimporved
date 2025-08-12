# Build stage (match your go.mod toolchain)
FROM golang:1.24.2 AS build
WORKDIR /src

# Better layer caching for deps
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

# App source
COPY . .
# Build (CGO needed for sqlite3)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o forum ./main.go

# Prepare runtime filesystem we will copy into the distroless image
RUN mkdir -p /runtime/internal /runtime/static /runtime/data && \
    cp -R ./internal /runtime/internal && \
    cp -R ./static /runtime/static
    # /runtime/data stays empty (just the directory)

# ---- Runtime image ----
FROM gcr.io/distroless/base-debian12
WORKDIR /app

# Binary + assets
COPY --from=build /src/forum /app/forum
COPY --from=build /runtime/internal /app/internal
COPY --from=build /runtime/static /app/static

# Writable data dir for sqlite; give ownership to non-root user
COPY --from=build --chown=65532:65532 /runtime/data /data

EXPOSE 8080
USER 65532:65532

# NOTE: point -templates to /app/internal/... and -data to /data
ENTRYPOINT ["/app/forum","-addr",":8080","-templates","/app/internal/web/templates","-data","/data"]

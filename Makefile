\
SHELL := /bin/bash

.PHONY: tidy build run test docker-build docker-run docker-run-ps

tidy:
\tgo mod tidy

build:
\tCGO_ENABLED=1 go build -o bin/server ./cmd/server

run:
\tgo run ./cmd/server

test:
\tgo test ./...

docker-build:
\tdocker build -t forum .

# POSIX / Git Bash
docker-run:
\tdocker run --name forum --rm -p 8080:8080 -v "${PWD}/data:/app/data" forum

# Windows PowerShell variant (invoke: make docker-run-ps)
docker-run-ps:
\tpowershell -Command "docker run --name forum --rm -p 8080:8080 -v \"$((Get-Location).Path)\\data:/app/data\" forum"

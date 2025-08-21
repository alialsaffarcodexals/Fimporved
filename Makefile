# Simple Makefile for the forum project

.PHONY: build run

build:
	go build ./...

run:
	go run ./cmd/server

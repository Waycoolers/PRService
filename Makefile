.PHONY: up down build

up:
	docker-compose up --build

down:
	docker-compose down

build:
	go build -o bin/PRService ./cmd/server

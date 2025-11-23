.PHONY: up down build

up:
	docker-compose up --build

down:
	docker-compose down

build:
	go build -o bin/PRService ./cmd/server

test:
	docker compose -f docker-compose.test.yml down --volumes --remove-orphans
	docker network prune -f
	docker compose -f docker-compose.test.yml up --abort-on-container-exit --build
	docker compose -f docker-compose.test.yml down --volumes --remove-orphans

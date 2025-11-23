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

lint:
	golangci-lint run -v ./...

tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.4.0
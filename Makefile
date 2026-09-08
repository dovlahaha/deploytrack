DB_URL ?= postgres://deploytrack:deploytrack@localhost:5432/deploytrack?sslmode=disable
TEST_RESULTS ?= /tmp/test-results

.PHONY: db test test-unit run build docker fmt

db:
	docker compose up -d db
	./scripts/wait-for-db.sh localhost 5432

test-unit:
	go test ./internal/metrics/...

test:
	mkdir -p $(TEST_RESULTS)
	TEST_DATABASE_URL="$(DB_URL)" \
		gotestsum --junitfile $(TEST_RESULTS)/gotestsum-report.xml -- ./...

run:
	DATABASE_URL="$(DB_URL)" go run ./cmd/api

build:
	go build ./...

docker:
	docker build -t deploytrack:local .

fmt:
	go fmt ./...
	go vet ./...

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	dotenvx run -- go run ./cmd/api

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	go build -o ./bin/api ./cmd/api

## audit: tidy and vendor dependencies and format, vet and test all code
.PHONY: audit
audit:
	go mod tidy
	go mod verify
	go fmt ./...
	go vet ./...
	staticcheck ./...
	CGO_ENABLED=1 dotenvx run -- go test -race -vet=off ./...

## test/e2e: build the API binary and force a real e2e run
.PHONY: test/e2e
test/e2e: build/api
	dotenvx run -- go test -count=1 ./e2e/

## gen/mocks: regenerate mocks from go:generate directives
.PHONY: gen/mocks
gen/mocks:
	go generate ./internal/data

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	dotenvx run -- sh -c 'psql $${DB_DSN}'

## db/migrate/create name=$1: create a new database migration
.PHONY: db/migrate/create
db/migrate/create:
	migrate create -ext=.sql -dir=./migrations -seq ${name}

## db/migrate/up: apply all up database migrations
.PHONY: db/migrate/up
db/migrate/up:
	dotenvx run -- sh -c 'migrate -path=./migrations -database=$${DB_DSN} up'

## db/migrate/down: resolve all up database migrations
.PHONY: db/migrate/down
db/migrate/down:
	dotenvx run -- sh -c 'migrate -path=./migrations -database=$${DB_DSN} down'

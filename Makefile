postgres:
	docker container run --network bank-net --name postgres -e POSTGRES_PASSWORD=secret -e POSTGRES_USER=root -d -p 5432:5432 postgres:14

createdb:
	docker container exec -it postgres createdb --username=root --owner=root simple_bank

dropdb:
	docker container exec -it postgres dropdb simple_bank

migrateup:
	migrate -path configs/db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up

migratedown:
	migrate -path configs/db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down

migrateup1:
	migrate -path configs/db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up 1

migratedown1:
	migrate -path configs/db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 1

test.integration.container:
	docker compose -f deployments/docker-compose.test.yaml up --build --attach api
	docker compose -f deployments/docker-compose.test.yaml down

test.integration:
	go test -p 1 -count 1 -cover -coverprofile cover.out -tags=integration ./...

test.integration.api:
	go test -p 1 -count 1 -cover -coverprofile cover.out -tags=integration ./cmd/httpserver

test.integration.repo:
	go test -p 1 -count 1 -cover -coverprofile cover.out -tags=integration ./internal/*repo

test.unit:
	go test -count 1 -cover -coverprofile cover.out ./...

test.api:
	go test -count 1 -tags=integration ./cmd/httpserver/tests 

server:
	go run cmd/main.go

dev.app.up:
	docker compose  -f deployments/docker-compose.dev.yaml up --build

dev.app.down:
	docker compose -f deployments/docker-compose.dev.yaml down

countloc:
	find . -type f -not -path "./vendor*" -not -path "*/\.*" -not -path "./docs*" | xargs wc -l

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 server composeappdown


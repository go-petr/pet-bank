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

sqlc:
	sqlc generate

test:
	go test -cover ./...

server:
	go run cmd/server/main.go

composeappup:
	docker-compose -f deployments/docker-compose.yaml up --build

composeappdown:
	 docker-compose -f deployments/docker-compose.yaml down

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 server composeappdown


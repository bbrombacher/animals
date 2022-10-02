LOCAL_PSQL_URL=postgres://pguser:pgpass@localhost:9001/shelters?sslmode=disable

db.setup:
	docker-compose up -d db
	sleep 5
	make migrate.up

db.reset:
	make db.delete
	make db.setup

db.delete:
	docker stop animal-shelter-data
	docker rm animal-shelter-data

migrate.up:
	migrate -path migrations -database $(LOCAL_PSQL_URL) -verbose up

migrate.down:
	migrate -path migrations -database $(LOCAL_PSQL_URL) -verbose down

mocks:
	go generate ./...
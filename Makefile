postgres:
	docker run --name greenlight-postgres -p 5432:5432 -e POSTGRES_USER=greenlight -e POSTGRES_PASSWORD=greenlight1234 -d postgres:17-alpine

start-docker-container:
	docker run greenlight-postgres

stop-docker-container:
	docker stop greenlight-postgres

createdb:
	docker exec -it greenlight-postgres createdb -U greenlight -O  greenlight
# meaning of the above command: docker execute -iterative container-name createdb -username=root -owner=root database-name

dropdb:
	docker exec -it greenlight-postgres17 dropdb greenlight

new_migration:
	migrate create -ext sql -dir migrations -seq $(name)

migrateup:
	migrate -path migrations -database "postgresql://greenlight:greenlight1234@localhost:5432/greenlight?sslmode=disable" -verbose up

migratedown:
	migrate -path migrations -database "postgresql://greenlight:greenlight1234@localhost:5432/greenlight?sslmode=disable" -verbose down

migrateup1:
	migrate -path migrations -database "postgresql://greenlight:greenlight1234@localhost:5432/greenlight?sslmode=disable" -verbose up 1

migratedown1:
	migrate -path migrations -database "postgresql://greenlight:greenlight1234@localhost:5432/greenlight?sslmode=disable" -verbose down 1

.PHONY: postgres start-docker-container start-docker-container createdb dropdb new_migration migrateup1 migratedown1

# note
# a subuser(a non super-user) was created to the postgres db with the following command:
# code: CREATE ROLE greenlight_user WITH LOGIN PASSWORD 'pa55word'
# the aim of this user is to be able to migratedb schemas to the postgres, and to not be able to delete the db (the super user is "greenlight" with the password "greenlight1234")

# I also added a postgres extension CITEXT, based on the tutors instructions
# code: CREATE EXTENSION IF NOT EXISTS citext
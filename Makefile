
include .env
#====================================================#
#HELPERS
#=====================================================#


## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^//'

confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

#==============================================================#
#DEVELOPMENT
#==============================================================#


## postgres: create a new postgres container
postgres:
	docker run --name greenlight-postgres -p 5432:5432 -e POSTGRES_USER=greenlight -e POSTGRES_PASSWORD=greenlight1234 -d postgres:17-alpine

## start-docker-container: start the greenlight postgres container
start-docker-container:
	docker run greenlight-postgres

## stop-docker-container: stop the greenlight postgres container
stop-docker-container:
	docker stop greenlight-postgres

## createdb: create the greenlight database
createdb:
	docker exec -it greenlight-postgres createdb -U greenlight -O  greenlight
# meaning of the above command: docker execute -iterative container-name createdb -username=root -owner=root database-name

## dropdb: drop the greenlight database
dropdb:
	docker exec -it greenlight-postgres17 dropdb greenlight

# with the following command.i have to enter the name with the command, that is "make new_migration name=migration_name"
# new_migration:
# 	@echo 'Creating new migration files for ${name}...'; \
# 	migrate create -ext sql -dir migrations -seq $(name)

# with the method below, the command line will as me for the migration name   

## new/migration: create a new migration file
new/migration: confirm
	@read -p "Enter migration name: " name; \
	echo "Creating new migration files for $$name..."; \
	migrate create -ext sql -dir migrations -seq $$name

## migrate/up: run up migrations
migrate/up: confirm
	@echo 'Running up migrations...'
	migrate -path migrations -database ${DB_SOURCE} -verbose up

## migrate/down: run down migrations
migrate/down: confirm
	@echo 'Running down migrations...'
	migrate -path migrations -database ${DB_SOURCE} -verbose down

## migrate/up1: run the next up migration
migrate/up1:
	migrate -path migrations -database "postgresql://greenlight:greenlight1234@localhost:5432/greenlight?sslmode=disable" -verbose up 1

## migrate/down1: run down the previous migration 
migrate/down1:
	migrate -path migrations -database "postgresql://greenlight:greenlight1234@localhost:5432/greenlight?sslmode=disable" -verbose down 1

## run: run the api server
run:
	go run ./cmd/api

#================================================================#

#QUALITY CONTROL
#================================================================#

## audit: tidy dependencies and format,vet, and test all code
audit:
	@echo 'Tidying and verifying dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

#==============================================================#
#BUILD
#==============================================================#

## build: build the cmd/api application
build:
	@echo 'Building the api server...'
	go build -ldflags="-s" -o ./bin/api ./cmd/api

.PHONY: postgres start-docker-container start-docker-container createdb dropdb new_migration migrateup1 migratedown1 run audit build

# note
# a subuser(a non super-user) was created to the postgres db with the following command:
# code: CREATE ROLE greenlight_user WITH LOGIN PASSWORD 'pa55word'
# the aim of this user is to be able to migratedb schemas to the postgres, and to not be able to delete the db (the super user is "greenlight" with the password "greenlight1234")

# I also added a postgres extension CITEXT, based on the tutors instructions
# code: CREATE EXTENSION IF NOT EXISTS citext
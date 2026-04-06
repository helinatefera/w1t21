.PHONY: dev-backend dev-frontend build up down migrate seed create-admin

# Development
dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

# Docker
build:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down

# Database
migrate:
	@echo "Migrations run automatically on server startup"

seed:
	docker-compose exec postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql

create-admin:
	@echo "Creating initial administrator account..."
	@read -p "Username: " user; \
	read -p "Display name: " name; \
	read -sp "Password (min 12 chars): " pass; echo; \
	curl -sf -X POST http://localhost:8080/api/setup/admin \
		-H 'Content-Type: application/json' \
		-d "{\"username\":\"$$user\",\"password\":\"$$pass\",\"display_name\":\"$$name\"}" \
	&& echo "\nAdministrator created successfully." \
	|| echo "\nFailed — setup may already be complete or the password does not meet requirements."

# Keys
generate-keys:
	./scripts/generate-keys.sh

generate-keyfile:
	./scripts/generate-keys.sh --keyfile secrets.enc

build-keyfile-tool:
	cd backend && go build -o ../bin/keyfile-tool ./cmd/keyfile-tool

# Backend
backend-deps:
	cd backend && go mod tidy && go mod vendor

# Frontend
frontend-deps:
	cd frontend && npm install

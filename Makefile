.PHONY: up up-backend up-frontend down logs build-backend

up: up-backend up-frontend

up-backend:
	docker compose up -d backend

up-frontend:
	npm install
	npm run dev

down:
	docker compose down

logs:
	docker compose logs -f backend

build-backend:
	docker compose build backend

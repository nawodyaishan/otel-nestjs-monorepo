.PHONY: install dev build test up down clean

install:
	pnpm install

dev:
	pnpm run dev

build:
	pnpm run build

test:
	pnpm run test

lint:
	pnpm run lint

up:
	docker compose up --build -d

down:
	docker compose down -v

clean:
	pnpm turbo run clean
	rm -rf node_modules apps/*/node_modules packages/*/node_modules

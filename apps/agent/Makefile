.PHONY: ci format-check lint type-check test build

format-check:
	uv run ruff format --check .

lint:
	uv run ruff check .

type-check:
	uv run mypy

test:
	uv run pytest

build:
	uv build

ci: format-check lint type-check test build

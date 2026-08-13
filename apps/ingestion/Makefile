.PHONY: build ci format format-check lint lock-check test test-integration type-check verify-persistence

build:
	uv build

format:
	uv run ruff format .

format-check:
	uv run ruff format --check .

lint:
	uv run ruff check .

lock-check:
	uv lock --check

test:
	uv run pytest

test-integration:
	uv run pytest -m integration

type-check:
	uv run mypy

verify-persistence:
	uv run norvii-verify-persistence

ci: lock-check format-check lint type-check test build

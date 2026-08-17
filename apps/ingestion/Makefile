.PHONY: build ci coverage coverage-integration format format-check lint lock-check test test-integration type-check verify-persistence

build:
	uv build

coverage:
	uv run pytest --cov=norvii_ingestion --cov-report=xml:coverage.xml

coverage-integration:
	uv run pytest -m "" --cov=norvii_ingestion --cov-report=xml:coverage.xml

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

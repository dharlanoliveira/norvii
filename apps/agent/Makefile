.PHONY: ci format-check lint type-check test test-integration build

INTEGRATION_ENVIRONMENT := python ../../infra/scripts/run-with-environment.py ../../infra/.env

format-check:
	uv run ruff format --check .

lint:
	uv run ruff check .

type-check:
	uv run mypy

test:
	uv run pytest

test-integration:
	$(INTEGRATION_ENVIRONMENT) uv run pytest -m integration

build:
	uv build

ci: format-check lint type-check test build

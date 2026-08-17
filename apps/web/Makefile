.PHONY: ci coverage format-check lint test typecheck build bundle-check e2e

ci: format-check lint typecheck test build e2e

coverage:
	npm run test:coverage

format-check:
	npm run format:check

lint:
	npm run lint

typecheck:
	npm run typecheck

test:
	npm run test

build:
	npm run build

bundle-check:
	npm run check:bundle

e2e:
	npm run test:e2e

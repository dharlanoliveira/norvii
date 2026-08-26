# Tasks: MCP Research Tools and Skills

## Phase 1: Setup

- [X] T001 Add the official MCP SDK and `norvii-mcp` entry point in `apps/agent/pyproject.toml` [FR-017]
- [X] T002 Add MCP contract and Docker operation documentation in `specs/010-mcp-research-tools/contracts/` and `specs/010-mcp-research-tools/quickstart.md` [FR-014, FR-017]

## Phase 2: Foundation

- [X] T003 Write bounded snapshot-aware read repository tests in `apps/agent/tests/unit/mcp/test_research.py` [FR-003-FR-010]
- [X] T004 Implement typed read-only MCP research services in `apps/agent/src/norvii_agent/mcp/research.py` [FR-003-FR-010]
- [X] T005 Implement Streamable HTTP and stdio MCP server composition in `apps/agent/src/norvii_agent/mcp/server.py` and `apps/agent/src/norvii_agent/mcp/__main__.py` [FR-001, FR-017, FR-020]

## Phase 3: User Story 1

- [X] T006 [US1] Implement catalog, document, article, metadata, and search tools in `apps/agent/src/norvii_agent/mcp/server.py` [FR-001-FR-006]
- [X] T007 [US1] Add SDK-backed discovery and corpus-isolation tests in `apps/agent/tests/unit/mcp/test_server.py` [SC-001-SC-004]

## Phase 4: User Story 2

- [X] T008 [US2] Implement bounded graph, related-article, and provision-comparison tools in `apps/agent/src/norvii_agent/mcp/server.py` [FR-007-FR-010]

## Phase 5: User Story 3

- [X] T009 [US3] Add reusable evidence, comparison, and citation-verification prompts in `apps/agent/src/norvii_agent/mcp/server.py` [FR-011-FR-012]

## Phase 6: Containerization and Validation

- [X] T010 Add the MCP Docker image, health endpoint, private-network service, and localhost-only port mapping in `apps/agent/Dockerfile`, `infra/compose.yaml`, and `infra/.env.example` [FR-018-FR-020]
- [X] T011 Run agent CI, contract validation, language validation, Docker Compose validation, and `git diff --check`; record results in `specs/010-mcp-research-tools/quickstart.md` [SC-006-SC-008]

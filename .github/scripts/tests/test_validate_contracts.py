from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT_DIRECTORY = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPT_DIRECTORY))

from validate_contracts import ContractValidationError, ContractValidator  # noqa: E402


class ContractValidatorTests(unittest.TestCase):
    def setUp(self) -> None:
        self._temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self._temporary_directory.cleanup)
        self._root = Path(self._temporary_directory.name)
        self._validator = ContractValidator()

    def test_accepts_versioned_openapi_and_ingestion_work_contracts(self) -> None:
        contract_directory = self._root / "contracts" / "corpus-ingestion" / "v1"
        self._write_json(
            contract_directory / "openapi.json",
            {
                "openapi": "3.1.0",
                "info": {"version": "1.0.0"},
                "paths": {
                    "/corpora": {
                        "get": {
                            "operationId": "listCorpora",
                            "responses": {"200": {"description": "OK"}},
                        }
                    }
                },
                "components": {"schemas": {}},
            },
        )
        self._write_json(
            contract_directory / "ingestion-work.schema.json",
            {
                "$schema": "https://json-schema.org/draft/2020-12/schema",
                "$id": "https://norvii.dev/contracts/corpus-ingestion/v1/ingestion-work.schema.json",
                "oneOf": [
                    {"$ref": "#/$defs/claim"},
                    {"$ref": "#/$defs/publication"},
                    {"$ref": "#/$defs/failure"},
                ],
                "$defs": {
                    "claim": {"type": "object"},
                    "publication": {"type": "object"},
                    "failure": {"type": "object"},
                },
            },
        )

        self._validator.validate(contract_directory)

    def test_rejects_duplicate_openapi_operation_ids(self) -> None:
        contract_directory = self._copy_repository_contracts()
        openapi_path = contract_directory / "openapi.json"
        document = json.loads(openapi_path.read_text(encoding="utf-8"))
        document["paths"]["/duplicate"] = {
            "get": {
                "operationId": "listCorpora",
                "responses": {"200": {"description": "OK"}},
            }
        }
        self._write_json(openapi_path, document)

        with self.assertRaisesRegex(ContractValidationError, "operationId"):
            self._validator.validate(contract_directory)

    def test_rejects_unresolved_local_references(self) -> None:
        contract_directory = self._copy_repository_contracts()
        openapi_path = contract_directory / "openapi.json"
        document = json.loads(openapi_path.read_text(encoding="utf-8"))
        document["paths"]["/broken"] = {
            "get": {
                "operationId": "brokenReference",
                "responses": {
                    "200": {
                        "description": "Broken",
                        "content": {
                            "application/json": {
                                "schema": {"$ref": "#/components/schemas/Missing"}
                            }
                        },
                    }
                },
            }
        }
        self._write_json(openapi_path, document)

        with self.assertRaisesRegex(ContractValidationError, "reference"):
            self._validator.validate(contract_directory)

    def test_accepts_grounded_chat_schema_and_fixtures(self) -> None:
        directory = Path(__file__).resolve().parents[3] / "specs" / "005-grounded-rag-chat" / "contracts"

        self._validator.validate_grounded_chat(directory)

    def test_rejects_a_grounded_chat_fixture_that_is_not_invalid(self) -> None:
        directory = self._copy_grounded_chat_contracts()
        fixture = directory / "fixtures" / "invalid" / "unexpected-valid.json"
        fixture.parent.mkdir(parents=True, exist_ok=True)
        fixture.write_text('{"type":"started","requestId":"request-id","corpusId":"corpus-id"}', encoding="utf-8")

        with self.assertRaisesRegex(ContractValidationError, "invalid fixture"):
            self._validator.validate_grounded_chat(directory)

    def _copy_repository_contracts(self) -> Path:
        source = Path(__file__).resolve().parents[3] / "contracts" / "corpus-ingestion" / "v1"
        destination = self._root / "contracts" / "corpus-ingestion" / "v1"
        destination.mkdir(parents=True)
        for name in ("openapi.json", "ingestion-work.schema.json"):
            (destination / name).write_bytes((source / name).read_bytes())
        return destination

    def _copy_grounded_chat_contracts(self) -> Path:
        source = Path(__file__).resolve().parents[3] / "specs" / "005-grounded-rag-chat" / "contracts"
        destination = self._root / "grounded-chat"
        for path in source.rglob("*.json"):
            target = destination / path.relative_to(source)
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(path.read_bytes())
        return destination

    @staticmethod
    def _write_json(path: Path, value: object) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(value), encoding="utf-8")


if __name__ == "__main__":
    unittest.main()

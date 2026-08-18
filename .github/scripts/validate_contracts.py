from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any, Iterator


class ContractValidationError(RuntimeError):
    """Raised when a versioned contract violates repository invariants."""


class ContractValidator:
    _HTTP_CONTRACT = "openapi.json"
    _WORK_CONTRACT = "ingestion-work.schema.json"

    def validate(self, contract_directory: Path) -> None:
        openapi = self._load_object(contract_directory / self._HTTP_CONTRACT)
        work_schema = self._load_object(contract_directory / self._WORK_CONTRACT)
        self._validate_openapi(openapi)
        self._validate_work_schema(work_schema)

    def _validate_openapi(self, document: dict[str, Any]) -> None:
        if document.get("openapi") != "3.1.0":
            raise ContractValidationError("OpenAPI contracts must use version 3.1.0")
        info = document.get("info")
        if not isinstance(info, dict) or info.get("version") != "1.0.0":
            raise ContractValidationError("The v1 OpenAPI contract must declare version 1.0.0")

        operation_ids: set[str] = set()
        paths = document.get("paths")
        if not isinstance(paths, dict):
            raise ContractValidationError("The OpenAPI contract must define paths")
        for value in self._walk_values(paths):
            if not isinstance(value, dict) or "operationId" not in value:
                continue
            operation_id = value["operationId"]
            if not isinstance(operation_id, str) or not operation_id:
                raise ContractValidationError("Every operationId must be a non-empty string")
            if operation_id in operation_ids:
                raise ContractValidationError(f"Duplicate operationId: {operation_id}")
            operation_ids.add(operation_id)

        self._validate_local_references(document)

    def _validate_work_schema(self, document: dict[str, Any]) -> None:
        schema_id = document.get("$id")
        if not isinstance(schema_id, str) or "/v1/" not in schema_id:
            raise ContractValidationError("The ingestion work schema must have a versioned $id")
        definitions = document.get("$defs")
        if not isinstance(definitions, dict):
            raise ContractValidationError("The ingestion work schema must define $defs")
        expected_references = {
            "#/$defs/claim",
            "#/$defs/publication",
            "#/$defs/failure",
        }
        variants = document.get("oneOf")
        if not isinstance(variants, list):
            raise ContractValidationError("The ingestion work schema must define oneOf")
        actual_references = {
            variant.get("$ref") for variant in variants if isinstance(variant, dict)
        }
        if actual_references != expected_references:
            raise ContractValidationError(
                "The ingestion work schema must define claim, publication, and failure variants"
            )
        self._validate_local_references(document)

    def _validate_local_references(self, document: dict[str, Any]) -> None:
        for value in self._walk_values(document):
            if not isinstance(value, dict):
                continue
            reference = value.get("$ref")
            if not isinstance(reference, str) or not reference.startswith("#/"):
                continue
            if not self._resolve_pointer(document, reference):
                raise ContractValidationError(f"Unresolved local reference: {reference}")

    @staticmethod
    def _resolve_pointer(document: dict[str, Any], reference: str) -> bool:
        current: Any = document
        for raw_token in reference[2:].split("/"):
            token = raw_token.replace("~1", "/").replace("~0", "~")
            if not isinstance(current, dict) or token not in current:
                return False
            current = current[token]
        return True

    @classmethod
    def _walk_values(cls, value: Any) -> Iterator[Any]:
        yield value
        if isinstance(value, dict):
            for child in value.values():
                yield from cls._walk_values(child)
        elif isinstance(value, list):
            for child in value:
                yield from cls._walk_values(child)

    @staticmethod
    def _load_object(path: Path) -> dict[str, Any]:
        try:
            value = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as error:
            raise ContractValidationError(f"Unable to load {path}: {error}") from error
        if not isinstance(value, dict):
            raise ContractValidationError(f"Contract root must be an object: {path}")
        return value


def main() -> int:
    root = Path(__file__).resolve().parents[2]
    directory = root / "contracts" / "corpus-ingestion" / "v1"
    try:
        ContractValidator().validate(directory)
    except ContractValidationError as error:
        print(f"Contract validation failed: {error}", file=sys.stderr)
        return 1
    print("Contract validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

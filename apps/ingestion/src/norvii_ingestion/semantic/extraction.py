"""Model adapter that produces validated semantic artifacts from bounded legal units."""

from __future__ import annotations

import hashlib
import json
import re
import time
import urllib.request
from dataclasses import dataclass
from typing import TYPE_CHECKING, Protocol, cast
from urllib.error import URLError
from urllib.request import Request
from uuid import UUID, uuid5

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256

if TYPE_CHECKING:
    from collections.abc import Iterable, Sequence

_EXTRACTION_NAMESPACE = UUID("fdd2031e-b41e-4554-a4a1-68a535c42943")
_ENTITY_TYPES = frozenset({"concept", "actor", "right", "obligation"})
_RELATIONSHIP_TYPES = frozenset(
    {"defines", "applies_to", "grants", "requires", "protects", "governs"}
)
_MAX_UNITS_PER_REQUEST = 12
_MAX_REQUESTS_PER_DOCUMENT = 32
_MAX_UNIT_CHARACTERS = 4_000
_MAX_LABEL_CHARACTERS = 240
_NORMALIZED_LABEL = re.compile(r"[^a-z0-9]+")


class ExtractionProviderError(RuntimeError):
    """Indicate a failed or malformed semantic extraction provider response."""


@dataclass(frozen=True, slots=True)
class SemanticEntity:
    """One evidence-backed legal semantic entity."""

    id: UUID
    evidence_unit_id: UUID
    entity_type: str
    label: str
    normalized_label: str


@dataclass(frozen=True, slots=True)
class SemanticRelationship:
    """One evidence-backed relationship between extracted entities."""

    id: UUID
    subject_entity_id: UUID
    object_entity_id: UUID
    evidence_unit_id: UUID
    relationship_type: str
    qualifier: str | None = None


@dataclass(frozen=True, slots=True)
class SemanticExtraction:
    """Immutable output and operational facts from one document extraction run."""

    id: UUID
    extraction_version: str
    model_identifier: str
    input_sha256: Sha256
    input_tokens: int | None
    output_tokens: int | None
    duration_milliseconds: int
    entities: tuple[SemanticEntity, ...]
    relationships: tuple[SemanticRelationship, ...]


class SemanticExtractor(Protocol):
    """Extract bounded semantic artifacts from a normalized immutable document."""

    def extract(self, artifact: DocumentArtifact) -> SemanticExtraction:
        """Return evidence-backed semantic artifacts for one document version."""
        ...


@dataclass(frozen=True, slots=True)
class OpenAICompatibleSemanticExtractor:
    """Call a JSON-only chat endpoint without retaining prompts or legal content."""

    endpoint: str
    api_key: str
    model: str
    timeout_seconds: int = 30
    reasoning_effort: str = "medium"
    extraction_version: str = "legal-semantic-v1"

    def __post_init__(self) -> None:
        """Reject incomplete configuration before an ingestion lease is claimed."""
        if not self.endpoint.strip() or not self.api_key.strip() or not self.model.strip():
            raise ValueError("semantic extraction endpoint, API key, and model are required")
        if self.timeout_seconds <= 0 or not self.extraction_version.strip():
            raise ValueError("semantic extraction configuration is invalid")

    def extract(self, artifact: DocumentArtifact) -> SemanticExtraction:
        """Extract only supported entities and relationships from bounded legal locations."""
        selected = tuple(_selected_units(artifact))
        started = time.perf_counter()
        entity_by_key = {
            (entity.evidence_unit_id, entity.entity_type, entity.normalized_label): entity
            for entity in _structural_entities(selected)
        }
        relationships: dict[UUID, SemanticRelationship] = {}
        input_tokens: int | None = 0
        output_tokens: int | None = 0
        for unit_batch in _batches(selected, _MAX_UNITS_PER_REQUEST):
            payload = self._request(unit_batch, artifact)
            batch_entities, batch_relationships, usage = _validated_batch(payload, unit_batch)
            for entity in batch_entities:
                key = (entity.evidence_unit_id, entity.entity_type, entity.normalized_label)
                entity_by_key[key] = entity
            for relationship in batch_relationships:
                relationships[relationship.id] = relationship
            input_tokens = _sum_usage(input_tokens, usage[0])
            output_tokens = _sum_usage(output_tokens, usage[1])
        entities = tuple(sorted(entity_by_key.values(), key=lambda entity: str(entity.id)))
        relationship_values = tuple(sorted(relationships.values(), key=lambda item: str(item.id)))
        return SemanticExtraction(
            id=uuid5(_EXTRACTION_NAMESPACE, f"{artifact.text_sha256}:{self.extraction_version}"),
            extraction_version=self.extraction_version,
            model_identifier=self.model,
            input_sha256=Sha256.from_bytes(_selection_bytes(selected, artifact)),
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            duration_milliseconds=max(0, round((time.perf_counter() - started) * 1000)),
            entities=entities,
            relationships=relationship_values,
        )

    def _request(self, units: Sequence[DocumentUnit], artifact: DocumentArtifact) -> object:
        content = [
            {
                "unitId": str(unit.id),
                "locator": unit.locator,
                "text": _unit_text(unit, artifact),
            }
            for unit in units
        ]
        body = json.dumps(
            {
                "model": self.model,
                "reasoning_effort": self.reasoning_effort,
                "response_format": {"type": "json_object"},
                "messages": [
                    {
                        "role": "system",
                        "content": (
                            "Extract only directly supported legal semantics. Return JSON with an "
                            "entities array and relationships array. Each entity has unitId, type "
                            "(concept|actor|right|obligation), and label. Each relationship has "
                            "unitId, type (defines|applies_to|grants|requires|protects|governs), "
                            "subject (an entity label), object (an entity label), and optional "
                            "qualifier. Do not infer facts. Do not include text not supported by "
                            "a provided unit."
                        ),
                    },
                    {"role": "user", "content": json.dumps({"units": content})},
                ],
            }
        ).encode("utf-8")
        request = Request(  # noqa: S310 - configured endpoint is deployment-owned
            self.endpoint,
            data=body,
            headers={"Authorization": f"Bearer {self.api_key}", "Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:  # noqa: S310
                decoded = json.loads(response.read())
        except (URLError, TimeoutError, json.JSONDecodeError) as error:
            raise ExtractionProviderError("semantic extraction provider request failed") from error
        return _completion_content(decoded)


def _selected_units(artifact: DocumentArtifact) -> Iterable[DocumentUnit]:
    candidates = tuple(
        unit
        for unit in artifact.units
        if unit.kind in {UnitKind.ARTICLE, UnitKind.SECTION, UnitKind.CHAPTER, UnitKind.TITLE}
        and artifact.text[unit.start_offset : unit.end_offset].strip()
    )
    selected = candidates or (artifact.units[0],)
    return selected[: _MAX_UNITS_PER_REQUEST * _MAX_REQUESTS_PER_DOCUMENT]


def _batches(items: Sequence[DocumentUnit], size: int) -> Iterable[tuple[DocumentUnit, ...]]:
    for start in range(0, len(items), size):
        yield tuple(items[start : start + size])


def _structural_entities(units: Sequence[DocumentUnit]) -> tuple[SemanticEntity, ...]:
    """Expose document locations as deterministic graph anchors, not model-derived facts."""
    return tuple(
        SemanticEntity(
            id=uuid5(_EXTRACTION_NAMESPACE, f"{unit.id}:location:{_normalize_label(unit.locator)}"),
            evidence_unit_id=unit.id,
            entity_type="location",
            label=unit.locator,
            normalized_label=_normalize_label(unit.locator),
        )
        for unit in units
    )


def _selection_bytes(units: Sequence[DocumentUnit], artifact: DocumentArtifact) -> bytes:
    digest = hashlib.sha256()
    for unit in units:
        digest.update(str(unit.id).encode("utf-8"))
        digest.update(str(unit.content_sha256).encode("utf-8"))
        digest.update(_unit_text(unit, artifact).encode("utf-8"))
    return digest.hexdigest().encode("ascii")


def _completion_content(payload: object) -> object:
    if not isinstance(payload, dict):
        raise ExtractionProviderError("semantic extraction response is malformed")
    try:
        content = _message_content(payload)
        return {"content": json.loads(content), "usage": payload.get("usage")}
    except json.JSONDecodeError as error:
        raise ExtractionProviderError("semantic extraction response is malformed") from error


def _message_content(payload: dict[object, object]) -> str:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices or not isinstance(choices[0], dict):
        raise ExtractionProviderError("semantic extraction response is malformed")
    message = choices[0].get("message")
    if not isinstance(message, dict) or not isinstance(message.get("content"), str):
        raise ExtractionProviderError("semantic extraction response is malformed")
    return cast("str", message["content"])


def _validated_batch(
    payload: object, units: Sequence[DocumentUnit]
) -> tuple[
    tuple[SemanticEntity, ...],
    tuple[SemanticRelationship, ...],
    tuple[int | None, int | None],
]:
    if not isinstance(payload, dict) or not isinstance(payload.get("content"), dict):
        raise ExtractionProviderError("semantic extraction response is malformed")
    content = payload["content"]
    unit_by_id = {str(unit.id): unit for unit in units}
    entities = _entities(content.get("entities"), unit_by_id)
    by_label = {entity.label.casefold(): entity for entity in entities}
    relationships = _relationships(content.get("relationships"), unit_by_id, by_label)
    return entities, relationships, _usage(payload.get("usage"))


def _entities(value: object, units: dict[str, DocumentUnit]) -> tuple[SemanticEntity, ...]:
    if not isinstance(value, list):
        raise ExtractionProviderError("semantic entities are malformed")
    result: dict[UUID, SemanticEntity] = {}
    for item in value:
        if not isinstance(item, dict):
            raise ExtractionProviderError("semantic entity is malformed")
        unit = units.get(str(item.get("unitId", "")))
        entity_type = item.get("type")
        label = item.get("label")
        if unit is None or not isinstance(entity_type, str) or entity_type not in _ENTITY_TYPES:
            raise ExtractionProviderError("semantic entity references an invalid unit or type")
        normalized = _normalize_label(label)
        entity = SemanticEntity(
            id=uuid5(_EXTRACTION_NAMESPACE, f"{unit.id}:{entity_type}:{normalized}"),
            evidence_unit_id=unit.id,
            entity_type=entity_type,
            label=_label(label),
            normalized_label=normalized,
        )
        result[entity.id] = entity
    return tuple(result.values())


def _relationships(
    value: object,
    units: dict[str, DocumentUnit],
    entities_by_label: dict[str, SemanticEntity],
) -> tuple[SemanticRelationship, ...]:
    if not isinstance(value, list):
        raise ExtractionProviderError("semantic relationships are malformed")
    result: dict[UUID, SemanticRelationship] = {}
    for item in value:
        if not isinstance(item, dict):
            raise ExtractionProviderError("semantic relationship is malformed")
        unit = units.get(str(item.get("unitId", "")))
        relationship_type = item.get("type")
        subject = entities_by_label.get(_label(item.get("subject")).casefold())
        object_ = entities_by_label.get(_label(item.get("object")).casefold())
        if (
            unit is None
            or not isinstance(relationship_type, str)
            or relationship_type not in _RELATIONSHIP_TYPES
            or subject is None
            or object_ is None
            or subject.id == object_.id
        ):
            raise ExtractionProviderError("semantic relationship is not evidence-backed")
        qualifier = item.get("qualifier")
        if qualifier is not None and not isinstance(qualifier, str):
            raise ExtractionProviderError("semantic relationship qualifier is malformed")
        relationship = SemanticRelationship(
            id=uuid5(
                _EXTRACTION_NAMESPACE,
                f"{unit.id}:{subject.id}:{object_.id}:{relationship_type}:{qualifier or ''}",
            ),
            subject_entity_id=subject.id,
            object_entity_id=object_.id,
            evidence_unit_id=unit.id,
            relationship_type=relationship_type,
            qualifier=_qualifier(qualifier),
        )
        result[relationship.id] = relationship
    return tuple(result.values())


def _label(value: object) -> str:
    if (
        not isinstance(value, str)
        or not value.strip()
        or len(value.strip()) > _MAX_LABEL_CHARACTERS
    ):
        raise ExtractionProviderError("semantic label is invalid")
    return value.strip()


def _normalize_label(value: object) -> str:
    normalized = _NORMALIZED_LABEL.sub(" ", _label(value).casefold()).strip()
    if not normalized:
        raise ExtractionProviderError("semantic label is invalid")
    return normalized


def _unit_text(unit: DocumentUnit, artifact: DocumentArtifact) -> str:
    """Limit one model input span while preserving the immutable unit identity."""
    end_offset = min(unit.end_offset, unit.start_offset + _MAX_UNIT_CHARACTERS)
    return artifact.text[unit.start_offset : end_offset]


def _qualifier(value: object) -> str | None:
    if value is None:
        return None
    if not isinstance(value, str):
        raise ExtractionProviderError("semantic relationship qualifier is malformed")
    return value.strip() or None


def _usage(value: object) -> tuple[int | None, int | None]:
    if not isinstance(value, dict):
        return None, None
    return _usage_value(value.get("prompt_tokens")), _usage_value(value.get("completion_tokens"))


def _usage_value(value: object) -> int | None:
    return value if isinstance(value, int) and not isinstance(value, bool) and value >= 0 else None


def _sum_usage(total: int | None, value: int | None) -> int | None:
    if total is None or value is None:
        return None
    return total + value

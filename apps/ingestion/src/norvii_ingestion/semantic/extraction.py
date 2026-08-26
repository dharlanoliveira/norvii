"""Model adapter that produces validated semantic artifacts from bounded legal units."""

from __future__ import annotations

import hashlib
import json
import math
import re
import time
import urllib.request
from dataclasses import dataclass
from typing import TYPE_CHECKING, Protocol, cast
from urllib.error import HTTPError, URLError
from urllib.request import Request
from uuid import UUID, uuid5

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256

if TYPE_CHECKING:
    from collections.abc import Iterable, Sequence

_EXTRACTION_NAMESPACE = UUID("fdd2031e-b41e-4554-a4a1-68a535c42943")
_ENTITY_TYPES = frozenset({"concept", "actor", "right", "obligation", "condition"})
_RELATIONSHIP_TYPES = frozenset(
    {
        "defines",
        "applies_to",
        "grants",
        "protects",
        "must_be_observed_by",
        "imposes_duty_on",
        "assigns_responsibility_to",
        "conditions",
    }
)
# Semantic output must remain small enough to be reliably JSON-validated. The
# POC intentionally samples opening legal locations instead of attempting an
# unbounded document-wide graph extraction in a single ingestion run.
# A single addressable legal unit per request keeps the provider's bounded JSON
# response attributable and small. The document budget remains capped at eight
# sampled units and eight provider calls.
_MAX_UNITS_PER_REQUEST = 1
_MAX_REQUESTS_PER_DOCUMENT = 8
_MAX_UNIT_CHARACTERS = 4_000
_MAX_ENTITIES_PER_UNIT = 4
_MAX_ASSERTIONS_PER_UNIT = 4
_MAX_COMPLETION_TOKENS = 1_600
_MAX_RESPONSE_ATTEMPTS = 2
_MAX_PROVIDER_REQUEST_SECONDS = 30
_MAX_LABEL_CHARACTERS = 240
_NORMALIZED_LABEL = re.compile(r"[^a-z0-9]+")
_REQUEST_FAILED = "semantic extraction provider request failed"
_MALFORMED_RESPONSE = "semantic extraction response is malformed"


class ExtractionProviderError(RuntimeError):
    """Indicate a failed or malformed semantic extraction provider response."""

    def __init__(
        self,
        message: str,
        detail: str = "provider_response_invalid",
        diagnostic: ProviderResponseDiagnostic | None = None,
    ) -> None:
        super().__init__(message)
        self.detail = detail
        self.diagnostic = diagnostic


class SemanticDiagnosticLogger(Protocol):
    """Persist safe provider-response diagnostics outside semantic artifacts."""

    def failure(self, event: str, **fields: object) -> None:
        """Record a safe structured failure event."""
        ...


@dataclass(frozen=True, slots=True)
class ProviderResponseDiagnostic:
    """Retain response metadata without retaining the response content."""

    code: str
    response_byte_count: int
    response_sha256: str
    response_content_type: str | None
    provider_request_id: str | None
    json_error_line: int | None = None
    json_error_column: int | None = None
    json_error_offset: int | None = None
    completion_byte_count: int | None = None
    completion_sha256: str | None = None


@dataclass(frozen=True, slots=True)
class ProviderResponse:
    """Pair one decoded provider response with safe diagnostic metadata."""

    payload: object
    diagnostic: ProviderResponseDiagnostic


@dataclass(frozen=True, slots=True)
class SemanticEntity:
    """One evidence-backed legal semantic entity."""

    id: UUID
    evidence_unit_id: UUID
    entity_type: str
    label: str
    normalized_label: str


@dataclass(frozen=True, slots=True)
class SemanticAssertion:
    """One atomic legal assertion established and evidenced by legal units."""

    id: UUID
    subject_entity_id: UUID
    object_entity_id: UUID
    establishing_unit_id: UUID
    evidence_unit_id: UUID
    predicate: str
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
    assertions: tuple[SemanticAssertion, ...]


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
    reasoning_effort: str = "none"
    extraction_version: str = "legal-semantic-v3"
    diagnostic_logger: SemanticDiagnosticLogger | None = None

    def __post_init__(self) -> None:
        """Reject incomplete configuration before an ingestion lease is claimed."""
        if not self.endpoint.strip() or not self.api_key.strip() or not self.model.strip():
            raise ValueError("semantic extraction endpoint, API key, and model are required")
        if self.timeout_seconds <= 0 or not self.extraction_version.strip():
            raise ValueError("semantic extraction configuration is invalid")

    def extract(self, artifact: DocumentArtifact) -> SemanticExtraction:
        """Extract only supported entities and assertions from bounded legal locations."""
        selected = tuple(_selected_units(artifact))
        started = time.perf_counter()
        entity_by_key: dict[tuple[UUID, str, str], SemanticEntity] = {}
        assertions: dict[UUID, SemanticAssertion] = {}
        input_tokens: int | None = 0
        output_tokens: int | None = 0
        for unit_batch in _batches(selected, _MAX_UNITS_PER_REQUEST):
            remaining_timeout_seconds = _remaining_timeout_seconds(started, self.timeout_seconds)
            if remaining_timeout_seconds is None:
                raise ExtractionProviderError(
                    "semantic extraction document budget was exhausted", "provider_timeout"
                )
            timeout_seconds = min(remaining_timeout_seconds, _MAX_PROVIDER_REQUEST_SECONDS)
            try:
                payload = self._request(unit_batch, artifact, timeout_seconds)
            except ExtractionProviderError as error:
                if error.detail != "provider_response_invalid":
                    raise
                continue
            batch_entities, batch_assertions, usage = _validated_batch(payload, unit_batch)
            for entity in batch_entities:
                key = (entity.evidence_unit_id, entity.entity_type, entity.normalized_label)
                entity_by_key[key] = entity
            for assertion in batch_assertions:
                assertions[assertion.id] = assertion
            input_tokens = _sum_usage(input_tokens, usage[0])
            output_tokens = _sum_usage(output_tokens, usage[1])
        entities = tuple(sorted(entity_by_key.values(), key=lambda entity: str(entity.id)))
        assertion_values = tuple(sorted(assertions.values(), key=lambda item: str(item.id)))
        return SemanticExtraction(
            id=uuid5(_EXTRACTION_NAMESPACE, f"{artifact.text_sha256}:{self.extraction_version}"),
            extraction_version=self.extraction_version,
            model_identifier=self.model,
            input_sha256=Sha256.from_bytes(_selection_bytes(selected, artifact)),
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            duration_milliseconds=max(0, round((time.perf_counter() - started) * 1000)),
            entities=entities,
            assertions=assertion_values,
        )

    def _request(
        self,
        units: Sequence[DocumentUnit],
        artifact: DocumentArtifact,
        timeout_seconds: int,
    ) -> object:
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
                "max_completion_tokens": _MAX_COMPLETION_TOKENS,
                "response_format": {"type": "json_object"},
                "messages": [
                    {
                        "role": "system",
                        "content": (
                            "Extract only directly supported legal semantics. Return JSON with an "
                            "entities array and assertions array. Each entity has unitId, type "
                            "(concept|actor|right|obligation|condition), and label. "
                            "Each assertion has "
                            "establishingUnitId, evidenceUnitId, predicate, subject (an entity "
                            "label), object (an entity label), and optional qualifier. Both unit "
                            "identifiers must be provided and must name a supplied unit. Use one "
                            "of these directed predicates "
                            "types only: defines (term to legal definition); applies_to (norm to "
                            "covered person, activity, or situation); grants (norm to granted "
                            "right or beneficiary); protects (norm to protected right); "
                            "must_be_observed_by (norm to public body required to observe it); "
                            "imposes_duty_on (norm to obligated actor); "
                            "assigns_responsibility_to (norm to responsible actor); conditions "
                            "(legal consequence or obligation to its condition). Do not use "
                            "requires or governs. Do not infer facts. Do not include text not "
                            "supported by a provided unit. Each entity label must name exactly "
                            "one legally addressable referent. Decompose a coordinated list of "
                            "independent actors, rights, obligations, concepts, or conditions "
                            "into one entity and one assertion per member; never emit a "
                            "comma-separated aggregate entity. Keep a collective as one entity "
                            "only when the source treats it as one indivisible legal subject. "
                            f"Return at most {_MAX_ENTITIES_PER_UNIT} entities and "
                            f"{_MAX_ASSERTIONS_PER_UNIT} assertions for each supplied unit."
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
        for attempt in range(_MAX_RESPONSE_ATTEMPTS):
            try:
                decoded = self._response_body(request, timeout_seconds)
                return _completion_content(decoded)
            except ExtractionProviderError as error:
                if error.detail == "provider_response_invalid":
                    self._record_invalid_response(error, attempt + 1, len(units))
                if (
                    error.detail != "provider_response_invalid"
                    or attempt + 1 == _MAX_RESPONSE_ATTEMPTS
                ):
                    raise
        raise AssertionError("semantic response attempts must either return or raise")

    @staticmethod
    def _response_body(request: Request, timeout_seconds: int) -> ProviderResponse:
        try:
            with urllib.request.urlopen(request, timeout=timeout_seconds) as response:  # noqa: S310
                body = response.read()
                diagnostic = _response_diagnostic(response, body)
                try:
                    payload = json.loads(body)
                except json.JSONDecodeError as error:
                    raise ExtractionProviderError(
                        _MALFORMED_RESPONSE,
                        "provider_response_invalid",
                        _with_json_error(diagnostic, "response_body_invalid_json", error),
                    ) from error
                return ProviderResponse(payload, diagnostic)
        except HTTPError as error:
            detail = f"provider_http_status_{error.code}"
            raise ExtractionProviderError(_REQUEST_FAILED, detail) from error
        except URLError as error:
            detail = (
                "provider_timeout"
                if isinstance(error.reason, TimeoutError)
                else "provider_transport"
            )
            raise ExtractionProviderError(_REQUEST_FAILED, detail) from error
        except TimeoutError as error:
            raise ExtractionProviderError(_REQUEST_FAILED, "provider_timeout") from error

    def _record_invalid_response(
        self,
        error: ExtractionProviderError,
        response_attempt: int,
        unit_count: int,
    ) -> None:
        diagnostic = error.diagnostic
        if self.diagnostic_logger is None or diagnostic is None:
            return
        self.diagnostic_logger.failure(
            "semantic_provider_response_invalid",
            diagnostic_code=diagnostic.code,
            response_byte_count=diagnostic.response_byte_count,
            response_sha256=diagnostic.response_sha256,
            response_content_type=diagnostic.response_content_type,
            provider_request_id=diagnostic.provider_request_id,
            json_error_line=diagnostic.json_error_line,
            json_error_column=diagnostic.json_error_column,
            json_error_offset=diagnostic.json_error_offset,
            completion_byte_count=diagnostic.completion_byte_count,
            completion_sha256=diagnostic.completion_sha256,
            provider_response_attempt=response_attempt,
            unit_count=unit_count,
        )


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


def _remaining_timeout_seconds(started: float, document_budget_seconds: int) -> int | None:
    elapsed = time.perf_counter() - started
    remaining = document_budget_seconds - elapsed
    return max(1, math.ceil(remaining)) if remaining > 0 else None


def _selection_bytes(units: Sequence[DocumentUnit], artifact: DocumentArtifact) -> bytes:
    digest = hashlib.sha256()
    for unit in units:
        digest.update(str(unit.id).encode("utf-8"))
        digest.update(str(unit.content_sha256).encode("utf-8"))
        digest.update(_unit_text(unit, artifact).encode("utf-8"))
    return digest.hexdigest().encode("ascii")


def _completion_content(response: ProviderResponse) -> object:
    if not isinstance(response.payload, dict):
        raise ExtractionProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
    try:
        content = _message_content(response.payload)
        return {"content": json.loads(content), "usage": response.payload.get("usage")}
    except json.JSONDecodeError as error:
        completion = content.encode("utf-8")
        raise ExtractionProviderError(
            _MALFORMED_RESPONSE,
            "provider_response_invalid",
            _with_json_error(
                response.diagnostic,
                "completion_content_invalid_json",
                error,
                completion,
            ),
        ) from error


def _response_diagnostic(response: object, body: bytes) -> ProviderResponseDiagnostic:
    return ProviderResponseDiagnostic(
        code="response_valid_json",
        response_byte_count=len(body),
        response_sha256=hashlib.sha256(body).hexdigest(),
        response_content_type=_header(response, "content-type"),
        provider_request_id=(
            _header(response, "x-request-id")
            or _header(response, "request-id")
            or _header(response, "openai-request-id")
        ),
    )


def _header(response: object, name: str) -> str | None:
    headers = getattr(response, "headers", None)
    if headers is None:
        return None
    value = headers.get(name)
    return value.strip()[:256] if isinstance(value, str) and value.strip() else None


def _with_json_error(
    diagnostic: ProviderResponseDiagnostic,
    code: str,
    error: json.JSONDecodeError,
    completion: bytes | None = None,
) -> ProviderResponseDiagnostic:
    return ProviderResponseDiagnostic(
        code=code,
        response_byte_count=diagnostic.response_byte_count,
        response_sha256=diagnostic.response_sha256,
        response_content_type=diagnostic.response_content_type,
        provider_request_id=diagnostic.provider_request_id,
        json_error_line=error.lineno,
        json_error_column=error.colno,
        json_error_offset=error.pos,
        completion_byte_count=len(completion) if completion is not None else None,
        completion_sha256=hashlib.sha256(completion).hexdigest()
        if completion is not None
        else None,
    )


def _message_content(payload: dict[object, object]) -> str:
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices or not isinstance(choices[0], dict):
        raise ExtractionProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
    message = choices[0].get("message")
    if not isinstance(message, dict) or not isinstance(message.get("content"), str):
        raise ExtractionProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
    return cast("str", message["content"])


def _validated_batch(
    payload: object, units: Sequence[DocumentUnit]
) -> tuple[
    tuple[SemanticEntity, ...],
    tuple[SemanticAssertion, ...],
    tuple[int | None, int | None],
]:
    if not isinstance(payload, dict) or not isinstance(payload.get("content"), dict):
        raise ExtractionProviderError(_MALFORMED_RESPONSE, "provider_response_schema_invalid")
    content = payload["content"]
    unit_by_id = {str(unit.id): unit for unit in units}
    entities = _entities(content.get("entities"), unit_by_id)
    by_label = {entity.normalized_label: entity for entity in entities}
    assertions = _assertions(content.get("assertions"), unit_by_id, by_label)
    return entities, assertions, _usage(payload.get("usage"))


def _entities(value: object, units: dict[str, DocumentUnit]) -> tuple[SemanticEntity, ...]:
    if not isinstance(value, list):
        return ()
    result: dict[UUID, SemanticEntity] = {}
    for item in value:
        if not isinstance(item, dict):
            continue
        unit = units.get(str(item.get("unitId", "")))
        entity_type = item.get("type")
        label = item.get("label")
        if unit is None or not isinstance(entity_type, str) or entity_type not in _ENTITY_TYPES:
            continue
        try:
            labels = _atomic_labels(label)
        except ExtractionProviderError:
            continue
        for entity_label in labels:
            normalized = _normalize_label(entity_label)
            entity = SemanticEntity(
                id=uuid5(_EXTRACTION_NAMESPACE, f"{unit.id}:{entity_type}:{normalized}"),
                evidence_unit_id=unit.id,
                entity_type=entity_type,
                label=entity_label,
                normalized_label=normalized,
            )
            result[entity.id] = entity
    return tuple(result.values())


def _assertions(
    value: object,
    units: dict[str, DocumentUnit],
    entities_by_label: dict[str, SemanticEntity],
) -> tuple[SemanticAssertion, ...]:
    if not isinstance(value, list):
        return ()
    result: dict[tuple[UUID, UUID, UUID, UUID, str], SemanticAssertion] = {}
    for item in value:
        assertions = _assertions_from_item(item, units, entities_by_label)
        if not assertions:
            continue
        for assertion in assertions:
            key = (
                assertion.subject_entity_id,
                assertion.object_entity_id,
                assertion.establishing_unit_id,
                assertion.evidence_unit_id,
                assertion.predicate,
            )
            result.setdefault(key, assertion)
    return tuple(result.values())


def _assertions_from_item(
    value: object,
    units: dict[str, DocumentUnit],
    entities_by_label: dict[str, SemanticEntity],
) -> tuple[SemanticAssertion, ...]:
    """Return valid atomic assertions emitted by the provider."""
    if not isinstance(value, dict):
        return ()
    try:
        subject_labels = _atomic_labels(value.get("subject"))
        object_labels = _atomic_labels(value.get("object"))
    except ExtractionProviderError:
        return ()
    establishing_unit = units.get(str(value.get("establishingUnitId", "")))
    evidence_unit = units.get(str(value.get("evidenceUnitId", "")))
    predicate = value.get("predicate")
    subjects = tuple(entities_by_label.get(_normalize_label(label)) for label in subject_labels)
    objects = tuple(entities_by_label.get(_normalize_label(label)) for label in object_labels)
    if (
        establishing_unit is None
        or evidence_unit is None
        or not isinstance(predicate, str)
        or predicate not in _RELATIONSHIP_TYPES
        or any(entity is None for entity in (*subjects, *objects))
    ):
        return ()
    qualifier = value.get("qualifier")
    if qualifier is not None and not isinstance(qualifier, str):
        qualifier = None
    return tuple(
        SemanticAssertion(
            id=uuid5(
                _EXTRACTION_NAMESPACE,
                (
                    f"{establishing_unit.id}:{evidence_unit.id}:{subject.id}:"
                    f"{object_.id}:{predicate}:{qualifier or ''}"
                ),
            ),
            subject_entity_id=subject.id,
            object_entity_id=object_.id,
            establishing_unit_id=establishing_unit.id,
            evidence_unit_id=evidence_unit.id,
            predicate=predicate,
            qualifier=_qualifier(qualifier),
        )
        for subject in subjects
        for object_ in objects
        if subject is not None and object_ is not None and subject.id != object_.id
    )


def _atomic_labels(value: object) -> tuple[str, ...]:
    """Split explicit coordinated lists while preserving indivisible single labels."""
    label = _label(value)
    parts = tuple(
        part.strip()
        for part in re.split(r"[,;]|\b(?:and|e)\b", label, flags=re.IGNORECASE)
        if part.strip()
    )
    return parts if len(parts) > 1 else (label,)


def _label(value: object) -> str:
    if (
        not isinstance(value, str)
        or not value.strip()
        or len(value.strip()) > _MAX_LABEL_CHARACTERS
    ):
        raise ExtractionProviderError(
            "semantic label is invalid", "provider_response_schema_invalid"
        )
    return value.strip()


def _normalize_label(value: object) -> str:
    normalized = _NORMALIZED_LABEL.sub(" ", _label(value).casefold()).strip()
    if not normalized:
        raise ExtractionProviderError(
            "semantic label is invalid", "provider_response_schema_invalid"
        )
    return normalized


def _unit_text(unit: DocumentUnit, artifact: DocumentArtifact) -> str:
    """Limit one model input span while preserving the immutable unit identity."""
    end_offset = min(unit.end_offset, unit.start_offset + _MAX_UNIT_CHARACTERS)
    return artifact.text[unit.start_offset : end_offset]


def _qualifier(value: object) -> str | None:
    if value is None:
        return None
    if not isinstance(value, str):
        raise ExtractionProviderError(
            "normative assertion qualifier is malformed", "provider_response_schema_invalid"
        )
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

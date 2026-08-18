"""Runtime adapters for the language-neutral ingestion-work v1 contract."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import TYPE_CHECKING, Self
from uuid import UUID

from norvii_ingestion.domain.models import Sha256

if TYPE_CHECKING:
    from collections.abc import Mapping
    from typing import Any

_MAX_FAILURE_DETAIL_LENGTH = 500


class ContractError(ValueError):
    """Report an invalid cross-module ingestion payload."""


@dataclass(frozen=True, slots=True)
class ClaimPayload:
    """Validated v1 work claim payload."""

    work_id: UUID
    corpus_id: UUID
    source_id: UUID
    source_kind: str
    reason: str
    lease_token: UUID
    lease_expires_at: datetime

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> Self:
        """Validate and decode a claim mapping."""
        _expect_kind(value, "claim")
        source_kind = _string(value, "sourceKind")
        reason = _string(value, "reason")
        if source_kind not in {"url", "pdf"}:
            raise ContractError("sourceKind must be url or pdf")
        if reason not in {"initial", "retry", "reprocess"}:
            raise ContractError("reason is not supported")
        return cls(
            work_id=_uuid(value, "workId"),
            corpus_id=_uuid(value, "corpusId"),
            source_id=_uuid(value, "sourceId"),
            source_kind=source_kind,
            reason=reason,
            lease_token=_uuid(value, "leaseToken"),
            lease_expires_at=_datetime(value, "leaseExpiresAt"),
        )


@dataclass(frozen=True, slots=True)
class PublicationUnitPayload:
    """Validated v1 publication unit payload."""

    id: UUID
    parent_id: UUID | None
    kind: str
    ordinal: int
    start_offset: int
    end_offset: int
    locator: str
    content_sha256: Sha256

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> Self:
        """Validate and decode one publication unit mapping."""
        parent_value = value.get("parentId")
        parent_id = None if parent_value is None else _parse_uuid(parent_value, "parentId")
        return cls(
            id=_uuid(value, "id"),
            parent_id=parent_id,
            kind=_string(value, "kind"),
            ordinal=_integer(value, "ordinal"),
            start_offset=_integer(value, "startOffset"),
            end_offset=_integer(value, "endOffset"),
            locator=_string(value, "locator"),
            content_sha256=Sha256(_string(value, "contentSha256")),
        )


@dataclass(frozen=True, slots=True)
class PublicationPayload:
    """Validated v1 atomic publication payload."""

    work_id: UUID
    lease_token: UUID
    pipeline_version: str
    origin_sha256: Sha256
    text: str
    text_sha256: Sha256
    units: tuple[PublicationUnitPayload, ...]

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> Self:
        """Validate and decode a publication mapping."""
        _expect_kind(value, "publication")
        raw_units = value.get("units")
        if not isinstance(raw_units, list) or not raw_units:
            raise ContractError("units must be a non-empty array")
        units: list[PublicationUnitPayload] = []
        for raw_unit in raw_units:
            if not isinstance(raw_unit, dict):
                raise ContractError("every publication unit must be an object")
            units.append(PublicationUnitPayload.from_mapping(raw_unit))
        return cls(
            work_id=_uuid(value, "workId"),
            lease_token=_uuid(value, "leaseToken"),
            pipeline_version=_string(value, "pipelineVersion"),
            origin_sha256=Sha256(_string(value, "originSha256")),
            text=_string(value, "text"),
            text_sha256=Sha256(_string(value, "textSha256")),
            units=tuple(units),
        )


@dataclass(frozen=True, slots=True)
class FailurePayload:
    """Validated v1 categorized failure payload."""

    work_id: UUID
    lease_token: UUID
    category: str
    detail: str | None

    @classmethod
    def from_mapping(cls, value: Mapping[str, Any]) -> Self:
        """Validate and decode a failure mapping."""
        _expect_kind(value, "failure")
        detail = value.get("detail")
        if detail is not None and not isinstance(detail, str):
            raise ContractError("detail must be a string or null")
        if isinstance(detail, str) and len(detail) > _MAX_FAILURE_DETAIL_LENGTH:
            raise ContractError("detail must contain no more than 500 characters")
        return cls(
            work_id=_uuid(value, "workId"),
            lease_token=_uuid(value, "leaseToken"),
            category=_string(value, "category"),
            detail=detail,
        )


def _expect_kind(value: Mapping[str, Any], expected: str) -> None:
    if _string(value, "kind") != expected:
        raise ContractError(f"kind must be {expected}")


def _string(value: Mapping[str, Any], key: str) -> str:
    item = value.get(key)
    if not isinstance(item, str) or not item:
        raise ContractError(f"{key} must be a non-empty string")
    return item


def _integer(value: Mapping[str, Any], key: str) -> int:
    item = value.get(key)
    if not isinstance(item, int) or isinstance(item, bool) or item < 0:
        raise ContractError(f"{key} must be a nonnegative integer")
    return item


def _uuid(value: Mapping[str, Any], key: str) -> UUID:
    return _parse_uuid(value.get(key), key)


def _parse_uuid(value: object, key: str) -> UUID:
    if not isinstance(value, str):
        raise ContractError(f"{key} must be a UUID string")
    try:
        parsed = UUID(value)
    except ValueError as error:
        raise ContractError(f"{key} must be a UUID string") from error
    if parsed.int == 0:
        raise ContractError(f"{key} must not be a nil UUID")
    return parsed


def _datetime(value: Mapping[str, Any], key: str) -> datetime:
    text = _string(value, key)
    try:
        parsed = datetime.fromisoformat(text)
    except ValueError as error:
        raise ContractError(f"{key} must be an RFC 3339 timestamp") from error
    if parsed.tzinfo is None:
        raise ContractError(f"{key} must include a timezone")
    return parsed


__all__ = [
    "ClaimPayload",
    "ContractError",
    "FailurePayload",
    "PublicationPayload",
    "PublicationUnitPayload",
]

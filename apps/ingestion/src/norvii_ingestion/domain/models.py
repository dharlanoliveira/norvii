"""Immutable acquisition, work, revision, and failure value objects."""

from __future__ import annotations

import hashlib
import re
from dataclasses import dataclass
from enum import StrEnum
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from datetime import datetime
    from typing import Self
    from uuid import UUID

_SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
_MAX_FAILURE_DETAIL_LENGTH = 500


class Sha256(str):
    """A canonical lowercase hexadecimal SHA-256 digest."""

    __slots__ = ()

    def __new__(cls, value: str) -> Self:
        """Validate and construct a canonical digest."""
        if not _SHA256_PATTERN.fullmatch(value):
            raise ValueError("SHA-256 must contain 64 lowercase hexadecimal characters")
        return super().__new__(cls, value)

    @classmethod
    def from_bytes(cls, value: bytes) -> Self:
        """Hash a bounded byte sequence into a canonical digest."""
        return cls(hashlib.sha256(value).hexdigest())


class SourceKind(StrEnum):
    """Supported immutable source origin types."""

    URL = "url"
    PDF = "pdf"


class WorkReason(StrEnum):
    """Explicit reasons for bounded ingestion work."""

    INITIAL = "initial"
    RETRY = "retry"
    REPROCESS = "reprocess"


class FailureCategory(StrEnum):
    """Safe failure categories shared with public contracts."""

    UNSAFE_URL = "unsafe_url"
    PAYLOAD_TOO_LARGE = "payload_too_large"
    UNSUPPORTED_CONTENT = "unsupported_content"
    ACQUISITION_FAILED = "acquisition_failed"
    EXTRACTION_FAILED = "extraction_failed"
    PUBLICATION_FAILED = "publication_failed"
    LEASE_EXPIRED = "lease_expired"
    INTERNAL_ERROR = "internal_error"


@dataclass(frozen=True, slots=True)
class WorkClaim:
    """One leased queue claim owned by an opaque lease token."""

    work_id: UUID
    corpus_id: UUID
    source_id: UUID
    source_kind: SourceKind
    reason: WorkReason
    lease_token: UUID
    lease_expires_at: datetime

    def __post_init__(self) -> None:
        """Reject missing identity or timezone-naive lease values."""
        if any(
            value.int == 0
            for value in (
                self.work_id,
                self.corpus_id,
                self.source_id,
                self.lease_token,
            )
        ):
            raise ValueError("work claim identifiers must not be nil")
        if self.lease_expires_at.tzinfo is None:
            raise ValueError("lease expiry must be timezone-aware")


@dataclass(frozen=True, slots=True)
class IngestionWork:
    """A leased claim plus its corpus and immutable origin processing input."""

    claim: WorkClaim
    attempt_id: UUID
    corpus_language: str
    url: str | None
    pdf_content: bytes | None

    def __post_init__(self) -> None:
        """Require one origin matching the claim discriminant."""
        if self.attempt_id.int == 0 or self.corpus_language not in {"en", "pt"}:
            raise ValueError("attempt identity and supported corpus language are required")
        if self.claim.source_kind is SourceKind.URL and self.url is None:
            raise ValueError("URL work requires an HTTPS origin")
        if self.claim.source_kind is SourceKind.PDF and self.pdf_content is None:
            raise ValueError("PDF work requires preserved content")


@dataclass(frozen=True, slots=True)
class SafeFailure:
    """A categorized bounded failure that excludes sensitive payloads."""

    category: FailureCategory
    detail: str | None = None

    def __post_init__(self) -> None:
        """Bound optional operational detail retained with an attempt."""
        if self.detail is not None and len(self.detail) > _MAX_FAILURE_DETAIL_LENGTH:
            raise ValueError("failure detail must contain no more than 500 characters")


@dataclass(frozen=True, slots=True)
class OriginCapture:
    """Immutable bounded metadata for one acquired origin."""

    content_sha256: Sha256
    captured_at: datetime
    media_type: str
    byte_size: int
    final_url: str | None = None

    def __post_init__(self) -> None:
        """Validate capture metadata independently from its transport."""
        if self.captured_at.tzinfo is None:
            raise ValueError("capture time must be timezone-aware")
        if not self.media_type.strip() or self.byte_size <= 0:
            raise ValueError("capture media type and positive byte size are required")


@dataclass(frozen=True, slots=True)
class SourceRevision:
    """An immutable source capture associated with one successful attempt."""

    id: UUID
    source_id: UUID
    attempt_id: UUID
    capture: OriginCapture
    pipeline_version: str

    def __post_init__(self) -> None:
        """Require stable non-empty revision identity and pipeline provenance."""
        if self.id.int == 0 or self.source_id.int == 0 or self.attempt_id.int == 0:
            raise ValueError("revision identifiers must not be nil")
        if not self.pipeline_version.strip():
            raise ValueError("pipeline version is required")

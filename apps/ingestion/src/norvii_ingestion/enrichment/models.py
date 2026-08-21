"""Immutable enrichment-stage values shared by chunking and publication."""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_ingestion.domain.models import Sha256


@dataclass(frozen=True, slots=True)
class RetrievalChunk:
    """One immutable bounded source span and its optional enrichment vector."""

    id: UUID
    source_unit_id: UUID
    context_locator: str
    start_offset: int
    end_offset: int
    text: str
    text_sha256: Sha256
    embedding: tuple[float, ...] | None = None
    embedding_model: str | None = None

"""Legal-aware retrieval chunking with stable source offsets."""

from __future__ import annotations

import hashlib
from uuid import UUID, uuid5

from norvii_ingestion.domain.artifacts import DocumentArtifact, DocumentUnit, UnitKind
from norvii_ingestion.domain.models import Sha256
from norvii_ingestion.enrichment.models import RetrievalChunk

_CHUNK_NAMESPACE = UUID("a63c6dd0-34a8-4c8e-9b6d-8ae6f3f49b1e")


class LegalChunker:
    """Split complete legal units without losing article-level context."""

    def __init__(self, max_characters: int = 1200) -> None:
        if max_characters < 1:
            raise ValueError("max_characters must be positive")
        self._max_characters = max_characters

    def chunk(self, artifact: DocumentArtifact) -> tuple[RetrievalChunk, ...]:
        """Return stable, bounded chunks covering the normalized document in order."""
        artifact.validate()
        units_by_id = {unit.id: unit for unit in artifact.units}
        roots = [unit for unit in artifact.units if unit.kind is UnitKind.ARTICLE]
        if not roots:
            roots = [unit for unit in artifact.units if unit.kind is UnitKind.DOCUMENT]
        chunks: list[RetrievalChunk] = []
        for unit in sorted(roots, key=lambda item: item.start_offset):
            context = self._article_context(unit, units_by_id)
            chunks.extend(self._split_unit(artifact.text, unit, context))
        return tuple(chunks)

    @staticmethod
    def _article_context(unit: DocumentUnit, units_by_id: dict[UUID, DocumentUnit]) -> str:
        current = unit
        while current.parent_id is not None:
            current = units_by_id[current.parent_id]
        if unit.kind is UnitKind.ARTICLE:
            return unit.locator
        return current.locator

    def _split_unit(
        self,
        text: str,
        unit: DocumentUnit,
        context_locator: str,
    ) -> list[RetrievalChunk]:
        chunks: list[RetrievalChunk] = []
        start = unit.start_offset
        while start < unit.end_offset:
            target = min(start + self._max_characters, unit.end_offset)
            end = target
            if target < unit.end_offset:
                boundary = text.rfind(" ", start + 1, target + 1)
                if boundary > start:
                    end = boundary
            content = text[start:end]
            identity = f"{unit.id}:{start}:{end}:{hashlib.sha256(content.encode()).hexdigest()}"
            chunks.append(
                RetrievalChunk(
                    id=uuid5(_CHUNK_NAMESPACE, identity),
                    source_unit_id=unit.id,
                    context_locator=context_locator,
                    start_offset=start,
                    end_offset=end,
                    text=content,
                    text_sha256=Sha256(hashlib.sha256(content.encode()).hexdigest()),
                ),
            )
            start = end
        return chunks

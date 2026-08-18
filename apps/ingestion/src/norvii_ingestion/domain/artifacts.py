"""Immutable document artifacts and publication validation."""

from __future__ import annotations

import hashlib
from dataclasses import dataclass
from enum import StrEnum
from itertools import pairwise
from typing import TYPE_CHECKING

from norvii_ingestion.domain.models import Sha256

if TYPE_CHECKING:
    from uuid import UUID


class UnitKind(StrEnum):
    """Recognized legal hierarchy and deterministic fallback unit kinds."""

    DOCUMENT = "document"
    TITLE = "title"
    CHAPTER = "chapter"
    SECTION = "section"
    ARTICLE = "article"
    PARAGRAPH = "paragraph"
    ITEM = "item"
    RECITAL = "recital"
    PAGE = "page"
    BLOCK = "block"


@dataclass(frozen=True, slots=True)
class DocumentUnit:
    """One addressable span in an immutable complete document."""

    id: UUID
    parent_id: UUID | None
    kind: UnitKind
    ordinal: int
    marker: str | None
    label: str | None
    start_offset: int
    end_offset: int
    start_page: int | None
    end_page: int | None
    locator: str
    content_sha256: Sha256


@dataclass(frozen=True, slots=True)
class DocumentArtifact:
    """Complete normalized text plus its canonical addressable hierarchy."""

    text: str
    text_sha256: Sha256
    units: tuple[DocumentUnit, ...]

    def validate(self) -> None:
        """Validate content hashes, identity, bounds, hierarchy, and peer order."""
        self._validate_document_identity()
        units_by_id = self._index_units()
        self._validate_root()

        children_by_parent: dict[UUID, list[DocumentUnit]] = {}
        for unit in self.units:
            self._validate_unit(unit, units_by_id)
            if unit.parent_id is not None:
                children_by_parent.setdefault(unit.parent_id, []).append(unit)

        self._validate_sibling_order(children_by_parent)
        self._reject_cycles(units_by_id)

    def _validate_document_identity(self) -> None:
        if not self.text:
            raise ValueError("document text must not be empty")
        if self.text_sha256 != _hash_text(self.text):
            raise ValueError("document text hash does not match normalized text")
        if not self.units:
            raise ValueError("document must contain at least one unit")

    def _index_units(self) -> dict[UUID, DocumentUnit]:
        units_by_id = {unit.id: unit for unit in self.units}
        if len(units_by_id) != len(self.units):
            raise ValueError("document unit identifiers must be unique")
        locators = {unit.locator for unit in self.units}
        if len(locators) != len(self.units) or "" in locators:
            raise ValueError("document unit locators must be non-empty and unique")
        return units_by_id

    def _validate_root(self) -> None:
        roots = [unit for unit in self.units if unit.parent_id is None]
        if len(roots) != 1:
            raise ValueError("document hierarchy must contain exactly one root")
        root = roots[0]
        if (
            root.kind is not UnitKind.DOCUMENT
            or root.start_offset != 0
            or root.end_offset != len(self.text)
        ):
            raise ValueError("document root must cover the complete normalized text")

    @staticmethod
    def _validate_sibling_order(
        children_by_parent: dict[UUID, list[DocumentUnit]],
    ) -> None:
        for children in children_by_parent.values():
            ordered = sorted(children, key=lambda unit: unit.ordinal)
            if [unit.ordinal for unit in ordered] != list(range(len(ordered))):
                raise ValueError("sibling ordinals must be contiguous from zero")
            for previous, current in pairwise(ordered):
                if current.start_offset < previous.end_offset:
                    raise ValueError("peer document unit spans must not overlap")

    def _validate_unit(self, unit: DocumentUnit, units_by_id: dict[UUID, DocumentUnit]) -> None:
        if unit.id.int == 0 or unit.ordinal < 0:
            raise ValueError("document unit identity and ordinal are invalid")
        if not 0 <= unit.start_offset <= unit.end_offset <= len(self.text):
            raise ValueError("document unit offsets are outside document bounds")
        if unit.content_sha256 != _hash_text(self.text[unit.start_offset : unit.end_offset]):
            raise ValueError("document unit content hash does not match its text span")
        if unit.parent_id is not None:
            parent = units_by_id.get(unit.parent_id)
            if parent is None:
                raise ValueError("document unit parent does not exist")
            if unit.start_offset < parent.start_offset or unit.end_offset > parent.end_offset:
                raise ValueError("document unit span must be contained by its parent")
        if (unit.start_page is None) != (unit.end_page is None):
            raise ValueError("document unit page range must be complete")
        if (
            unit.start_page is not None
            and unit.end_page is not None
            and (unit.start_page <= 0 or unit.end_page < unit.start_page)
        ):
            raise ValueError("document unit page range is invalid")

    @staticmethod
    def _reject_cycles(units_by_id: dict[UUID, DocumentUnit]) -> None:
        for unit in units_by_id.values():
            visited: set[UUID] = set()
            current = unit
            while current.parent_id is not None:
                if current.id in visited:
                    raise ValueError("document unit hierarchy must not contain cycles")
                visited.add(current.id)
                current = units_by_id[current.parent_id]


@dataclass(frozen=True, slots=True)
class PublicationCommand:
    """All immutable artifacts required for one atomic publication."""

    work_id: UUID
    lease_token: UUID
    pipeline_version: str
    origin_sha256: Sha256
    artifact: DocumentArtifact

    def validate(self) -> None:
        """Validate publication identity, provenance, and the complete artifact."""
        if self.work_id.int == 0 or self.lease_token.int == 0:
            raise ValueError("publication work and lease identifiers are required")
        if not self.pipeline_version.strip():
            raise ValueError("publication pipeline version is required")
        self.artifact.validate()


def _hash_text(text: str) -> Sha256:
    return Sha256(hashlib.sha256(text.encode("utf-8")).hexdigest())

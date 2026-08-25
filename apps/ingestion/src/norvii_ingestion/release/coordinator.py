"""Cohesive orchestration of automatic graph-ready snapshot publication."""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING, Protocol

from norvii_ingestion.graph_projection import GraphProjectionBuildError

if TYPE_CHECKING:
    from uuid import UUID

    from norvii_ingestion.domain.models import IngestionWork


class GraphReleaseCoordinatorError(RuntimeError):
    """Expose a safe, actionable category when graph-ready publication cannot finish."""

    def __init__(self, detail: str) -> None:
        super().__init__("Graph-ready snapshot publication failed.")
        self.detail = detail


@dataclass(frozen=True, slots=True)
class StagedSnapshot:
    """Return the immutable candidate and the active release version it observed."""

    snapshot_id: UUID
    expected_release_version: int


class SnapshotReleasePort(Protocol):
    """Stage and activate corpus snapshots through the API ownership boundary."""

    def stage(self, corpus_id: UUID, source_id: UUID, document_id: UUID) -> StagedSnapshot:
        """Create or reuse an immutable candidate snapshot without activating it."""
        ...

    def activate(self, corpus_id: UUID, snapshot_id: UUID, expected_release_version: int) -> None:
        """Activate one graph-ready candidate snapshot."""
        ...


class GraphBuilderPort(Protocol):
    """Build the Neo4j projection for one immutable snapshot."""

    def build(self, corpus_id: UUID, snapshot_id: UUID) -> object:
        """Persist a ready graph release or raise a safe build error."""
        ...


class GraphReleaseCoordinator:
    """Guarantee that a successful ingestion exposes vector and graph retrieval together."""

    def __init__(self, snapshots: SnapshotReleasePort, graph_builder: GraphBuilderPort) -> None:
        self._snapshots = snapshots
        self._graph_builder = graph_builder

    def publish(self, work: IngestionWork, document_id: UUID) -> None:
        """Stage, materialize, and activate a candidate without advancing an incomplete release."""
        try:
            candidate = self._snapshots.stage(
                work.claim.corpus_id, work.claim.source_id, document_id
            )
            self._graph_builder.build(work.claim.corpus_id, candidate.snapshot_id)
            self._snapshots.activate(
                work.claim.corpus_id,
                candidate.snapshot_id,
                candidate.expected_release_version,
            )
        except GraphProjectionBuildError as error:
            raise GraphReleaseCoordinatorError("graph_projection_failed") from error
        except Exception as error:
            if isinstance(error, GraphReleaseCoordinatorError):
                raise
            raise GraphReleaseCoordinatorError("graph_release_publication_failed") from error

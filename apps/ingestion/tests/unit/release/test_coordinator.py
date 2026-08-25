from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest

from norvii_ingestion.domain.models import IngestionWork, SourceKind, WorkClaim, WorkReason
from norvii_ingestion.graph_projection import GraphProjectionBuildError
from norvii_ingestion.release.coordinator import (
    GraphReleaseCoordinator,
    GraphReleaseCoordinatorError,
    StagedSnapshot,
)


class RecordingSnapshotReleasePort:
    def __init__(self) -> None:
        self.staged: list[tuple[object, object, object]] = []
        self.activated: list[tuple[object, object, int]] = []
        self.candidate = StagedSnapshot(uuid4(), 2)

    def stage(self, corpus_id: object, source_id: object, document_id: object) -> StagedSnapshot:
        self.staged.append((corpus_id, source_id, document_id))
        return self.candidate

    def activate(
        self, corpus_id: object, snapshot_id: object, expected_release_version: int
    ) -> None:
        self.activated.append((corpus_id, snapshot_id, expected_release_version))


class RecordingGraphBuilder:
    def __init__(self, failure: Exception | None = None) -> None:
        self._failure = failure
        self.calls: list[tuple[object, object]] = []

    def build(self, corpus_id: object, snapshot_id: object) -> None:
        self.calls.append((corpus_id, snapshot_id))
        if self._failure is not None:
            raise self._failure


def test_coordinator_activates_only_after_building_the_staged_snapshot() -> None:
    work = _work()
    snapshots = RecordingSnapshotReleasePort()
    graph_builder = RecordingGraphBuilder()
    coordinator = GraphReleaseCoordinator(snapshots, graph_builder)

    coordinator.publish(work, work.attempt_id)

    assert graph_builder.calls == [(work.claim.corpus_id, snapshots.candidate.snapshot_id)]
    assert snapshots.activated == [
        (
            work.claim.corpus_id,
            snapshots.candidate.snapshot_id,
            snapshots.candidate.expected_release_version,
        )
    ]


def test_coordinator_does_not_activate_when_graph_build_fails() -> None:
    work = _work()
    snapshots = RecordingSnapshotReleasePort()
    coordinator = GraphReleaseCoordinator(
        snapshots, RecordingGraphBuilder(GraphProjectionBuildError("no relationships"))
    )

    with pytest.raises(
        GraphReleaseCoordinatorError, match="Graph-ready snapshot publication failed"
    ) as error:
        coordinator.publish(work, work.attempt_id)

    assert error.value.detail == "graph_projection_failed"
    assert snapshots.activated == []


def _work() -> IngestionWork:
    now = datetime(2026, 8, 25, 12, 0, tzinfo=UTC)
    return IngestionWork(
        claim=WorkClaim(
            work_id=uuid4(),
            corpus_id=uuid4(),
            source_id=uuid4(),
            source_kind=SourceKind.URL,
            reason=WorkReason.REPROCESS,
            lease_token=uuid4(),
            lease_expires_at=now + timedelta(minutes=2),
        ),
        attempt_id=uuid4(),
        corpus_language="en",
        url="https://example.org/law",
        pdf_content=None,
    )

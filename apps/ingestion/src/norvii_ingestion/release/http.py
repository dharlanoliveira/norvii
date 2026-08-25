"""Bounded HTTP adapter for snapshot staging and graph-ready activation."""

from __future__ import annotations

import json
from typing import TYPE_CHECKING, cast
from urllib import error, request
from uuid import UUID

from norvii_ingestion.release.coordinator import GraphReleaseCoordinatorError, StagedSnapshot

if TYPE_CHECKING:
    from collections.abc import Callable
    from urllib.response import addinfourl


class SnapshotReleaseHttpClient:
    """Call the API's snapshot release boundary without leaking HTTP details upstream."""

    def __init__(
        self,
        base_url: str,
        timeout_seconds: int,
        opener: Callable[..., addinfourl] = request.urlopen,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout_seconds = timeout_seconds
        self._opener = opener

    def stage(self, corpus_id: UUID, source_id: UUID, document_id: UUID) -> StagedSnapshot:
        """Create or reuse a candidate snapshot and return the observed active-release version."""
        payload = self._post(
            f"/api/v1/corpora/{corpus_id}/snapshots/stage",
            {"sourceId": str(source_id), "documentId": str(document_id)},
        )
        try:
            return self._staged_snapshot(payload)
        except (KeyError, TypeError, ValueError) as error:
            raise GraphReleaseCoordinatorError("snapshot_stage_response_invalid") from error

    def activate(self, corpus_id: UUID, snapshot_id: UUID, expected_release_version: int) -> None:
        """Advance the active release after the graph builder has completed successfully."""
        self._post(
            f"/api/v1/corpora/{corpus_id}/snapshots/{snapshot_id}/activate",
            {"expectedReleaseVersion": expected_release_version},
        )

    def _post(self, path: str, payload: dict[str, object]) -> dict[str, object]:
        body = json.dumps(payload).encode("utf-8")
        outgoing = request.Request(  # noqa: S310 - constructor receives validated HTTP base URL
            self._base_url + path,
            data=body,
            method="POST",
            headers={"Content-Type": "application/json"},
        )
        try:
            with self._opener(outgoing, timeout=self._timeout_seconds) as response:
                decoded = json.load(response)
        except (
            error.HTTPError,
            error.URLError,
            OSError,
            TimeoutError,
            json.JSONDecodeError,
        ) as exception:
            raise GraphReleaseCoordinatorError("snapshot_release_api_unavailable") from exception
        if not isinstance(decoded, dict):
            raise GraphReleaseCoordinatorError("snapshot_release_response_invalid")
        return cast("dict[str, object]", decoded)

    @staticmethod
    def _staged_snapshot(payload: dict[str, object]) -> StagedSnapshot:
        snapshot = payload["snapshot"]
        release = payload["release"]
        if not isinstance(snapshot, dict) or not isinstance(release, dict):
            raise TypeError("snapshot release payload is invalid")
        snapshot_id = UUID(snapshot["id"])
        expected_release_version = int(release["version"])
        if expected_release_version < 0:
            raise ValueError("snapshot release version is invalid")
        return StagedSnapshot(snapshot_id, expected_release_version)

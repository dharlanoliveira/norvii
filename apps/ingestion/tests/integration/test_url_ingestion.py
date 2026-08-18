from __future__ import annotations

from datetime import timedelta

import pytest

from norvii_ingestion.acquisition.https import HttpsAcquirer
from norvii_ingestion.config import WorkerConfig
from norvii_ingestion.domain.artifacts import UnitKind
from norvii_ingestion.extraction.html import HtmlExtractor

pytestmark = pytest.mark.integration


class ControlledResponse:
    status = 200

    def __init__(self, content: bytes) -> None:
        self._content = content

    def getheader(self, name: str) -> str | None:
        return {
            "Content-Type": "text/html; charset=utf-8",
            "Content-Length": str(len(self._content)),
        }.get(name)

    def read(self, amount: int = -1) -> bytes:
        if amount < 0:
            return self._content
        result, self._content = self._content[:amount], self._content[amount:]
        return result


class ControlledConnection:
    def __init__(self, content: bytes) -> None:
        self._response = ControlledResponse(content)
        self.closed = False

    def request(self, method: str, target: str, *, headers: dict[str, str]) -> None:
        assert method == "GET"
        assert target == "/official-law"
        assert headers["Host"] == "example.org"

    def getresponse(self) -> ControlledResponse:
        return self._response

    def close(self) -> None:
        self.closed = True


def test_controlled_https_capture_produces_complete_legal_artifact() -> None:
    html = (
        "<main><h1>Title I Official law</h1>"
        "<p>This title establishes a controlled legal framework for integration verification.</p>"
        "<p>Article 1 The complete official content remains addressable and preserved. "
        + ("Legal evidence remains stable. " * 20)
        + "</p></main>"
    ).encode()
    connection = ControlledConnection(html)
    acquisition = HttpsAcquirer(
        _config(),
        resolver=lambda _host, _port: ("93.184.216.34",),
        connection_factory=lambda *_args: connection,
    ).acquire("https://example.org/official-law")

    artifact = HtmlExtractor().extract(acquisition.content)

    artifact.validate()
    assert connection.closed
    assert acquisition.final_url == "https://example.org/official-law"
    assert artifact.units[0].end_offset == len(artifact.text)
    assert UnitKind.ARTICLE in {unit.kind for unit in artifact.units}


def _config() -> WorkerConfig:
    return WorkerConfig(
        poll_interval=timedelta(seconds=1),
        lease_duration=timedelta(minutes=2),
        max_source_bytes=1024 * 1024,
        max_redirects=5,
        connect_timeout=timedelta(seconds=2),
        read_timeout=timedelta(seconds=2),
        pipeline_version="corpus-ingestion-v1",
    )

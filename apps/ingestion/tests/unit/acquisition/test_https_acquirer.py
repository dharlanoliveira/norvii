from __future__ import annotations

from datetime import timedelta

import pytest

from norvii_ingestion.acquisition.https import (
    AcquisitionError,
    AcquisitionFailureReason,
    AcquisitionLimitError,
    HttpsAcquirer,
    UnsafeUrlError,
    UnsupportedContentError,
)
from norvii_ingestion.config import WorkerConfig


class FakeResponse:
    def __init__(
        self,
        *,
        status: int = 200,
        headers: dict[str, str] | None = None,
        chunks: list[bytes] | None = None,
        failure: Exception | None = None,
    ) -> None:
        self.status = status
        self._headers = headers or {"Content-Type": "text/html"}
        self._chunks = list(chunks or [b"legal text", b""])
        self._failure = failure

    def getheader(self, name: str) -> str | None:
        return self._headers.get(name)

    def read(self, _amount: int = -1) -> bytes:
        if self._failure is not None:
            raise self._failure
        return self._chunks.pop(0)


class FakeConnection:
    def __init__(self, response: FakeResponse, response_failure: Exception | None = None) -> None:
        self.response = response
        self.response_failure = response_failure
        self.requests: list[tuple[str, str, dict[str, str]]] = []
        self.closed = False

    def request(self, method: str, target: str, *, headers: dict[str, str]) -> None:
        self.requests.append((method, target, headers))

    def getresponse(self) -> FakeResponse:
        if self.response_failure is not None:
            raise self.response_failure
        return self.response

    def close(self) -> None:
        self.closed = True


def test_acquirer_pins_validated_address_and_ignores_proxy_environment() -> None:
    connection = FakeConnection(FakeResponse())
    calls: list[tuple[str, int, str]] = []

    def factory(
        host: str, port: int, address: str, _connect: float, _read: float
    ) -> FakeConnection:
        calls.append((host, port, address))
        return connection

    acquisition = HttpsAcquirer(
        _config(), resolver=lambda _host, _port: ("93.184.216.34",), connection_factory=factory
    ).acquire("https://example.org/legal?q=1")

    assert acquisition.content == b"legal text"
    assert calls == [("example.org", 443, "93.184.216.34")]
    assert connection.requests[0][1] == "/legal?q=1"
    assert connection.requests[0][2]["Host"] == "example.org"
    assert connection.requests[0][2]["User-Agent"] == (
        "Mozilla/5.0 (compatible; Norvii/1.0; +https://github.com/dharlanoliveira/norvii)"
    )
    assert connection.requests[0][2]["Accept"] == (
        "application/pdf, application/xhtml+xml;q=0.9, text/html;q=0.8, text/plain;q=0.7"
    )
    assert connection.requests[0][2]["Accept-Language"] == "eng, en;q=0.9"
    assert connection.requests[0][2]["Accept-Max-Cs-Size"] == "1024"


@pytest.mark.parametrize(
    "address",
    [
        "127.0.0.1",
        "10.0.0.1",
        "172.16.0.1",
        "192.168.0.1",
        "169.254.1.1",
        "0.0.0.0",  # noqa: S104 - deliberate unsafe-address policy case
        "224.0.0.1",
        "::1",
        "fc00::1",
        "fe80::1",
        "2001:db8::1",
    ],
)
def test_acquirer_rejects_every_non_public_address_class(address: str) -> None:
    acquirer = HttpsAcquirer(_config(), resolver=lambda _host, _port: (address,))

    with pytest.raises(UnsafeUrlError):
        acquirer.acquire("https://example.org/legal")


def test_acquirer_revalidates_redirect_destination() -> None:
    first = FakeConnection(
        FakeResponse(status=302, headers={"Location": "https://internal.example/legal"})
    )

    def resolver(host: str, _port: int) -> tuple[str, ...]:
        return ("93.184.216.34",) if host == "example.org" else ("10.0.0.1",)

    acquirer = HttpsAcquirer(_config(), resolver=resolver, connection_factory=lambda *_args: first)

    with pytest.raises(UnsafeUrlError):
        acquirer.acquire("https://example.org/legal")
    assert first.closed


def test_acquirer_rejects_declared_and_streamed_oversize_content() -> None:
    declared = FakeConnection(
        FakeResponse(headers={"Content-Type": "text/html", "Content-Length": "9"})
    )
    streamed = FakeConnection(FakeResponse(chunks=[b"12345", b"6789", b""]))
    connections = iter((declared, streamed))
    acquirer = HttpsAcquirer(
        _config(max_source_bytes=8),
        resolver=lambda _host, _port: ("93.184.216.34",),
        connection_factory=lambda *_args: next(connections),
    )

    with pytest.raises(AcquisitionLimitError):
        acquirer.acquire("https://example.org/declared")
    with pytest.raises(AcquisitionLimitError):
        acquirer.acquire("https://example.org/streamed")


def test_acquirer_categorizes_timeout_and_unsupported_media() -> None:
    timed_out = FakeConnection(FakeResponse(failure=TimeoutError()))
    unsupported = FakeConnection(FakeResponse(headers={"Content-Type": "application/octet-stream"}))
    connections = iter((timed_out, unsupported))
    acquirer = HttpsAcquirer(
        _config(),
        resolver=lambda _host, _port: ("93.184.216.34",),
        connection_factory=lambda *_args: next(connections),
    )

    with pytest.raises(AcquisitionError):
        acquirer.acquire("https://example.org/timeout")
    with pytest.raises(UnsupportedContentError):
        acquirer.acquire("https://example.org/binary")


@pytest.mark.parametrize(
    ("header", "expected_media_type"),
    [
        ("application/pdf", "application/pdf"),
        ("Application/PDF; charset=binary", "application/pdf"),
    ],
)
def test_acquirer_accepts_pdf_content_type(header: str, expected_media_type: str) -> None:
    acquirer = HttpsAcquirer(
        _config(),
        resolver=lambda _host, _port: ("93.184.216.34",),
        connection_factory=lambda *_args: FakeConnection(
            FakeResponse(headers={"Content-Type": header})
        ),
    )

    acquisition = acquirer.acquire("https://example.org/official-law.pdf")

    assert acquisition.media_type == expected_media_type


def test_acquirer_records_an_allowlisted_connection_reset_reason() -> None:
    connection = FakeConnection(
        FakeResponse(), ConnectionResetError("origin-specific private detail")
    )
    acquirer = HttpsAcquirer(
        _config(),
        resolver=lambda _host, _port: ("93.184.216.34",),
        connection_factory=lambda *_args: connection,
    )

    with pytest.raises(AcquisitionError) as captured:
        acquirer.acquire("https://example.org/legal")

    assert captured.value.reason is AcquisitionFailureReason.CONNECTION_RESET
    assert "origin-specific" not in str(captured.value)


def _config(max_source_bytes: int = 1024) -> WorkerConfig:
    return WorkerConfig(
        poll_interval=timedelta(seconds=1),
        lease_duration=timedelta(minutes=2),
        max_source_bytes=max_source_bytes,
        max_redirects=2,
        connect_timeout=timedelta(seconds=1),
        read_timeout=timedelta(seconds=1),
        pipeline_version="corpus-ingestion-v1",
    )

"""Proxy-free HTTPS acquisition pinned to validated public addresses."""

from __future__ import annotations

import http.client
import ipaddress
import socket
import ssl
from collections.abc import Callable
from dataclasses import dataclass
from enum import StrEnum
from typing import TYPE_CHECKING, Protocol, cast
from urllib.parse import SplitResult, urljoin, urlsplit, urlunsplit

if TYPE_CHECKING:
    from norvii_ingestion.config import WorkerConfig

_REDIRECT_STATUSES = frozenset({301, 302, 303, 307, 308})
_SUPPORTED_MEDIA_TYPES = frozenset({"text/html", "application/xhtml+xml", "text/plain"})
_READ_CHUNK_SIZE = 64 * 1024
_HTTPS_PORT = 443
_SUCCESS_STATUS_MINIMUM = 200
_SUCCESS_STATUS_MAXIMUM = 300
_USER_AGENT = "Mozilla/5.0 (compatible; Norvii/1.0; +https://github.com/dharlanoliveira/norvii)"
_ACCEPTED_REPRESENTATIONS = "application/xhtml+xml, text/html;q=0.9, text/plain;q=0.8"
_PREFERRED_LANGUAGES = "eng, en;q=0.9"


class UnsafeUrlError(ValueError):
    """Report a URL that violates the public HTTPS destination policy."""


class UnsupportedContentError(ValueError):
    """Report a response media type unsupported by extraction."""


class AcquisitionLimitError(ValueError):
    """Report a redirect or byte limit reached before extraction."""


class AcquisitionFailureReason(StrEnum):
    """Allowlisted transport reasons safe to retain for operations."""

    CONNECTION_RESET = "connection_reset"
    TIMEOUT = "timeout"
    TLS_ERROR = "tls_error"
    HTTP_STATUS = "http_status"
    INVALID_RESPONSE = "invalid_response"
    TRANSPORT_ERROR = "transport_error"


class AcquisitionError(RuntimeError):
    """Report a bounded HTTPS transport failure without private response data."""

    def __init__(
        self,
        message: str,
        reason: AcquisitionFailureReason = AcquisitionFailureReason.TRANSPORT_ERROR,
    ) -> None:
        super().__init__(message)
        self.reason = reason


@dataclass(frozen=True, slots=True)
class Acquisition:
    """A bounded supported response ready for deterministic extraction."""

    content: bytes
    final_url: str
    media_type: str


class _Response(Protocol):
    status: int

    def getheader(self, name: str) -> str | None:
        """Return one response header."""
        ...

    def read(self, amount: int = -1) -> bytes:
        """Read at most amount response bytes."""
        ...


class _Connection(Protocol):
    def request(self, method: str, target: str, *, headers: dict[str, str]) -> None:
        """Send one origin-form HTTPS request."""
        ...

    def getresponse(self) -> _Response:
        """Return the origin response."""
        ...

    def close(self) -> None:
        """Release the underlying socket."""
        ...


Resolver = Callable[[str, int], tuple[str, ...]]
ConnectionFactory = Callable[[str, int, str, float, float], _Connection]


class HttpsAcquirer:
    """Acquire HTTPS content after DNS validation and address pinning."""

    def __init__(
        self,
        config: WorkerConfig,
        *,
        resolver: Resolver | None = None,
        connection_factory: ConnectionFactory | None = None,
    ) -> None:
        self._config = config
        self._resolver = resolver or _resolve_addresses
        self._connection_factory = connection_factory or _create_connection

    def acquire(self, url: str) -> Acquisition:
        """Acquire one supported response while revalidating every redirect."""
        current_url = url
        for redirect_count in range(self._config.max_redirects + 1):
            parsed = self._validate_url(current_url)
            response, connection = self._request(parsed)
            try:
                if response.status in _REDIRECT_STATUSES:
                    location = response.getheader("Location")
                    if location is None:
                        raise AcquisitionError(
                            "HTTPS redirect is missing a location.",
                            AcquisitionFailureReason.INVALID_RESPONSE,
                        )
                    if redirect_count >= self._config.max_redirects:
                        raise AcquisitionLimitError("HTTPS redirect limit was exceeded.")
                    current_url = urljoin(current_url, location)
                    continue
                if (
                    response.status < _SUCCESS_STATUS_MINIMUM
                    or response.status >= _SUCCESS_STATUS_MAXIMUM
                ):
                    raise AcquisitionError(
                        "HTTPS origin returned an unsuccessful status.",
                        AcquisitionFailureReason.HTTP_STATUS,
                    )
                media_type = self._media_type(response)
                try:
                    content = self._read_bounded(response)
                except (OSError, http.client.HTTPException) as error:
                    raise AcquisitionError(
                        "HTTPS response streaming failed.",
                        _transport_failure_reason(error),
                    ) from error
                return Acquisition(content, _normalized_url(parsed), media_type)
            finally:
                connection.close()
        raise AcquisitionLimitError("HTTPS redirect limit was exceeded.")

    def _request(self, parsed: SplitResult) -> tuple[_Response, _Connection]:
        host = parsed.hostname
        if host is None:
            raise UnsafeUrlError("HTTPS URL must include a host.")
        port = parsed.port or _HTTPS_PORT
        addresses = self._resolver(host, port)
        if not addresses:
            raise UnsafeUrlError("HTTPS host did not resolve to a public address.")
        for address in addresses:
            self._validate_public_address(address)
        pinned_address = min(addresses)
        try:
            connection = self._connection_factory(
                host,
                port,
                pinned_address,
                self._config.connect_timeout.total_seconds(),
                self._config.read_timeout.total_seconds(),
            )
        except OSError as error:
            raise AcquisitionError(
                "HTTPS connection failed.", _transport_failure_reason(error)
            ) from error
        target = parsed.path or "/"
        if parsed.query:
            target = f"{target}?{parsed.query}"
        host_header = host if port == _HTTPS_PORT else f"{host}:{port}"
        try:
            connection.request(
                "GET",
                target,
                headers={
                    "Host": host_header,
                    "Accept": _ACCEPTED_REPRESENTATIONS,
                    "Accept-Language": _PREFERRED_LANGUAGES,
                    "Accept-Max-Cs-Size": str(self._config.max_source_bytes),
                    "User-Agent": _USER_AGENT,
                    "Connection": "close",
                },
            )
            return connection.getresponse(), connection
        except (OSError, http.client.HTTPException) as error:
            connection.close()
            raise AcquisitionError(
                "HTTPS acquisition failed.", _transport_failure_reason(error)
            ) from error

    @staticmethod
    def _validate_url(url: str) -> SplitResult:
        try:
            parsed = urlsplit(url)
            _ = parsed.port
        except ValueError as error:
            raise UnsafeUrlError("HTTPS URL is invalid.") from error
        if parsed.scheme.lower() != "https" or parsed.hostname is None:
            raise UnsafeUrlError("Only absolute HTTPS URLs are supported.")
        if parsed.username is not None or parsed.password is not None:
            raise UnsafeUrlError("HTTPS URL must not contain credentials.")
        if parsed.fragment:
            parsed = parsed._replace(fragment="")
        return parsed

    @staticmethod
    def _validate_public_address(address: str) -> None:
        try:
            parsed = ipaddress.ip_address(address)
        except ValueError as error:
            raise UnsafeUrlError("HTTPS host resolved to an invalid address.") from error
        if not parsed.is_global or parsed.is_multicast:
            raise UnsafeUrlError("HTTPS host must resolve exclusively to public addresses.")

    def _read_bounded(self, response: _Response) -> bytes:
        declared_length = response.getheader("Content-Length")
        if declared_length is not None:
            try:
                parsed_length = int(declared_length)
            except ValueError as error:
                raise AcquisitionError(
                    "HTTPS response content length is invalid.",
                    AcquisitionFailureReason.INVALID_RESPONSE,
                ) from error
            if parsed_length < 0 or parsed_length > self._config.max_source_bytes:
                raise AcquisitionLimitError("HTTPS response exceeds the supported size.")
        chunks: list[bytes] = []
        total = 0
        while chunk := response.read(_READ_CHUNK_SIZE):
            total += len(chunk)
            if total > self._config.max_source_bytes:
                raise AcquisitionLimitError("HTTPS response exceeds the supported size.")
            chunks.append(chunk)
        return b"".join(chunks)

    @staticmethod
    def _media_type(response: _Response) -> str:
        header = response.getheader("Content-Type")
        if header is None:
            raise UnsupportedContentError("HTTPS response content type is required.")
        media_type = header.split(";", maxsplit=1)[0].strip().lower()
        if media_type not in _SUPPORTED_MEDIA_TYPES:
            raise UnsupportedContentError("HTTPS response content type is unsupported.")
        return media_type


class _PinnedHTTPSConnection(http.client.HTTPSConnection):
    def __init__(
        self,
        host: str,
        port: int,
        pinned_address: str,
        connect_timeout: float,
        read_timeout: float,
    ) -> None:
        tls_context = ssl.create_default_context()
        super().__init__(host, port, timeout=connect_timeout, context=tls_context)
        self._pinned_address = pinned_address
        self._read_timeout = read_timeout
        self._tls_context = tls_context

    def connect(self) -> None:
        """Connect only to the validated address while preserving TLS hostname checks."""
        raw_socket = socket.create_connection(
            (self._pinned_address, self.port),
            self.timeout,
            None,
        )
        try:
            wrapped = self._tls_context.wrap_socket(raw_socket, server_hostname=self.host)
        except Exception:
            raw_socket.close()
            raise
        wrapped.settimeout(self._read_timeout)
        self.sock = wrapped


def _resolve_addresses(host: str, port: int) -> tuple[str, ...]:
    try:
        records = socket.getaddrinfo(host, port, type=socket.SOCK_STREAM)
    except socket.gaierror as error:
        raise UnsafeUrlError("HTTPS host could not be resolved.") from error
    return tuple(sorted({cast("str", record[4][0]) for record in records}))


def _create_connection(
    host: str,
    port: int,
    pinned_address: str,
    connect_timeout: float,
    read_timeout: float,
) -> _Connection:
    return _PinnedHTTPSConnection(host, port, pinned_address, connect_timeout, read_timeout)


def _transport_failure_reason(error: BaseException) -> AcquisitionFailureReason:
    if isinstance(error, ConnectionResetError):
        return AcquisitionFailureReason.CONNECTION_RESET
    if isinstance(error, TimeoutError):
        return AcquisitionFailureReason.TIMEOUT
    if isinstance(error, ssl.SSLError):
        return AcquisitionFailureReason.TLS_ERROR
    return AcquisitionFailureReason.TRANSPORT_ERROR


def _normalized_url(parsed: SplitResult) -> str:
    return urlunsplit(("https", parsed.netloc, parsed.path or "/", parsed.query, ""))

"""One-shot persistence verification with deterministic resource ownership."""

from __future__ import annotations

from dataclasses import dataclass
from typing import ClassVar, Protocol


class PersistenceVerificationError(RuntimeError):
    """Indicate a service-scoped connectivity or cleanup failure."""


class PersistenceStore(Protocol):
    """Define the narrow behavior consumed by the ingestion verifier."""

    name: ClassVar[str]

    def verify(self) -> None:
        """Authenticate and execute a constant read-only operation."""

    def close(self) -> None:
        """Release the underlying driver resources."""


@dataclass(frozen=True, slots=True)
class VerificationResult:
    """Identify one successfully checked persistence store."""

    store: str


class PersistenceVerifier:
    """Check ordered stores and close every acquired resource in reverse order."""

    def __init__(self, stores: list[PersistenceStore]) -> None:
        self._stores = tuple(stores)

    def verify(self) -> tuple[VerificationResult, ...]:
        """Verify every store once and always release each store."""
        results: list[VerificationResult] = []
        verification_error: PersistenceVerificationError | None = None
        try:
            for store in self._stores:
                try:
                    store.verify()
                except Exception as error:
                    raise PersistenceVerificationError(
                        f"Verify {store.name} connectivity failed."
                    ) from error
                results.append(VerificationResult(store=store.name))
        except PersistenceVerificationError as error:
            verification_error = error
        finally:
            cleanup_error = self._close_stores()

        if verification_error is not None:
            raise verification_error
        if cleanup_error is not None:
            raise cleanup_error
        return tuple(results)

    def _close_stores(self) -> PersistenceVerificationError | None:
        first_error: PersistenceVerificationError | None = None
        for store in reversed(self._stores):
            try:
                store.close()
            except Exception:  # noqa: BLE001 - cleanup must cover every protocol implementation
                if first_error is None:
                    first_error = PersistenceVerificationError(
                        f"Close {store.name} persistence connection failed."
                    )
        return first_error

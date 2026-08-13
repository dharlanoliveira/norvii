from __future__ import annotations

import pytest

from norvii_ingestion.publication.persistence.verifier import (
    PersistenceVerificationError,
    PersistenceVerifier,
)


def test_verifier_checks_and_closes_every_store_in_order() -> None:
    events: list[str] = []
    verifier = PersistenceVerifier([FakeStore("PostgreSQL", events), FakeStore("Neo4j", events)])

    results = verifier.verify()

    assert [result.store for result in results] == ["PostgreSQL", "Neo4j"]
    assert events == [
        "verify PostgreSQL",
        "verify Neo4j",
        "close Neo4j",
        "close PostgreSQL",
    ]


class FakeStore:
    def __init__(
        self,
        name: str,
        events: list[str],
        verification_error: Exception | None = None,
        close_error: Exception | None = None,
    ) -> None:
        self.name = name
        self._events = events
        self._verification_error = verification_error
        self._close_error = close_error

    def verify(self) -> None:
        self._events.append(f"verify {self.name}")
        if self._verification_error is not None:
            raise self._verification_error

    def close(self) -> None:
        self._events.append(f"close {self.name}")
        if self._close_error is not None:
            raise self._close_error


def test_verifier_attributes_failure_and_still_closes_every_store() -> None:
    events: list[str] = []
    verifier = PersistenceVerifier(
        [
            FakeStore("PostgreSQL", events, RuntimeError("authentication rejected")),
            FakeStore("Neo4j", events),
        ]
    )

    with pytest.raises(PersistenceVerificationError, match="Verify PostgreSQL connectivity failed"):
        verifier.verify()

    assert events == ["verify PostgreSQL", "close Neo4j", "close PostgreSQL"]


def test_verifier_continues_cleanup_after_one_store_close_fails() -> None:
    events: list[str] = []
    verifier = PersistenceVerifier(
        [
            FakeStore("PostgreSQL", events),
            FakeStore("Neo4j", events, close_error=RuntimeError("close failed")),
        ]
    )

    with pytest.raises(PersistenceVerificationError, match="Close Neo4j"):
        verifier.verify()

    assert events == [
        "verify PostgreSQL",
        "verify Neo4j",
        "close Neo4j",
        "close PostgreSQL",
    ]

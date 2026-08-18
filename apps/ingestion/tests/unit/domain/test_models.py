from __future__ import annotations

from dataclasses import FrozenInstanceError
from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest

from norvii_ingestion.domain.models import (
    FailureCategory,
    SafeFailure,
    Sha256,
    SourceKind,
    WorkClaim,
    WorkReason,
)


def test_claim_is_immutable_and_retains_lease_identity() -> None:
    claim = WorkClaim(
        work_id=uuid4(),
        corpus_id=uuid4(),
        source_id=uuid4(),
        source_kind=SourceKind.URL,
        reason=WorkReason.INITIAL,
        lease_token=uuid4(),
        lease_expires_at=datetime.now(UTC) + timedelta(minutes=2),
    )

    with pytest.raises(FrozenInstanceError):
        claim.reason = WorkReason.RETRY  # type: ignore[misc]


@pytest.mark.parametrize("value", ["short", "G" * 64, "a" * 63])
def test_sha256_rejects_noncanonical_values(value: str) -> None:
    with pytest.raises(ValueError, match="SHA-256"):
        Sha256(value)


def test_safe_failure_rejects_unbounded_detail() -> None:
    with pytest.raises(ValueError, match="500"):
        SafeFailure(FailureCategory.EXTRACTION_FAILED, "x" * 501)

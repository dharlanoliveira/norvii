from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from norvii_ingestion.contracts import ClaimPayload, FailurePayload, PublicationPayload

FIXTURE_DIRECTORY = (
    Path(__file__).resolve().parents[4] / "contracts" / "corpus-ingestion" / "v1" / "fixtures"
)


def test_claim_fixture_decodes_with_opaque_lease() -> None:
    claim = ClaimPayload.from_mapping(_fixture("claim.json"))

    assert claim.source_kind == "url"
    assert claim.reason == "initial"
    assert claim.lease_token.int != 0


def test_publication_and_failure_fixtures_are_discriminated() -> None:
    publication = PublicationPayload.from_mapping(_fixture("publication.json"))
    failure = FailurePayload.from_mapping(_fixture("failure.json"))

    assert publication.units[0].kind == "document"
    assert failure.category == "extraction_failed"


def _fixture(name: str) -> dict[str, Any]:
    value: object = json.loads((FIXTURE_DIRECTORY / name).read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value

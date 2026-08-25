from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import NAMESPACE_URL, UUID, uuid4, uuid5

from norvii_ingestion.acquisition.https import (
    Acquisition,
    AcquisitionError,
    AcquisitionFailureReason,
    UnsafeUrlError,
)
from norvii_ingestion.domain.artifacts import (
    DocumentArtifact,
    DocumentUnit,
    PublicationCommand,
    UnitKind,
)
from norvii_ingestion.domain.models import (
    FailureCategory,
    IngestionWork,
    OriginCapture,
    SafeFailure,
    Sha256,
    SourceKind,
    WorkClaim,
    WorkReason,
)
from norvii_ingestion.enrichment.embedding import EmbeddingProviderError
from norvii_ingestion.orchestration.processor import ArtifactExtractors, IngestionProcessor
from norvii_ingestion.publication.postgres.repository import WorkRepositoryError
from norvii_ingestion.semantic.extraction import ExtractionProviderError


class FakeAcquirer:
    def __init__(self, failure: Exception | None = None) -> None:
        self._failure = failure

    def acquire(self, url: str) -> Acquisition:
        if self._failure is not None:
            raise self._failure
        return Acquisition(b"Legal text", url, "text/html")


class FakeExtractor:
    def extract(self, content: bytes) -> DocumentArtifact:
        text = content.decode()
        digest = Sha256.from_bytes(content)
        root_id = uuid5(NAMESPACE_URL, str(digest))
        return DocumentArtifact(
            text=text,
            text_sha256=digest,
            units=(
                DocumentUnit(
                    id=root_id,
                    parent_id=None,
                    kind=UnitKind.DOCUMENT,
                    ordinal=0,
                    marker=None,
                    label=None,
                    start_offset=0,
                    end_offset=len(text),
                    start_page=None,
                    end_page=None,
                    locator="document",
                    content_sha256=digest,
                ),
            ),
        )


class FakeEmbeddingProvider:
    def embed(self, texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        return tuple((0.0,) * 1536 for _ in texts)


class FailingEmbeddingProvider:
    def embed(self, _texts: tuple[str, ...]) -> tuple[tuple[float, ...], ...]:
        raise EmbeddingProviderError("provider detail must not be exposed", "provider_timeout")


class FailingSemanticExtractor:
    def extract(self, _artifact: DocumentArtifact) -> None:
        raise ExtractionProviderError(
            "provider response must not be exposed", "provider_http_status_400"
        )


class RecordingRepository:
    def __init__(self) -> None:
        self.publications: list[tuple[IngestionWork, OriginCapture]] = []
        self.failures: list[SafeFailure] = []

    def publish(
        self,
        work: IngestionWork,
        capture: OriginCapture,
        command: PublicationCommand,
        now: datetime,
    ) -> UUID:
        command.validate()
        assert work.claim.work_id == command.work_id
        assert now.tzinfo is not None
        self.publications.append((work, capture))
        return uuid4()

    def fail(
        self,
        work: IngestionWork,
        failure: SafeFailure,
        now: datetime,
    ) -> None:
        assert work.claim.work_id.int != 0
        assert now.tzinfo is not None
        self.failures.append(failure)


class FailingPublicationRepository(RecordingRepository):
    def publish(self, *_args: object, **_kwargs: object) -> None:
        raise WorkRepositoryError(
            "database detail must not be exposed", "repository_sqlstate_23505"
        )


def test_processor_acquires_extracts_and_publishes_url_work() -> None:
    repository = RecordingRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v1",
        clock=_now,
        embedding_provider=FakeEmbeddingProvider(),
        embedding_model="test-embedding",
    )

    processor.process(_work())

    assert len(repository.publications) == 1
    assert repository.failures == []


def test_processor_categorizes_unsafe_url_without_exposing_its_detail() -> None:
    repository = RecordingRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(UnsafeUrlError("private address 10.0.0.1")),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v1",
        clock=_now,
        embedding_provider=FakeEmbeddingProvider(),
        embedding_model="test-embedding",
    )

    processor.process(_work())

    assert repository.publications == []
    assert repository.failures == [SafeFailure(FailureCategory.UNSAFE_URL)]


def test_processor_retains_only_allowlisted_acquisition_diagnostics() -> None:
    repository = RecordingRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(
            AcquisitionError(
                "origin-specific private detail",
                AcquisitionFailureReason.CONNECTION_RESET,
            )
        ),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v1",
        clock=_now,
        embedding_provider=FakeEmbeddingProvider(),
        embedding_model="test-embedding",
    )

    processor.process(_work())

    assert repository.failures == [
        SafeFailure(FailureCategory.ACQUISITION_FAILED, "connection_reset")
    ]


def test_processor_extracts_preserved_pdf_without_network_acquisition() -> None:
    repository = RecordingRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(AssertionError("PDF must not use network acquisition")),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v1",
        clock=_now,
        embedding_provider=FakeEmbeddingProvider(),
        embedding_model="test-embedding",
    )

    processor.process(_pdf_work())

    assert len(repository.publications) == 1
    assert repository.publications[0][1].media_type == "application/pdf"
    assert repository.publications[0][1].final_url is None


def test_processor_preserves_the_ready_version_when_embedding_fails() -> None:
    repository = RecordingRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v3",
        clock=_now,
        embedding_provider=FailingEmbeddingProvider(),
        embedding_model="test-embedding",
    )

    processor.process(_work())

    assert repository.publications == []
    assert repository.failures == [
        SafeFailure(FailureCategory.PUBLICATION_FAILED, "provider_timeout")
    ]


def test_processor_retains_allowlisted_semantic_provider_diagnostic() -> None:
    repository = RecordingRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v3",
        clock=_now,
        embedding_provider=FakeEmbeddingProvider(),
        embedding_model="test-embedding",
        semantic_extractor=FailingSemanticExtractor(),
    )

    processor.process(_work())

    assert repository.publications == []
    assert repository.failures == [
        SafeFailure(FailureCategory.EXTRACTION_FAILED, "provider_http_status_400")
    ]


def test_processor_retains_allowlisted_repository_diagnostic() -> None:
    repository = FailingPublicationRepository()
    processor = IngestionProcessor(
        repository=repository,
        acquirer=FakeAcquirer(),
        extractors=ArtifactExtractors(html=FakeExtractor(), pdf=FakeExtractor()),
        pipeline_version="corpus-ingestion-v3",
        clock=_now,
        embedding_provider=FakeEmbeddingProvider(),
        embedding_model="test-embedding",
    )

    processor.process(_work())

    assert repository.failures == [
        SafeFailure(FailureCategory.PUBLICATION_FAILED, "repository_sqlstate_23505")
    ]


def _work() -> IngestionWork:
    return IngestionWork(
        claim=WorkClaim(
            work_id=uuid4(),
            corpus_id=uuid4(),
            source_id=uuid4(),
            source_kind=SourceKind.URL,
            reason=WorkReason.INITIAL,
            lease_token=uuid4(),
            lease_expires_at=_now() + timedelta(minutes=2),
        ),
        attempt_id=uuid4(),
        corpus_language="en",
        url="https://example.com/legal",
        pdf_content=None,
    )


def _pdf_work() -> IngestionWork:
    return IngestionWork(
        claim=WorkClaim(
            work_id=uuid4(),
            corpus_id=uuid4(),
            source_id=uuid4(),
            source_kind=SourceKind.PDF,
            reason=WorkReason.INITIAL,
            lease_token=uuid4(),
            lease_expires_at=_now() + timedelta(minutes=2),
        ),
        attempt_id=uuid4(),
        corpus_language="en",
        url=None,
        pdf_content=b"Legal PDF text",
    )


def _now() -> datetime:
    return datetime(2026, 8, 17, 12, 0, tzinfo=UTC)

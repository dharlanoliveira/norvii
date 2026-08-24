"""One-attempt URL acquisition, extraction, publication, and safe failure mapping."""

from __future__ import annotations

from dataclasses import dataclass, replace
from typing import TYPE_CHECKING, Protocol

from norvii_ingestion.acquisition.https import (
    AcquisitionError,
    AcquisitionLimitError,
    UnsafeUrlError,
    UnsupportedContentError,
)
from norvii_ingestion.domain.artifacts import PublicationCommand
from norvii_ingestion.domain.models import (
    FailureCategory,
    OriginCapture,
    SafeFailure,
    Sha256,
    SourceKind,
)
from norvii_ingestion.enrichment.chunking import LegalChunker
from norvii_ingestion.enrichment.embedding import EmbeddingProviderError
from norvii_ingestion.extraction.html import ExtractionError
from norvii_ingestion.extraction.pdf import PdfExtractionError
from norvii_ingestion.publication.postgres.repository import WorkRepositoryError
from norvii_ingestion.semantic.extraction import ExtractionProviderError

if TYPE_CHECKING:
    from collections.abc import Callable
    from datetime import datetime
    from uuid import UUID

    from norvii_ingestion.acquisition.https import Acquisition
    from norvii_ingestion.domain.artifacts import DocumentArtifact
    from norvii_ingestion.domain.models import IngestionWork
    from norvii_ingestion.enrichment.embedding import EmbeddingProvider
    from norvii_ingestion.semantic.extraction import SemanticExtractor

_FAILURE_MAPPINGS: tuple[tuple[type[Exception], FailureCategory], ...] = (
    (UnsafeUrlError, FailureCategory.UNSAFE_URL),
    (AcquisitionLimitError, FailureCategory.PAYLOAD_TOO_LARGE),
    (UnsupportedContentError, FailureCategory.UNSUPPORTED_CONTENT),
    (AcquisitionError, FailureCategory.ACQUISITION_FAILED),
    (ExtractionError, FailureCategory.EXTRACTION_FAILED),
    (PdfExtractionError, FailureCategory.EXTRACTION_FAILED),
    (ExtractionProviderError, FailureCategory.EXTRACTION_FAILED),
    (EmbeddingProviderError, FailureCategory.PUBLICATION_FAILED),
    (WorkRepositoryError, FailureCategory.PUBLICATION_FAILED),
)


class UrlAcquirer(Protocol):
    """Acquire one bounded validated URL response."""

    def acquire(self, url: str) -> Acquisition:
        """Return supported content and final provenance."""
        ...


class HtmlExtractorPort(Protocol):
    """Extract a complete normalized document from supported HTML."""

    def extract(self, content: bytes) -> DocumentArtifact:
        """Return a validated immutable artifact."""
        ...


class PdfExtractorPort(Protocol):
    """Extract a complete normalized document from a preserved PDF."""

    def extract(self, content: bytes) -> DocumentArtifact:
        """Return a validated immutable artifact."""
        ...


@dataclass(frozen=True, slots=True)
class ArtifactExtractors:
    """Group origin-specific extractors behind one cohesive dependency."""

    html: HtmlExtractorPort
    pdf: PdfExtractorPort


class PublicationRepository(Protocol):
    """Publish success or a categorized failure under the active lease."""

    def publish(
        self,
        work: IngestionWork,
        capture: OriginCapture,
        command: PublicationCommand,
        now: datetime,
    ) -> UUID:
        """Atomically publish one successful artifact."""
        ...

    def fail(
        self,
        work: IngestionWork,
        failure: SafeFailure,
        now: datetime,
    ) -> None:
        """Atomically record one safe terminal failure."""
        ...


class IngestionProcessor:
    """Process exactly one claim without automatic retries."""

    def __init__(  # noqa: PLR0913
        self,
        *,
        repository: PublicationRepository,
        acquirer: UrlAcquirer,
        extractors: ArtifactExtractors,
        pipeline_version: str,
        clock: Callable[[], datetime],
        embedding_provider: EmbeddingProvider,
        embedding_model: str,
        semantic_extractor: SemanticExtractor | None = None,
    ) -> None:
        self._repository = repository
        self._acquirer = acquirer
        self._extractors = extractors
        self._pipeline_version = pipeline_version
        self._clock = clock
        self._embedding_provider = embedding_provider
        self._embedding_model = embedding_model
        self._semantic_extractor = semantic_extractor

    def process(self, work: IngestionWork) -> None:
        """Acquire, extract, and publish, or persist one safe failure category."""
        try:
            content, media_type, final_url = self._origin(work)
            artifact = self._extract(work, content)
            semantic_extraction = (
                self._semantic_extractor.extract(artifact)
                if self._semantic_extractor is not None
                else None
            )
            retrieval_chunks = LegalChunker().chunk(artifact)
            embeddings = self._embedding_provider.embed(
                tuple(chunk.text for chunk in retrieval_chunks)
            )
            if len(embeddings) != len(retrieval_chunks):
                raise ValueError(  # noqa: TRY301
                    "embedding provider returned an unexpected item count"
                )
            enriched_chunks = tuple(
                replace(chunk, embedding=embedding, embedding_model=self._embedding_model)
                for chunk, embedding in zip(retrieval_chunks, embeddings, strict=True)
            )
            now = self._clock()
            capture = OriginCapture(
                content_sha256=Sha256.from_bytes(content),
                captured_at=now,
                media_type=media_type,
                byte_size=len(content),
                final_url=final_url,
            )
            self._repository.publish(
                work,
                capture,
                PublicationCommand(
                    work_id=work.claim.work_id,
                    lease_token=work.claim.lease_token,
                    pipeline_version=self._pipeline_version,
                    origin_sha256=capture.content_sha256,
                    artifact=artifact,
                    retrieval_chunks=enriched_chunks,
                    semantic_extraction=semantic_extraction,
                ),
                now,
            )
        except Exception as error:  # noqa: BLE001 - boundary converts failures to safe categories
            self._repository.fail(work, _safe_failure(error), self._clock())

    def _origin(self, work: IngestionWork) -> tuple[bytes, str, str | None]:
        if work.claim.source_kind is SourceKind.URL and work.url is not None:
            acquisition = self._acquirer.acquire(work.url)
            return acquisition.content, acquisition.media_type, acquisition.final_url
        if work.claim.source_kind is SourceKind.PDF and work.pdf_content is not None:
            return work.pdf_content, "application/pdf", None
        raise UnsupportedContentError("The source origin is unavailable.")

    def _extract(self, work: IngestionWork, content: bytes) -> DocumentArtifact:
        if work.claim.source_kind is SourceKind.PDF:
            return self._extractors.pdf.extract(content)
        return self._extractors.html.extract(content)


def _category(error: Exception) -> FailureCategory:
    for error_type, category in _FAILURE_MAPPINGS:
        if isinstance(error, error_type):
            return category
    return FailureCategory.INTERNAL_ERROR


def _safe_failure(error: Exception) -> SafeFailure:
    detail = error.reason.value if isinstance(error, AcquisitionError) else None
    return SafeFailure(_category(error), detail)

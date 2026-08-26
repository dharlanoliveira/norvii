"""Transactional PostgreSQL queue claim and immutable artifact publication."""

from __future__ import annotations

from dataclasses import dataclass, replace
from datetime import datetime, timedelta
from typing import TYPE_CHECKING, cast
from uuid import UUID, uuid4, uuid5

import psycopg

from norvii_ingestion.domain.models import (
    IngestionWork,
    SourceKind,
    WorkClaim,
    WorkReason,
)
from norvii_ingestion.semantic import SemanticAssertion, SemanticEntity, SemanticExtraction

if TYPE_CHECKING:
    from norvii_ingestion.domain.artifacts import PublicationCommand
    from norvii_ingestion.domain.models import OriginCapture, SafeFailure
    from norvii_ingestion.publication.persistence.config import PostgresConfiguration


class WorkRepositoryError(RuntimeError):
    """Report a safe queue or publication transaction failure."""

    def __init__(self, message: str, detail: str = "repository_operation_failed") -> None:
        super().__init__(message)
        self.detail = detail


_RETRIEVAL_CHUNK_NAMESPACE = UUID("dec70116-ff61-48f5-8d67-dbf457330dd2")
_SEMANTIC_ARTIFACT_NAMESPACE = UUID("e32667d2-998c-4b49-a814-a9380de0d4a3")


@dataclass(frozen=True, slots=True)
class _Completion:
    document_id: UUID
    command: PublicationCommand
    capture: OriginCapture
    now: datetime


@dataclass(frozen=True, slots=True)
class _RevisionData:
    capture: OriginCapture
    extracted_content_sha256: str
    now: datetime


class PostgresWorkRepository:
    """Own queue claim and atomic artifact publication transactions."""

    def __init__(
        self,
        connection: psycopg.Connection[tuple[object, ...]],
        pipeline_version: str = "corpus-ingestion-v3",
    ) -> None:
        self.connection = connection
        self._pipeline_version = pipeline_version

    @classmethod
    def connect(
        cls,
        configuration: PostgresConfiguration,
        timeout_seconds: int,
    ) -> PostgresWorkRepository:
        """Open a bounded canonical-store connection without credential URLs."""
        try:
            connection = psycopg.connect(
                host=configuration.host,
                port=configuration.port,
                dbname=configuration.database,
                user=configuration.user,
                password=configuration.password,
                connect_timeout=timeout_seconds,
                options=f"-c statement_timeout={timeout_seconds * 1000}",
            )
        except psycopg.Error as error:
            raise WorkRepositoryError("Connect to ingestion work storage failed.") from error
        return cls(connection)

    def claim(
        self,
        worker_id: str,
        lease_duration: timedelta,
        now: datetime,
    ) -> IngestionWork | None:
        """Atomically lease the oldest pending work and create its attempt."""
        if not worker_id.strip() or lease_duration <= timedelta(0):
            raise ValueError("worker identity and a positive lease duration are required")
        lease_token = uuid4()
        attempt_id = uuid4()
        lease_expires_at = now + lease_duration
        try:
            with self.connection.transaction(), self.connection.cursor() as cursor:
                self._recover_expired(cursor, now)
                cursor.execute(
                    """
                    SELECT w.id, w.corpus_id, w.source_id, s.kind, w.reason,
                           c.language, u.submitted_url, p.content,
                           COALESCE((
                               SELECT max(a.attempt_number)
                               FROM processing_attempts a
                               WHERE a.work_id = w.id
                           ), 0) + 1
                    FROM ingestion_work w
                    JOIN sources s ON s.id = w.source_id AND s.corpus_id = w.corpus_id
                    JOIN corpora c ON c.id = w.corpus_id
                    LEFT JOIN url_origins u
                      ON u.source_id = s.id AND u.corpus_id = s.corpus_id
                    LEFT JOIN pdf_origins p
                      ON p.source_id = s.id AND p.corpus_id = s.corpus_id
                    WHERE w.status = 'pending'
                    ORDER BY w.requested_at, w.id
                    FOR UPDATE OF w, s SKIP LOCKED
                    LIMIT 1
                    """
                )
                row = cursor.fetchone()
                if row is None:
                    return None
                work_id = cast("UUID", row[0])
                corpus_id = cast("UUID", row[1])
                source_id = cast("UUID", row[2])
                source_kind = SourceKind(cast("str", row[3]))
                reason = WorkReason(cast("str", row[4]))
                corpus_language = cast("str", row[5])
                url = cast("str | None", row[6])
                pdf_content = cast("bytes | None", row[7])
                attempt_number = cast("int", row[8])
                cursor.execute(
                    """
                    UPDATE ingestion_work
                    SET status = 'leased', lease_token = %s, worker_id = %s,
                        lease_expires_at = %s, updated_at = %s
                    WHERE id = %s
                    """,
                    (lease_token, worker_id, lease_expires_at, now, work_id),
                )
                cursor.execute(
                    """
                    UPDATE sources
                    SET processing_status = 'processing', version = version + 1,
                        updated_at = %s
                    WHERE corpus_id = %s AND id = %s
                    """,
                    (now, corpus_id, source_id),
                )
                cursor.execute(
                    """
                    INSERT INTO processing_attempts (
                        id, work_id, source_id, corpus_id, attempt_number,
                        pipeline_version, status, lease_token, worker_id, started_at
                    ) VALUES (%s, %s, %s, %s, %s, %s, 'processing', %s, %s, %s)
                    """,
                    (
                        attempt_id,
                        work_id,
                        source_id,
                        corpus_id,
                        attempt_number,
                        self._pipeline_version,
                        lease_token,
                        worker_id,
                        now,
                    ),
                )
        except psycopg.Error as error:
            raise WorkRepositoryError("Claim ingestion work failed.") from error
        return IngestionWork(
            claim=WorkClaim(
                work_id=work_id,
                corpus_id=corpus_id,
                source_id=source_id,
                source_kind=source_kind,
                reason=reason,
                lease_token=lease_token,
                lease_expires_at=lease_expires_at,
            ),
            attempt_id=attempt_id,
            corpus_language=corpus_language,
            url=url,
            pdf_content=pdf_content,
        )

    def renew(
        self,
        work: IngestionWork,
        lease_duration: timedelta,
        now: datetime,
    ) -> datetime:
        """Extend an active owned lease before it expires."""
        if lease_duration <= timedelta(0):
            raise ValueError("lease renewal duration must be positive")
        expires_at = now + lease_duration
        try:
            with self.connection.transaction(), self.connection.cursor() as cursor:
                cursor.execute(
                    """
                    UPDATE ingestion_work
                    SET lease_expires_at = %s, updated_at = %s
                    WHERE id = %s AND source_id = %s AND corpus_id = %s
                      AND status = 'leased' AND lease_token = %s
                      AND lease_expires_at >= %s
                    """,
                    (
                        expires_at,
                        now,
                        work.claim.work_id,
                        work.claim.source_id,
                        work.claim.corpus_id,
                        work.claim.lease_token,
                        now,
                    ),
                )
                if cursor.rowcount != 1:
                    raise WorkRepositoryError("The ingestion lease is unavailable.")
        except psycopg.Error as error:
            raise WorkRepositoryError("Renew ingestion lease failed.") from error
        return expires_at

    @staticmethod
    def _recover_expired(cursor: psycopg.Cursor[tuple[object, ...]], now: datetime) -> None:
        cursor.execute(
            """
            UPDATE processing_attempts a
            SET status = 'failed', finished_at = %s,
                failure_category = 'lease_expired',
                duration_milliseconds = GREATEST(
                    0,
                    FLOOR(EXTRACT(EPOCH FROM (%s - a.started_at)) * 1000)
                )::bigint
            FROM ingestion_work w
            WHERE a.work_id = w.id AND a.status = 'processing'
              AND w.status = 'leased' AND w.lease_expires_at < %s
            """,
            (now, now, now),
        )
        cursor.execute(
            """
            UPDATE sources s
            SET processing_status = 'pending', latest_failure_category = 'lease_expired',
                version = version + 1, updated_at = %s
            FROM ingestion_work w
            WHERE s.id = w.source_id AND s.corpus_id = w.corpus_id
              AND w.status = 'leased' AND w.lease_expires_at < %s
            """,
            (now, now),
        )
        cursor.execute(
            """
            UPDATE ingestion_work
            SET status = 'pending', lease_token = NULL, worker_id = NULL,
                lease_expires_at = NULL, requested_at = %s, updated_at = %s
            WHERE status = 'leased' AND lease_expires_at < %s
            """,
            (now, now, now),
        )

    def publish(
        self,
        work: IngestionWork,
        capture: OriginCapture,
        command: PublicationCommand,
        now: datetime,
    ) -> UUID:
        """Atomically publish or reuse immutable artifacts while preserving the active lease."""
        command.validate()
        if (
            command.work_id != work.claim.work_id
            or command.lease_token != work.claim.lease_token
            or command.origin_sha256 != capture.content_sha256
        ):
            raise ValueError("publication identity does not match the claimed work")
        try:
            with self.connection.transaction(), self.connection.cursor() as cursor:
                self._lock_lease(cursor, work, now)
                revision_id = self._upsert_revision(
                    cursor,
                    work,
                    _RevisionData(capture, str(command.artifact.text_sha256), now),
                )
                document_id, created = self._upsert_document(
                    cursor, work, revision_id, command, now
                )
                if created:
                    self._insert_units(cursor, document_id, command)
                    self._insert_retrieval_chunks(cursor, work, document_id, command)
                    self._insert_semantic_extraction(cursor, work, document_id, command, now)
                else:
                    self._insert_missing_semantic_extraction(
                        cursor, work, document_id, command, now
                    )
        except psycopg.Error as error:
            raise WorkRepositoryError(
                "Publish ingestion artifacts failed.", _repository_error_detail(error)
            ) from error
        return document_id

    def complete(
        self,
        work: IngestionWork,
        capture: OriginCapture,
        command: PublicationCommand,
        document_id: UUID,
        now: datetime,
    ) -> None:
        """Mark one leased ingestion attempt ready only after derived releases are available."""
        try:
            with self.connection.transaction(), self.connection.cursor() as cursor:
                self._lock_lease(cursor, work, now)
                self._complete_attempt(
                    cursor,
                    work,
                    _Completion(document_id, command, capture, now),
                )
        except psycopg.Error as error:
            raise WorkRepositoryError(
                "Complete ingestion attempt failed.", _repository_error_detail(error)
            ) from error

    def close(self) -> None:
        """Release the canonical-store connection."""
        self.connection.close()

    def fail(
        self,
        work: IngestionWork,
        failure: SafeFailure,
        now: datetime,
    ) -> None:
        """Atomically fail the attempt, preserve any ready document, and clear its lease."""
        try:
            with self.connection.transaction(), self.connection.cursor() as cursor:
                self._lock_lease(cursor, work, now)
                cursor.execute(
                    """
                    UPDATE processing_attempts
                    SET status = 'failed', finished_at = %s,
                        failure_category = %s, failure_detail = %s,
                        duration_milliseconds = GREATEST(
                            0,
                            FLOOR(EXTRACT(EPOCH FROM (%s - started_at)) * 1000)
                        )::bigint
                    WHERE id = %s AND lease_token = %s
                    """,
                    (
                        now,
                        failure.category.value,
                        failure.detail,
                        now,
                        work.attempt_id,
                        work.claim.lease_token,
                    ),
                )
                cursor.execute(
                    """
                    UPDATE sources
                    SET processing_status = 'failed', latest_failure_category = %s,
                        version = version + 1, updated_at = %s
                    WHERE corpus_id = %s AND id = %s
                    """,
                    (
                        failure.category.value,
                        now,
                        work.claim.corpus_id,
                        work.claim.source_id,
                    ),
                )
                cursor.execute(
                    """
                    UPDATE ingestion_work
                    SET status = 'failed', lease_token = NULL, worker_id = NULL,
                        lease_expires_at = NULL, updated_at = %s
                    WHERE id = %s AND lease_token = %s
                    """,
                    (now, work.claim.work_id, work.claim.lease_token),
                )
        except psycopg.Error as error:
            raise WorkRepositoryError("Record ingestion failure failed.") from error

    @staticmethod
    def _lock_lease(
        cursor: psycopg.Cursor[tuple[object, ...]], work: IngestionWork, now: datetime
    ) -> None:
        cursor.execute(
            """
            SELECT 1 FROM ingestion_work
            WHERE id = %s AND source_id = %s AND corpus_id = %s
              AND status = 'leased' AND lease_token = %s AND lease_expires_at >= %s
            FOR UPDATE
            """,
            (
                work.claim.work_id,
                work.claim.source_id,
                work.claim.corpus_id,
                work.claim.lease_token,
                now,
            ),
        )
        if cursor.fetchone() is None:
            raise WorkRepositoryError("The ingestion lease is unavailable.")

    def _upsert_revision(
        self,
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        data: _RevisionData,
    ) -> UUID:
        revision_id = uuid4()
        cursor.execute(
            """
            INSERT INTO source_revisions (
                id, source_id, corpus_id, attempt_id, content_sha256,
                captured_at, media_type, byte_size, pipeline_version,
                final_url, extracted_content_sha256, created_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (source_id, content_sha256) DO NOTHING
            RETURNING id
            """,
            (
                revision_id,
                work.claim.source_id,
                work.claim.corpus_id,
                work.attempt_id,
                str(data.capture.content_sha256),
                data.capture.captured_at,
                data.capture.media_type,
                data.capture.byte_size,
                self._pipeline_version,
                data.capture.final_url,
                data.extracted_content_sha256,
                data.now,
            ),
        )
        row = cursor.fetchone()
        if row is not None:
            return cast("UUID", row[0])
        cursor.execute(
            "SELECT id FROM source_revisions WHERE source_id = %s AND content_sha256 = %s",
            (work.claim.source_id, str(data.capture.content_sha256)),
        )
        return cast("UUID", cursor.fetchone()[0])  # type: ignore[index]

    def _upsert_document(
        self,
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        revision_id: UUID,
        command: PublicationCommand,
        now: datetime,
    ) -> tuple[UUID, bool]:
        document_id = uuid4()
        cursor.execute(
            """
            INSERT INTO document_versions (
                id, source_revision_id, source_id, corpus_id, pipeline_version,
                text_content, text_sha256, publication_status, published_at, created_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, 'published', %s, %s)
            ON CONFLICT (source_revision_id, pipeline_version) DO NOTHING
            RETURNING id
            """,
            (
                document_id,
                revision_id,
                work.claim.source_id,
                work.claim.corpus_id,
                command.pipeline_version,
                command.artifact.text,
                str(command.artifact.text_sha256),
                now,
                now,
            ),
        )
        row = cursor.fetchone()
        if row is not None:
            return cast("UUID", row[0]), True
        cursor.execute(
            """
            SELECT id FROM document_versions
            WHERE source_revision_id = %s AND pipeline_version = %s
            """,
            (revision_id, command.pipeline_version),
        )
        return cast("UUID", cursor.fetchone()[0]), False  # type: ignore[index]

    @staticmethod
    def _insert_units(
        cursor: psycopg.Cursor[tuple[object, ...]],
        document_id: UUID,
        command: PublicationCommand,
    ) -> None:
        cursor.executemany(
            """
            INSERT INTO document_units (
                id, document_id, parent_id, kind, ordinal, marker, label,
                start_offset, end_offset, start_page, end_page, locator, canonical_locator,
                content_sha256
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """,
            [
                (
                    unit.id,
                    document_id,
                    unit.parent_id,
                    unit.kind.value,
                    unit.ordinal,
                    unit.marker,
                    unit.label,
                    unit.start_offset,
                    unit.end_offset,
                    unit.start_page,
                    unit.end_page,
                    unit.locator,
                    unit.canonical_locator,
                    str(unit.content_sha256),
                )
                for unit in command.artifact.units
            ],
        )

    @staticmethod
    def _insert_retrieval_chunks(
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        document_id: UUID,
        command: PublicationCommand,
    ) -> None:
        """Persist enriched source spans and vectors in the publication transaction."""
        chunks = command.retrieval_chunks
        cursor.executemany(
            """
            INSERT INTO retrieval_chunks (
                id, corpus_id, source_id, document_id, unit_id, ordinal,
                start_offset, end_offset, content, content_sha256, context_locator,
                embedding, embedding_model, enrichment_status
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s::vector, %s, 'ready')
            """,
            [
                (
                    retrieval_chunk_id_for_document(document_id, chunk.id),
                    work.claim.corpus_id,
                    work.claim.source_id,
                    document_id,
                    chunk.source_unit_id,
                    ordinal,
                    chunk.start_offset,
                    chunk.end_offset,
                    chunk.text,
                    str(chunk.text_sha256),
                    chunk.context_locator,
                    _vector_literal(chunk.embedding),
                    chunk.embedding_model,
                )
                for ordinal, chunk in enumerate(chunks)
            ],
        )

    @staticmethod
    def _insert_semantic_extraction(
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        document_id: UUID,
        command: PublicationCommand,
        now: datetime,
    ) -> None:
        extraction = command.semantic_extraction
        if extraction is None:
            return
        extraction = _document_scoped_semantic_extraction(document_id, extraction)
        cursor.execute(
            """
            INSERT INTO semantic_extraction_runs (
                id, corpus_id, source_id, document_id, extraction_version, model_identifier,
                input_sha256, status, input_tokens, output_tokens, duration_milliseconds,
                created_at, completed_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, 'ready', %s, %s, %s, %s, %s)
            """,
            (
                extraction.id,
                work.claim.corpus_id,
                work.claim.source_id,
                document_id,
                extraction.extraction_version,
                extraction.model_identifier,
                str(extraction.input_sha256),
                extraction.input_tokens,
                extraction.output_tokens,
                extraction.duration_milliseconds,
                now,
                now,
            ),
        )
        cursor.executemany(
            """
            INSERT INTO semantic_entities (
                id, extraction_run_id, corpus_id, source_id, document_id, evidence_unit_id,
                entity_type, label, normalized_label, validation_status
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, 'supported')
            """,
            [
                (
                    entity.id,
                    extraction.id,
                    work.claim.corpus_id,
                    work.claim.source_id,
                    document_id,
                    entity.evidence_unit_id,
                    entity.entity_type,
                    entity.label,
                    entity.normalized_label,
                )
                for entity in extraction.entities
            ],
        )
        cursor.executemany(
            """
            INSERT INTO normative_assertions (
                id, extraction_run_id, corpus_id, source_id, document_id, subject_entity_id,
                object_entity_id, establishing_unit_id, evidence_unit_id, predicate, qualifier,
                validation_status
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 'supported')
            """,
            [
                (
                    assertion.id,
                    extraction.id,
                    work.claim.corpus_id,
                    work.claim.source_id,
                    document_id,
                    assertion.subject_entity_id,
                    assertion.object_entity_id,
                    assertion.establishing_unit_id,
                    assertion.evidence_unit_id,
                    assertion.predicate,
                    assertion.qualifier,
                )
                for assertion in extraction.assertions
            ],
        )

    def _insert_missing_semantic_extraction(
        self,
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        document_id: UUID,
        command: PublicationCommand,
        now: datetime,
    ) -> None:
        """Backfill required graph artifacts when a historical document version is reused."""
        extraction = command.semantic_extraction
        if extraction is None:
            return
        extraction = _document_scoped_semantic_extraction(document_id, extraction)
        extraction = replace(
            extraction,
            id=self._insert_missing_semantic_run(cursor, work, document_id, extraction, now),
        )
        extraction = self._insert_missing_semantic_entities(cursor, work, document_id, extraction)
        self._insert_missing_normative_assertions(cursor, work, document_id, extraction)

    @staticmethod
    def _insert_missing_semantic_run(
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        document_id: UUID,
        extraction: SemanticExtraction,
        now: datetime,
    ) -> UUID:
        """Return the persisted run identity for a logically equivalent extraction."""
        cursor.execute(
            """
            INSERT INTO semantic_extraction_runs (
                id, corpus_id, source_id, document_id, extraction_version, model_identifier,
                input_sha256, status, input_tokens, output_tokens, duration_milliseconds,
                created_at, completed_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, 'ready', %s, %s, %s, %s, %s)
            ON CONFLICT (document_id, extraction_version, input_sha256) DO NOTHING
            RETURNING id
            """,
            (
                extraction.id,
                work.claim.corpus_id,
                work.claim.source_id,
                document_id,
                extraction.extraction_version,
                extraction.model_identifier,
                str(extraction.input_sha256),
                extraction.input_tokens,
                extraction.output_tokens,
                extraction.duration_milliseconds,
                now,
                now,
            ),
        )
        row = cursor.fetchone()
        if row is not None:
            return cast("UUID", row[0])
        cursor.execute(
            """
            SELECT id FROM semantic_extraction_runs
            WHERE document_id = %s AND extraction_version = %s AND input_sha256 = %s
            """,
            (document_id, extraction.extraction_version, str(extraction.input_sha256)),
        )
        return cast("UUID", cursor.fetchone()[0])  # type: ignore[index]

    @staticmethod
    def _insert_missing_semantic_entities(
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        document_id: UUID,
        extraction: SemanticExtraction,
    ) -> SemanticExtraction:
        """Persist entities and return assertions bound to their canonical identities."""
        cursor.executemany(
            """
            INSERT INTO semantic_entities (
                id, extraction_run_id, corpus_id, source_id, document_id, evidence_unit_id,
                entity_type, label, normalized_label, validation_status
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, 'supported')
            ON CONFLICT (extraction_run_id, entity_type, normalized_label, evidence_unit_id)
            DO NOTHING
            """,
            [
                (
                    entity.id,
                    extraction.id,
                    work.claim.corpus_id,
                    work.claim.source_id,
                    document_id,
                    entity.evidence_unit_id,
                    entity.entity_type,
                    entity.label,
                    entity.normalized_label,
                )
                for entity in extraction.entities
            ],
        )
        cursor.execute(
            """
            SELECT id, entity_type, normalized_label, evidence_unit_id
            FROM semantic_entities
            WHERE extraction_run_id = %s
            """,
            (extraction.id,),
        )
        canonical_ids = {
            (str(entity_type), str(normalized_label), evidence_unit_id): entity_id
            for entity_id, entity_type, normalized_label, evidence_unit_id in cursor.fetchall()
        }
        replacements = {
            entity.id: cast(
                "UUID",
                canonical_ids[
                    (entity.entity_type, entity.normalized_label, entity.evidence_unit_id)
                ],
            )
            for entity in extraction.entities
        }
        return replace(
            extraction,
            entities=tuple(
                replace(entity, id=replacements[entity.id]) for entity in extraction.entities
            ),
            assertions=tuple(
                replace(
                    assertion,
                    subject_entity_id=replacements[assertion.subject_entity_id],
                    object_entity_id=replacements[assertion.object_entity_id],
                )
                for assertion in extraction.assertions
            ),
        )

    @staticmethod
    def _insert_missing_normative_assertions(
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        document_id: UUID,
        extraction: SemanticExtraction,
    ) -> None:
        cursor.executemany(
            """
            INSERT INTO normative_assertions (
                id, extraction_run_id, corpus_id, source_id, document_id, subject_entity_id,
                object_entity_id, establishing_unit_id, evidence_unit_id, predicate, qualifier,
                validation_status
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, 'supported')
            ON CONFLICT (
                extraction_run_id, subject_entity_id, object_entity_id, predicate,
                establishing_unit_id, evidence_unit_id
            ) DO NOTHING
            """,
            [
                (
                    assertion.id,
                    extraction.id,
                    work.claim.corpus_id,
                    work.claim.source_id,
                    document_id,
                    assertion.subject_entity_id,
                    assertion.object_entity_id,
                    assertion.establishing_unit_id,
                    assertion.evidence_unit_id,
                    assertion.predicate,
                    assertion.qualifier,
                )
                for assertion in extraction.assertions
            ],
        )

    @staticmethod
    def _complete_attempt(
        cursor: psycopg.Cursor[tuple[object, ...]],
        work: IngestionWork,
        completion: _Completion,
    ) -> None:
        cursor.execute(
            """
            UPDATE sources
            SET processing_status = 'ready', latest_failure_category = NULL,
                latest_ready_document_id = %s, version = version + 1, updated_at = %s
            WHERE corpus_id = %s AND id = %s
            """,
            (
                completion.document_id,
                completion.now,
                work.claim.corpus_id,
                work.claim.source_id,
            ),
        )
        cursor.execute(
            """
            UPDATE processing_attempts
            SET status = 'succeeded', finished_at = %s,
                acquired_byte_count = %s, normalized_character_count = %s,
                unit_count = %s,
                duration_milliseconds = GREATEST(
                    0,
                    FLOOR(EXTRACT(EPOCH FROM (%s - started_at)) * 1000)
                )::bigint
            WHERE id = %s AND lease_token = %s
            """,
            (
                completion.now,
                completion.capture.byte_size,
                len(completion.command.artifact.text),
                len(completion.command.artifact.units),
                completion.now,
                work.attempt_id,
                work.claim.lease_token,
            ),
        )
        cursor.execute(
            """
            UPDATE ingestion_work
            SET status = 'succeeded', lease_token = NULL, worker_id = NULL,
                lease_expires_at = NULL, updated_at = %s
            WHERE id = %s AND lease_token = %s
            """,
            (completion.now, work.claim.work_id, work.claim.lease_token),
        )


def _vector_literal(vector: tuple[float, ...] | None) -> str:
    if vector is None:
        raise ValueError("retrieval chunk embedding is required")
    return "[" + ",".join(format(value, ".17g") for value in vector) + "]"


def retrieval_chunk_id_for_document(document_id: UUID, logical_chunk_id: UUID) -> UUID:
    """Scope a stable logical chunk identity to its immutable document version."""
    return uuid5(_RETRIEVAL_CHUNK_NAMESPACE, f"{document_id}:{logical_chunk_id}")


def _document_scoped_semantic_extraction(
    document_id: UUID, extraction: SemanticExtraction
) -> SemanticExtraction:
    """Bind logical semantic output to one immutable document version."""
    entity_ids = {
        entity.id: uuid5(_SEMANTIC_ARTIFACT_NAMESPACE, f"{document_id}:{entity.id}")
        for entity in extraction.entities
    }
    entities = tuple(
        SemanticEntity(
            id=entity_ids[entity.id],
            evidence_unit_id=entity.evidence_unit_id,
            entity_type=entity.entity_type,
            label=entity.label,
            normalized_label=entity.normalized_label,
        )
        for entity in extraction.entities
    )
    assertions = tuple(
        SemanticAssertion(
            id=uuid5(_SEMANTIC_ARTIFACT_NAMESPACE, f"{document_id}:{assertion.id}"),
            subject_entity_id=entity_ids[assertion.subject_entity_id],
            object_entity_id=entity_ids[assertion.object_entity_id],
            establishing_unit_id=assertion.establishing_unit_id,
            evidence_unit_id=assertion.evidence_unit_id,
            predicate=assertion.predicate,
            qualifier=assertion.qualifier,
        )
        for assertion in extraction.assertions
    )
    return SemanticExtraction(
        id=uuid5(_SEMANTIC_ARTIFACT_NAMESPACE, f"{document_id}:{extraction.id}"),
        extraction_version=extraction.extraction_version,
        model_identifier=extraction.model_identifier,
        input_sha256=extraction.input_sha256,
        input_tokens=extraction.input_tokens,
        output_tokens=extraction.output_tokens,
        duration_milliseconds=extraction.duration_milliseconds,
        entities=entities,
        assertions=assertions,
    )


def _repository_error_detail(error: psycopg.Error) -> str:
    return (
        f"repository_sqlstate_{error.sqlstate}" if error.sqlstate else "repository_operation_failed"
    )

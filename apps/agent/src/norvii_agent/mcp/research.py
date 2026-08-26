"""Snapshot-scoped, read-only research queries for MCP tools."""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING
from uuid import UUID

import psycopg

if TYPE_CHECKING:
    from norvii_agent.config import AgentConfig


@dataclass(frozen=True, slots=True)
class MCPResearchRepository:
    """Query only enabled corpora and their immutable active snapshots."""

    configuration: AgentConfig

    def list_corpora(self) -> list[dict[str, object]]:
        """Return the bounded set of enabled corpora."""
        return self._query(
            """
            SELECT id::text AS id, name, description, language, jurisdiction
            FROM corpora WHERE status = 'enabled' ORDER BY language, name, id LIMIT 50
            """
        )

    def list_documents(self, corpus_id: UUID) -> dict[str, object]:
        """Return documents belonging to one active immutable snapshot."""
        snapshot_id = self._snapshot(corpus_id)
        if snapshot_id is None:
            return _not_found()
        documents = self._query(
            """
            SELECT s.id::text AS source_id, s.title, d.id::text AS document_version_id,
                   d.pipeline_version
            FROM corpus_snapshot_documents sd
            JOIN sources s ON s.id = sd.source_id AND s.corpus_id = sd.corpus_id
            JOIN document_versions d ON d.id = sd.document_id AND d.corpus_id = sd.corpus_id
            WHERE sd.corpus_id = %s AND sd.snapshot_id = %s
            ORDER BY s.title, d.id LIMIT 50
            """,
            (corpus_id, snapshot_id),
        )
        return _completed(snapshot_id, documents=documents)

    def search(self, corpus_id: UUID, query: str) -> dict[str, object]:
        """Return bounded lexical evidence from one active snapshot."""
        snapshot_id = self._snapshot(corpus_id)
        if snapshot_id is None:
            return _not_found()
        term = query.strip()
        if not term:
            return _invalid_input()
        evidence = self._query(
            """
            SELECT s.id::text AS source_id, s.title AS source_title,
                   d.id::text AS document_version_id, u.locator AS unit_locator,
                   u.start_offset, u.end_offset,
                   substring(d.text_content FROM evidence.start_offset + 1 FOR LEAST(1000,
                   u.end_offset - u.start_offset)) AS excerpt
            FROM corpus_snapshot_documents sd
            JOIN document_versions d ON d.id = sd.document_id AND d.corpus_id = sd.corpus_id
            JOIN sources s ON s.id = sd.source_id AND s.corpus_id = sd.corpus_id
            JOIN document_units u ON u.document_id = d.id
            WHERE sd.corpus_id = %s AND sd.snapshot_id = %s
              AND d.text_content ILIKE %s
            ORDER BY s.title, u.ordinal LIMIT 8
            """,
            (corpus_id, snapshot_id, f"%{term}%"),
        )
        return _completed(snapshot_id, evidence=evidence)

    def document_metadata(self, corpus_id: UUID, document_id: UUID) -> dict[str, object]:
        """Return metadata for an immutable document in the active snapshot."""
        snapshot_id = self._snapshot(corpus_id)
        if snapshot_id is None:
            return _not_found()
        documents = self._query(
            """
            SELECT s.id::text AS source_id, s.title, d.id::text AS document_version_id,
                   d.pipeline_version, d.published_at::text AS published_at
            FROM corpus_snapshot_documents sd
            JOIN document_versions d ON d.id = sd.document_id AND d.corpus_id = sd.corpus_id
            JOIN sources s ON s.id = sd.source_id AND s.corpus_id = sd.corpus_id
            WHERE sd.corpus_id = %s AND sd.snapshot_id = %s AND d.id = %s
            """,
            (corpus_id, snapshot_id, document_id),
        )
        return _completed(snapshot_id, document=documents[0]) if documents else _not_found()

    def article(self, corpus_id: UUID, document_id: UUID, locator: str) -> dict[str, object]:
        """Return one immutable legal unit by its locator."""
        snapshot_id = self._snapshot(corpus_id)
        if snapshot_id is None or not locator.strip():
            return _not_found() if snapshot_id is None else _invalid_input()
        units = self._query(
            """
            SELECT s.id::text AS source_id, s.title AS source_title,
                   d.id::text AS document_version_id, u.locator AS unit_locator,
                   u.start_offset, u.end_offset,
                   substring(d.text_content FROM u.start_offset + 1 FOR
                   u.end_offset - u.start_offset) AS content
            FROM corpus_snapshot_documents sd
            JOIN document_versions d ON d.id = sd.document_id AND d.corpus_id = sd.corpus_id
            JOIN sources s ON s.id = sd.source_id AND s.corpus_id = sd.corpus_id
            JOIN document_units u ON u.document_id = d.id
            WHERE sd.corpus_id = %s AND sd.snapshot_id = %s AND d.id = %s AND u.locator = %s
            """,
            (corpus_id, snapshot_id, document_id, locator.strip()),
        )
        return _completed(snapshot_id, article=units[0]) if units else _not_found()

    def related(self, corpus_id: UUID, query: str) -> dict[str, object]:
        """Return bounded evidence-backed normative assertions."""
        snapshot_id = self._snapshot(corpus_id)
        if snapshot_id is None:
            return _not_found()
        term = query.strip().lower()
        if not term:
            return _invalid_input()
        if not self._graph_is_ready(corpus_id, snapshot_id):
            return _unavailable()
        assertions = self._query(
            """
            SELECT assertion.id::text AS assertion_id, assertion.predicate,
                   assertion.qualifier, subject.label AS subject_label,
                   object.label AS object_label,
                   s.id::text AS source_id, s.title AS source_title,
                   d.id::text AS document_version_id,
                   establishing.locator AS establishing_locator,
                   evidence.locator AS evidence_locator,
                   evidence.start_offset, evidence.end_offset,
                   substring(d.text_content FROM u.start_offset + 1 FOR LEAST(1000,
                   evidence.end_offset - evidence.start_offset)) AS evidence_excerpt
            FROM graph_releases release
            JOIN graph_release_assertions member ON member.graph_release_id = release.id
            JOIN normative_assertions assertion ON assertion.id = member.normative_assertion_id
            JOIN semantic_entities subject ON subject.id = assertion.subject_entity_id
            JOIN semantic_entities object ON object.id = assertion.object_entity_id
            JOIN document_units establishing ON establishing.id = assertion.establishing_unit_id
            JOIN document_units evidence ON evidence.id = assertion.evidence_unit_id
            JOIN document_versions d ON d.id = assertion.document_id
            JOIN sources s ON s.id = assertion.source_id AND s.corpus_id = assertion.corpus_id
            WHERE release.corpus_id = %s AND release.snapshot_id = %s AND release.status = 'ready'
              AND (lower(subject.label) LIKE %s OR lower(object.label) LIKE %s)
            ORDER BY evidence.locator, assertion.id LIMIT 8
            """,
            (corpus_id, snapshot_id, f"%{term}%", f"%{term}%"),
        )
        return _completed(snapshot_id, assertions=assertions)

    def compare(self, corpus_id: UUID, first_id: UUID, second_id: UUID) -> dict[str, object]:
        """Return two cited provisions without legal synthesis."""
        first = self._first_article(corpus_id, first_id)
        second = self._first_article(corpus_id, second_id)
        if first is None or second is None:
            return _not_found()
        return _completed(
            str(first["snapshot_id"]), first=first["article"], second=second["article"]
        )

    def _first_article(self, corpus_id: UUID, document_id: UUID) -> dict[str, object] | None:
        snapshot_id = self._snapshot(corpus_id)
        if snapshot_id is None:
            return None
        units = self._query(
            """
            SELECT s.id::text AS source_id, s.title AS source_title,
                   d.id::text AS document_version_id,
                   u.locator AS unit_locator, u.start_offset, u.end_offset,
                   substring(d.text_content FROM u.start_offset + 1 FOR
                   u.end_offset - u.start_offset) AS content
            FROM corpus_snapshot_documents sd
            JOIN document_versions d ON d.id = sd.document_id AND d.corpus_id = sd.corpus_id
            JOIN sources s ON s.id = sd.source_id AND s.corpus_id = sd.corpus_id
            JOIN document_units u ON u.document_id = d.id
            WHERE sd.corpus_id = %s AND sd.snapshot_id = %s AND d.id = %s
            ORDER BY CASE WHEN u.kind = 'article' THEN 0 ELSE 1 END, u.ordinal LIMIT 1
            """,
            (corpus_id, snapshot_id, document_id),
        )
        return {"snapshot_id": snapshot_id, "article": units[0]} if units else None

    def _snapshot(self, corpus_id: UUID) -> str | None:
        rows = self._query(
            """
            SELECT release.snapshot_id::text AS snapshot_id
            FROM corpus_snapshot_releases release
            JOIN corpora corpus ON corpus.id = release.corpus_id AND corpus.status = 'enabled'
            WHERE release.corpus_id = %s
            """,
            (corpus_id,),
        )
        return str(rows[0]["snapshot_id"]) if rows else None

    def active_snapshot(self, corpus_id: UUID) -> str | None:
        """Return the active published snapshot for a reusable MCP workflow."""
        return self._snapshot(corpus_id)

    def _graph_is_ready(self, corpus_id: UUID, snapshot_id: str) -> bool:
        """Return whether the snapshot has a ready graph release."""
        rows = self._query(
            """
            SELECT 1
            FROM graph_releases
            WHERE corpus_id = %s AND snapshot_id = %s AND status = 'ready'
            LIMIT 1
            """,
            (corpus_id, snapshot_id),
        )
        return bool(rows)

    def _query(
        self, statement: str, parameters: tuple[object, ...] = ()
    ) -> list[dict[str, object]]:
        with (
            psycopg.connect(
                host=self.configuration.postgres_host,
                port=self.configuration.postgres_port,
                dbname=self.configuration.postgres_database,
                user=self.configuration.postgres_user,
                password=self.configuration.postgres_password,
                connect_timeout=5,
            ) as connection,
            connection.cursor(row_factory=psycopg.rows.dict_row) as cursor,
        ):
            cursor.execute(statement, parameters)
            return [dict(row) for row in cursor.fetchall()]


def _completed(snapshot_id: str, **content: object) -> dict[str, object]:
    return {"outcome": "completed", "snapshot_id": snapshot_id, **content}


def _not_found() -> dict[str, object]:
    return {"outcome": "not_found"}


def _unavailable() -> dict[str, object]:
    return {"outcome": "unavailable"}


def _invalid_input() -> dict[str, object]:
    return {"outcome": "invalid_input"}

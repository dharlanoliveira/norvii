package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/graphrelease/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestRepositoryGetScopesReleaseToCorpusAndSnapshot(t *testing.T) {
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	completed := now.Add(time.Minute)
	want := domain.Release{
		ID: uuid.New(), CorpusID: uuid.New(), SnapshotID: uuid.New(),
		ManifestSHA256: "manifest", BuildVersion: "legal-graph-v1", Status: domain.StatusReady,
		CreatedAt: now, CompletedAt: &completed,
	}
	store := &fakeQueryer{row: fakeRow{values: []any{
		want.ID, want.CorpusID, want.SnapshotID, want.ManifestSHA256, want.BuildVersion,
		want.Status, want.FailureCategory, want.EntityCount, want.RelationshipCount,
		want.CreatedAt, want.CompletedAt,
	}}}

	got, err := NewRepository(store).Get(context.Background(), want.CorpusID, want.SnapshotID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != want.ID || store.arguments[0] != want.CorpusID || store.arguments[1] != want.SnapshotID {
		t.Fatalf("Get() = %#v, arguments = %#v; want scoped release", got, store.arguments)
	}
}

func TestRepositoryGetMapsMissingRelease(t *testing.T) {
	_, err := NewRepository(&fakeQueryer{row: fakeRow{err: pgx.ErrNoRows}}).Get(
		context.Background(), uuid.New(), uuid.New(),
	)
	if err != domain.ErrNotFound {
		t.Fatalf("Get() error = %v, want %v", err, domain.ErrNotFound)
	}
}

type fakeQueryer struct {
	row       pgx.Row
	arguments []any
}

func (fake *fakeQueryer) QueryRow(_ context.Context, _ string, arguments ...any) pgx.Row {
	fake.arguments = arguments
	return fake.row
}

type fakeRow struct {
	values []any
	err    error
}

func (row fakeRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	for index, destination := range destinations {
		switch target := destination.(type) {
		case *uuid.UUID:
			*target = row.values[index].(uuid.UUID)
		case *string:
			*target = row.values[index].(string)
		case *domain.Status:
			*target = row.values[index].(domain.Status)
		case *int:
			*target = row.values[index].(int)
		case *time.Time:
			*target = row.values[index].(time.Time)
		case **time.Time:
			*target = row.values[index].(*time.Time)
		default:
			return pgx.ErrNoRows
		}
	}
	return nil
}

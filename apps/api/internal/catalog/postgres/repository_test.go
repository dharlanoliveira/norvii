//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/catalog/domain"
	catalogpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/catalog/postgres"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
	sourcepostgres "github.com/dharlanoliveira/norvii/apps/api/internal/source/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCorpusMutationsAreAtomicAndVersioned(t *testing.T) {
	ctx, pool, _ := openPool(t)
	repository := catalogpostgres.NewRepository(pool)
	id := uuid.New()
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	createdCorpus, err := domain.NewCorpus(id, domain.Draft{
		Name: "Commercial law", Description: "Official commercial materials.",
		Language: domain.LanguageEnglish, Jurisdiction: "United States",
	}, now)
	if err != nil {
		t.Fatalf("NewCorpus() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM corpora WHERE id = $1", id)
	})

	created, err := repository.Create(ctx, createdCorpus)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Version != 1 || created.Status != domain.StatusEnabled {
		t.Fatalf("created corpus = %+v, want enabled version 1", created)
	}

	updated, err := repository.Update(ctx, id, domain.Draft{
		Name: "Updated commercial law", Description: "Updated official materials.",
		Language: domain.LanguageEnglish, Jurisdiction: "United States",
	}, 1, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Version != 2 || updated.Name != "Updated commercial law" {
		t.Fatalf("updated corpus = %+v, want normalized version 2", updated)
	}

	if _, err := repository.Update(
		ctx, id, domain.Draft{
			Name: "Stale write", Description: "Must not persist.",
			Language: domain.LanguageEnglish, Jurisdiction: "United States",
		}, 1, now.Add(2*time.Minute),
	); !errors.Is(err, catalogpostgres.ErrStaleState) {
		t.Fatalf("stale Update() error = %v, want ErrStaleState", err)
	}

	disabled, err := repository.SetStatus(
		ctx, id, domain.StatusDisabled, 2, now.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("SetStatus(disabled) error = %v", err)
	}
	enabled, err := repository.SetStatus(
		ctx, id, domain.StatusEnabled, 3, now.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatalf("SetStatus(enabled) error = %v", err)
	}
	if disabled.Status != domain.StatusDisabled || enabled.Status != domain.StatusEnabled || enabled.Version != 4 {
		t.Fatalf("lifecycle states = disabled:%+v enabled:%+v", disabled, enabled)
	}
}

func TestInitialRepositoriesAreOrderedStableAndCorpusIsolated(t *testing.T) {
	ctx, pool, configuration := openPool(t)
	catalogRepository := catalogpostgres.NewRepository(pool)
	sourceRepository := sourcepostgres.NewRepository(pool)

	corpora, err := catalogRepository.List(ctx, false)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(corpora) != 2 {
		t.Fatalf("List() count = %d, want 2", len(corpora))
	}
	if corpora[0].ID.String() != "10000000-0000-4000-8000-000000000002" || corpora[0].Language != "en" {
		t.Fatalf("first corpus = %+v, want fixed English corpus", corpora[0])
	}
	if corpora[1].ID.String() != "10000000-0000-4000-8000-000000000001" || corpora[1].Language != "pt" {
		t.Fatalf("second corpus = %+v, want fixed Portuguese corpus", corpora[1])
	}

	sources, err := sourceRepository.ListByCorpus(ctx, corpora[0].ID)
	if err != nil {
		t.Fatalf("ListByCorpus() error = %v", err)
	}
	if len(sources) != 1 || sources[0].CorpusID != corpora[0].ID {
		t.Fatalf("ListByCorpus() = %+v, want one owned source", sources)
	}
	foreignSources, err := sourceRepository.ListByCorpus(ctx, uuid.New())
	if err != nil {
		t.Fatalf("foreign ListByCorpus() error = %v", err)
	}
	if len(foreignSources) != 0 {
		t.Fatalf("foreign ListByCorpus() disclosed %+v", foreignSources)
	}

	const editedName = "Maintainer-edited English corpus"
	if _, err := pool.Exec(ctx, `UPDATE corpora SET name = $1 WHERE id = $2`, editedName, corpora[0].ID); err != nil {
		t.Fatalf("edit seeded corpus error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			UPDATE corpora
			SET name = 'European Union General Data Protection Regulation'
			WHERE id = $1`, corpora[0].ID)
	})
	migrator, err := persistence.OpenPostgresMigrator(ctx, configuration.Postgres)
	if err != nil {
		t.Fatalf("OpenPostgresMigrator() error = %v", err)
	}
	t.Cleanup(func() { _ = migrator.Close(context.Background()) })
	if _, err := persistence.NewMigrationService(migrator).Apply(ctx); err != nil {
		t.Fatalf("repeated initialization error = %v", err)
	}
	edited, err := catalogRepository.Get(ctx, corpora[0].ID, true)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if edited.Name != editedName {
		t.Fatalf("repeated initialization replaced name with %q", edited.Name)
	}
}

func openPool(t *testing.T) (context.Context, *pgxpool.Pool, persistence.Config) {
	t.Helper()
	configuration, err := persistence.LoadConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	t.Cleanup(cancel)
	poolConfig, err := pgxpool.ParseConfig("sslmode=disable")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	poolConfig.ConnConfig.Host = configuration.Postgres.Host
	poolConfig.ConnConfig.Port = configuration.Postgres.Port
	poolConfig.ConnConfig.Database = configuration.Postgres.Database
	poolConfig.ConnConfig.User = configuration.Postgres.User
	poolConfig.ConnConfig.Password = configuration.Postgres.Password
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool, configuration
}

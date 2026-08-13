package persistence

import (
	"context"
	"fmt"

	"github.com/dharlanoliveira/norvii/apps/api/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

const migrationVersionTable = "public.norvii_schema_version"

type migrationEngine interface {
	Migrate(context.Context) error
	GetCurrentVersion(context.Context) (int32, error)
}

// MigrationStatus reports the canonical schema version after a migration operation.
type MigrationStatus struct {
	CurrentVersion int32
}

// MigrationService applies and inspects ordered PostgreSQL migrations.
type MigrationService struct {
	engine migrationEngine
}

// NewMigrationService creates migration behavior around an initialized engine.
func NewMigrationService(engine migrationEngine) *MigrationService {
	return &MigrationService{engine: engine}
}

// Apply migrates to the latest embedded version and returns the resulting status.
func (service *MigrationService) Apply(ctx context.Context) (MigrationStatus, error) {
	if err := service.engine.Migrate(ctx); err != nil {
		return MigrationStatus{}, fmt.Errorf("apply PostgreSQL migrations: %w", err)
	}
	return service.Status(ctx)
}

// Status returns the current migration version without changing schema state.
func (service *MigrationService) Status(ctx context.Context) (MigrationStatus, error) {
	version, err := service.engine.GetCurrentVersion(ctx)
	if err != nil {
		return MigrationStatus{}, fmt.Errorf("read PostgreSQL migration status: %w", err)
	}
	return MigrationStatus{CurrentVersion: version}, nil
}

// PostgresMigrator adapts tern and owns its PostgreSQL connection.
type PostgresMigrator struct {
	connection *pgx.Conn
	migrator   *migrate.Migrator
}

// OpenPostgresMigrator connects, initializes tern's ledger, and loads embedded migrations.
func OpenPostgresMigrator(ctx context.Context, config PostgresConfig) (*PostgresMigrator, error) {
	connectionConfig, err := newPGXConfig(config)
	if err != nil {
		return nil, err
	}
	connection, err := pgx.ConnectConfig(ctx, connectionConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL for migrations: %w", err)
	}

	ternMigrator, err := migrate.NewMigrator(ctx, connection, migrationVersionTable)
	if err != nil {
		_ = connection.Close(ctx)
		return nil, fmt.Errorf("initialize PostgreSQL migration ledger: %w", err)
	}
	if err := ternMigrator.LoadMigrations(migrations.Files); err != nil {
		_ = connection.Close(ctx)
		return nil, fmt.Errorf("load embedded PostgreSQL migrations: %w", err)
	}

	return &PostgresMigrator{connection: connection, migrator: ternMigrator}, nil
}

// Migrate applies every pending migration.
func (migrator *PostgresMigrator) Migrate(ctx context.Context) error {
	return migrator.migrator.Migrate(ctx)
}

// GetCurrentVersion returns tern's current schema version.
func (migrator *PostgresMigrator) GetCurrentVersion(ctx context.Context) (int32, error) {
	return migrator.migrator.GetCurrentVersion(ctx)
}

// Close releases the migration connection.
func (migrator *PostgresMigrator) Close(ctx context.Context) error {
	return migrator.connection.Close(ctx)
}

func newPGXConfig(config PostgresConfig) (*pgx.ConnConfig, error) {
	connectionConfig, err := pgx.ParseConfig("sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL driver configuration: %w", err)
	}
	connectionConfig.Host = config.Host
	connectionConfig.Port = config.Port
	connectionConfig.Database = config.Database
	connectionConfig.User = config.User
	connectionConfig.Password = config.Password
	return connectionConfig, nil
}

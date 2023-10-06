package postgresql

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strings"
)

const (
	otelName = "storage/datastore/relational/postgresql"
)

type PostgresSQLDataStore struct {
	ctx    context.Context
	logger *zap.Logger
	pool   *pgxpool.Pool
}

// NewPostgresSQLDatastore returns a new postgresSQL connection
func NewPostgresSQLDatastore(logger *zap.Logger, dbDsn, serviceName string) *PostgresSQLDataStore {
	ctx := context.Background()
	// establish the connection
	dsn := fmt.Sprintf("%s/%s", strings.ReplaceAll(dbDsn, "postgresql", "postgres"), serviceName)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("pgxpool.Connect failed")
	}
	//defer pool.Close()
	logger.Info("AWS RDS PostgresSQL Database connection established")

	// Check if the DB is connected
	if err = pool.Ping(ctx); err != nil {
		logger.Error("db connection has dropped")
	}
	if err != nil {

	}
	return &PostgresSQLDataStore{
		ctx:    ctx,
		pool:   pool,
		logger: logger,
	}
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	return span
}

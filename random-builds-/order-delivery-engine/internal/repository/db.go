package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
	_ "github.com/lib/pq"
)

// DB wraps sql.DB with helper methods
type DB struct {
	*sql.DB
	logger *slog.Logger
	obsCfg config.ObservabilityConfig
}

type Tx struct {
	*sql.Tx
	logger *slog.Logger
	obsCfg config.ObservabilityConfig
}

type Row struct {
	row    *sql.Row
	ctx    context.Context
	logger *slog.Logger
	obsCfg config.ObservabilityConfig
	query  string
	args   int
	start  time.Time
}

// Connect opens a PostgreSQL connection using the provided config
func Connect(cfg config.DatabaseConfig, logger *slog.Logger, obsCfg config.ObservabilityConfig) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	logger.Info("connected to PostgreSQL", slog.String("database", cfg.DBName), slog.String("host", cfg.Host))
	return &DB{DB: db, logger: logger, obsCfg: obsCfg}, nil
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, logger: db.logger, obsCfg: db.obsCfg}, nil
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := db.DB.ExecContext(ctx, query, args...)
	logQuery(ctx, db.logger, db.obsCfg, query, len(args), time.Since(start), err, res)
	return res, err
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryContext(ctx, query, args...)
	logQuery(ctx, db.logger, db.obsCfg, query, len(args), time.Since(start), err, nil)
	return rows, err
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row {
	start := time.Now()
	row := db.DB.QueryRowContext(ctx, query, args...)
	return &Row{row: row, ctx: ctx, logger: db.logger, obsCfg: db.obsCfg, query: query, args: len(args), start: start}
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := tx.Tx.ExecContext(ctx, query, args...)
	logQuery(ctx, tx.logger, tx.obsCfg, query, len(args), time.Since(start), err, res)
	return res, err
}

func (row *Row) Scan(dest ...interface{}) error {
	err := row.row.Scan(dest...)
	logQuery(row.ctx, row.logger, row.obsCfg, row.query, row.args, time.Since(row.start), err, nil)
	return err
}

func logQuery(ctx context.Context, logger *slog.Logger, cfg config.ObservabilityConfig, query string, argCount int, duration time.Duration, err error, res sql.Result) {
	if logger == nil {
		logger = slog.Default()
	}
	slow := cfg.SlowQueryThreshold > 0 && duration >= cfg.SlowQueryThreshold
	if err == sql.ErrNoRows {
		err = nil
	}
	if !cfg.LogAllQueries && !slow && err == nil {
		return
	}
	attrs := []any{
		slog.String("event", observability.EventDBQuery),
		slog.String("request_id", observability.RequestID(ctx)),
		slog.String("db_operation", observability.SQLOperation(query)),
		slog.Int("arg_count", argCount),
		slog.Float64("duration_ms", observability.DurationMillis(duration)),
		slog.Bool("slow", slow),
		slog.String("sql", observability.NormalizeSQL(query, cfg.MaxSQLLength)),
	}
	if res != nil {
		if rows, rowsErr := res.RowsAffected(); rowsErr == nil {
			attrs = append(attrs, slog.Int64("rows_affected", rows))
		}
	}
	ctxLogger := observability.LoggerFromContext(ctx, logger)
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		ctxLogger.ErrorContext(ctx, "db query failed", attrs...)
		return
	}
	if slow {
		ctxLogger.WarnContext(ctx, "slow db query", attrs...)
		return
	}
	ctxLogger.DebugContext(ctx, "db query", attrs...)
}

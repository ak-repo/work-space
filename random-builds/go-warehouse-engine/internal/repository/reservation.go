package repository

import (
	"context"
	"time"

	"warehouse-engine/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReservationRepo struct {
	pool *pgxpool.Pool
}

func NewReservationRepo(pool *pgxpool.Pool) *ReservationRepo {
	return &ReservationRepo{pool: pool}
}

func (r *ReservationRepo) Create(ctx context.Context, res domain.Reservation) error {
	const q = `
		INSERT INTO reservations (id, product_id, order_id, quantity, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, q,
		res.ID, res.ProductID, res.OrderID,
		res.Quantity, res.ExpiresAt, string(res.Status))
	return err
}

func (r *ReservationRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Reservation, error) {
	const q = `
		SELECT id, product_id, order_id, quantity, expires_at, status
		FROM   reservations
		WHERE  id = $1`

	var res domain.Reservation
	var status string
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&res.ID, &res.ProductID, &res.OrderID,
			&res.Quantity, &res.ExpiresAt, &status)
	res.Status = domain.Status(status)
	return res, err
}

func (r *ReservationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status) error {
	const q = `UPDATE reservations SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, q, string(status), id)
	return err
}

// ListAll returns all reservations in the database.
func (r *ReservationRepo) ListAll(ctx context.Context) ([]domain.Reservation, error) {
	const q = `
		SELECT id, product_id, order_id, quantity, expires_at, status
		FROM   reservations
		ORDER BY expires_at DESC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		var status string
		if err := rows.Scan(&res.ID, &res.ProductID, &res.OrderID,
			&res.Quantity, &res.ExpiresAt, &status); err != nil {
			return nil, err
		}
		res.Status = domain.Status(status)
		reservations = append(reservations, res)
	}
	return reservations, rows.Err()
}

// ListExpiredPending returns all pending reservations whose TTL has passed.
// Used by the background TTL watcher goroutine.
func (r *ReservationRepo) ListExpiredPending(ctx context.Context, now time.Time) ([]domain.Reservation, error) {
	const q = `
		SELECT id, product_id, order_id, quantity, expires_at, status
		FROM   reservations
		WHERE  status     = 'pending'
		  AND  expires_at <= $1`

	rows, err := r.pool.Query(ctx, q, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		var status string
		if err := rows.Scan(&res.ID, &res.ProductID, &res.OrderID,
			&res.Quantity, &res.ExpiresAt, &status); err != nil {
			return nil, err
		}
		res.Status = domain.Status(status)
		out = append(out, res)
	}
	return out, rows.Err()
}

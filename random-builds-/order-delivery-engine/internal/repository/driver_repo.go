package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ak-repo/order-delivery-engine/internal/models"
)

type DriverRepository struct {
	db *DB
}

func NewDriverRepository(db *DB) *DriverRepository {
	return &DriverRepository{db: db}
}

func (r *DriverRepository) Create(ctx context.Context, d *models.Driver) error {
	query := `
		INSERT INTO drivers (id, name, phone, status, current_lat, current_lng, capacity, active_orders, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		d.ID, d.Name, d.Phone, d.Status,
		d.CurrentLat, d.CurrentLng,
		d.Capacity, d.ActiveOrders,
	)
	return err
}

func (r *DriverRepository) GetByID(ctx context.Context, id string) (*models.Driver, error) {
	query := `
		SELECT id, name, phone, status, current_lat, current_lng, capacity, active_orders, created_at, updated_at
		FROM drivers WHERE id = $1
	`
	d := &models.Driver{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.Name, &d.Phone, &d.Status,
		&d.CurrentLat, &d.CurrentLng,
		&d.Capacity, &d.ActiveOrders,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("driver not found: %s", id)
	}
	return d, err
}

func (r *DriverRepository) GetAvailable(ctx context.Context) ([]models.Driver, error) {
	query := `
		SELECT id, name, phone, status, current_lat, current_lng, capacity, active_orders, created_at, updated_at
		FROM drivers
		WHERE status = 'available' AND active_orders < capacity
		ORDER BY active_orders ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []models.Driver
	for rows.Next() {
		var d models.Driver
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Phone, &d.Status,
			&d.CurrentLat, &d.CurrentLng,
			&d.Capacity, &d.ActiveOrders,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

func (r *DriverRepository) GetAll(ctx context.Context) ([]models.Driver, error) {
	query := `
		SELECT id, name, phone, status, current_lat, current_lng, capacity, active_orders, created_at, updated_at
		FROM drivers ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []models.Driver
	for rows.Next() {
		var d models.Driver
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Phone, &d.Status,
			&d.CurrentLat, &d.CurrentLng,
			&d.Capacity, &d.ActiveOrders,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		drivers = append(drivers, d)
	}
	return drivers, rows.Err()
}

func (r *DriverRepository) UpdateLocation(ctx context.Context, id string, lat, lng float64) error {
	query := `UPDATE drivers SET current_lat=$1, current_lng=$2, updated_at=NOW() WHERE id=$3`
	res, err := r.db.ExecContext(ctx, query, lat, lng, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("driver not found: %s", id)
	}
	return nil
}

func (r *DriverRepository) UpdateStatus(ctx context.Context, id string, status models.DriverStatus) error {
	query := `UPDATE drivers SET status=$1, updated_at=NOW() WHERE id=$2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// IncrementActiveOrders increments or decrements active_orders atomically
func (r *DriverRepository) ChangeActiveOrders(ctx context.Context, id string, delta int) error {
	return changeActiveOrders(ctx, r.db, id, delta)
}

func (r *DriverRepository) ChangeActiveOrdersTx(ctx context.Context, tx *Tx, id string, delta int) error {
	return changeActiveOrders(ctx, tx, id, delta)
}

func changeActiveOrders(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}, id string, delta int) error {
	query := `
		UPDATE drivers
		SET active_orders = GREATEST(0, active_orders + $1),
		    status = CASE
		        WHEN active_orders + $1 >= capacity THEN 'busy'
		        ELSE 'available'
		    END,
		    updated_at = NOW()
		WHERE id = $2
		  AND active_orders + $1 >= 0
		  AND ($1 <= 0 OR active_orders < capacity)
	`
	res, err := exec.ExecContext(ctx, query, delta, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("driver %s cannot change active_orders by %d", id, delta)
	}
	return nil
}

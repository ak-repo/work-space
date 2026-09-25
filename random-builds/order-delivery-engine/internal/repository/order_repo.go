package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ak-repo/order-delivery-engine/internal/models"
)

type OrderRepository struct {
	db *DB
}

func NewOrderRepository(db *DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	return r.db.BeginTx(ctx, opts)
}

func (r *OrderRepository) Create(ctx context.Context, o *models.Order) error {
	query := `
		INSERT INTO orders (
			id, customer_name, customer_phone,
			pickup_lat, pickup_lng, pickup_address,
			delivery_lat, delivery_lng, delivery_address,
			status, priority, notes, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		o.ID, o.CustomerName, o.CustomerPhone,
		o.PickupLat, o.PickupLng, o.PickupAddress,
		o.DeliveryLat, o.DeliveryLng, o.DeliveryAddress,
		o.Status, o.Priority, o.Notes,
	)
	return err
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	query := `
		SELECT id, customer_name, customer_phone,
		       pickup_lat, pickup_lng, pickup_address,
		       delivery_lat, delivery_lng, delivery_address,
		       status, driver_id, priority, notes,
		       created_at, updated_at, assigned_at, delivered_at
		FROM orders WHERE id = $1
	`
	o := &models.Order{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&o.ID, &o.CustomerName, &o.CustomerPhone,
		&o.PickupLat, &o.PickupLng, &o.PickupAddress,
		&o.DeliveryLat, &o.DeliveryLng, &o.DeliveryAddress,
		&o.Status, &o.DriverID, &o.Priority, &o.Notes,
		&o.CreatedAt, &o.UpdatedAt, &o.AssignedAt, &o.DeliveredAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %s", id)
	}
	return o, err
}

func (r *OrderRepository) GetAll(ctx context.Context) ([]models.Order, error) {
	query := `
		SELECT id, customer_name, customer_phone,
		       pickup_lat, pickup_lng, pickup_address,
		       delivery_lat, delivery_lng, delivery_address,
		       status, driver_id, priority, notes,
		       created_at, updated_at, assigned_at, delivered_at
		FROM orders ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerName, &o.CustomerPhone,
			&o.PickupLat, &o.PickupLng, &o.PickupAddress,
			&o.DeliveryLat, &o.DeliveryLng, &o.DeliveryAddress,
			&o.Status, &o.DriverID, &o.Priority, &o.Notes,
			&o.CreatedAt, &o.UpdatedAt, &o.AssignedAt, &o.DeliveredAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) GetByStatus(ctx context.Context, status models.OrderStatus) ([]models.Order, error) {
	query := `
		SELECT id, customer_name, customer_phone,
		       pickup_lat, pickup_lng, pickup_address,
		       delivery_lat, delivery_lng, delivery_address,
		       status, driver_id, priority, notes,
		       created_at, updated_at, assigned_at, delivered_at
		FROM orders WHERE status = $1 ORDER BY priority DESC, created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerName, &o.CustomerPhone,
			&o.PickupLat, &o.PickupLng, &o.PickupAddress,
			&o.DeliveryLat, &o.DeliveryLng, &o.DeliveryAddress,
			&o.Status, &o.DriverID, &o.Priority, &o.Notes,
			&o.CreatedAt, &o.UpdatedAt, &o.AssignedAt, &o.DeliveredAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// AssignDriver atomically assigns driver to order
func (r *OrderRepository) AssignDriver(ctx context.Context, orderID, driverID string) error {
	return assignDriver(ctx, r.db, orderID, driverID)
}

func (r *OrderRepository) AssignDriverTx(ctx context.Context, tx *Tx, orderID, driverID string) error {
	return assignDriver(ctx, tx, orderID, driverID)
}

func assignDriver(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}, orderID, driverID string) error {
	query := `
		UPDATE orders
		SET driver_id=$1, status='assigned', assigned_at=NOW(), updated_at=NOW()
		WHERE id=$2 AND status='pending'
	`
	res, err := exec.ExecContext(ctx, query, driverID, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("order %s is not in pending state or not found", orderID)
	}
	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	return updateStatus(ctx, r.db, id, status)
}

func (r *OrderRepository) UpdateStatusTx(ctx context.Context, tx *Tx, id string, status models.OrderStatus) error {
	return updateStatus(ctx, tx, id, status)
}

func updateStatus(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}, id string, status models.OrderStatus) error {
	var query string
	switch status {
	case models.OrderDelivered:
		query = `UPDATE orders SET status=$1, delivered_at=NOW(), updated_at=NOW() WHERE id=$2`
	default:
		query = `UPDATE orders SET status=$1, updated_at=NOW() WHERE id=$2`
	}
	res, err := exec.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("order not found: %s", id)
	}
	return nil
}

// GetByDriver returns all active orders for a driver
func (r *OrderRepository) GetByDriver(ctx context.Context, driverID string) ([]models.Order, error) {
	query := `
		SELECT id, customer_name, customer_phone,
		       pickup_lat, pickup_lng, pickup_address,
		       delivery_lat, delivery_lng, delivery_address,
		       status, driver_id, priority, notes,
		       created_at, updated_at, assigned_at, delivered_at
		FROM orders
		WHERE driver_id=$1 AND status NOT IN ('delivered','cancelled')
		ORDER BY priority DESC, assigned_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerName, &o.CustomerPhone,
			&o.PickupLat, &o.PickupLng, &o.PickupAddress,
			&o.DeliveryLat, &o.DeliveryLng, &o.DeliveryAddress,
			&o.Status, &o.DriverID, &o.Priority, &o.Notes,
			&o.CreatedAt, &o.UpdatedAt, &o.AssignedAt, &o.DeliveredAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

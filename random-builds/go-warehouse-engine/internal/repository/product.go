package repository

import (
	"context"
	"errors"

	"warehouse-engine/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrVersionConflict is returned when an optimistic lock UPDATE matches 0 rows,
// meaning another transaction already mutated the row since we read it.
var ErrVersionConflict = errors.New("optimistic lock: version conflict")

type ProductRepo struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{pool: pool}
}

// GetByID fetches a single product by UUID.
func (r *ProductRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	const q = `
		SELECT id, name, sku, total_stock, reserved, version, reorder_at
		FROM   products
		WHERE  id = $1`

	var p domain.Product
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&p.ID, &p.Name, &p.SKU, &p.TotalStock, &p.Reserved, &p.Version, &p.ReorderAt)
	return p, err
}

// ListAll returns all products in the database.
func (r *ProductRepo) ListAll(ctx context.Context) ([]domain.Product, error) {
	const q = `
		SELECT id, name, sku, total_stock, reserved, version, reorder_at
		FROM   products
		ORDER BY name`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.TotalStock, &p.Reserved, &p.Version, &p.ReorderAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// ReserveStock increments reserved by qty using an optimistic lock on version.
//
// The WHERE clause checks:
//   - version = $3  (no one else changed the row since we read it)
//   - (total_stock - reserved) >= $1  (enough stock actually available)
//
// If 0 rows are updated the caller should retry after re-reading the product.
func (r *ProductRepo) ReserveStock(ctx context.Context, p domain.Product, qty int) error {
	const q = `
		UPDATE products
		SET    reserved = reserved + $1,
		       version  = version  + 1
		WHERE  id       = $2
		  AND  version  = $3
		  AND  (total_stock - reserved) >= $1`

	tag, err := r.pool.Exec(ctx, q, qty, p.ID, p.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrVersionConflict
	}
	return nil
}

// ReleaseStock decrements reserved (called on cancel or TTL expiry).
func (r *ProductRepo) ReleaseStock(ctx context.Context, productID uuid.UUID, qty int) error {
	const q = `
		UPDATE products
		SET reserved = GREATEST(reserved - $1, 0),
		    version  = version + 1
		WHERE id = $2`

	_, err := r.pool.Exec(ctx, q, qty, productID)
	return err
}

// ConfirmStock permanently deducts from total_stock and releases the reserve hold.
// Fails atomically if total_stock is insufficient.
func (r *ProductRepo) ConfirmStock(ctx context.Context, productID uuid.UUID, qty int) error {
	const q = `
		UPDATE products
		SET total_stock = total_stock - $1,
		    reserved    = GREATEST(reserved - $1, 0),
		    version     = version + 1
		WHERE id          = $2
		  AND total_stock >= $1`

	tag, err := r.pool.Exec(ctx, q, qty, productID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("confirm failed: insufficient total_stock")
	}
	return nil
}

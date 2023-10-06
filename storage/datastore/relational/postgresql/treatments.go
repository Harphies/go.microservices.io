package postgresql

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"streamex/treatment/internal"
)

type TreatmentDBResp struct {
	ID          uuid.UUID
	Name        string
	Description string
	Image       string
	Price       string
	Type        string
	Cure        string
	Certified   bool
	CertifiedBy []string
	CreatedAt   pgtype.Timestamp
	UpdatedAt   pgtype.Timestamp
	DeletedAt   pgtype.Timestamp
}

type TreatmentsDBRes []TreatmentDBResp

// CreateItem - Persist record to the database
func (d *PostgresSQLDataStore) CreateItem(ctx context.Context, item internal.Treatment) string {

	query := `
		INSERT INTO treatments (name, description, image, price, type, cure, certified, certified_by, created_at, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_DATE, CURRENT_DATE)
		RETURNING id
		`

	// execute the computed query and handle any error
	row := d.pool.QueryRow(ctx, query,
		item.Name,
		item.Description,
		item.Image,
		item.Price,
		item.Type,
		item.Cure,
		item.Certified,
		item.CertifiedBy,
	)
	var id uuid.UUID
	_ = row.Scan(&id)
	return id.String()
}

// GetItem Retrieve an Item from the datastore
func (d *PostgresSQLDataStore) GetItem(ctx context.Context, itemId string) TreatmentDBResp {
	query := `SELECT * FROM treatments WHERE id = $1 LIMIT 1`

	parsedId, err := uuid.Parse(itemId)

	// execute the computed query and handle any error
	row := d.pool.QueryRow(ctx, query, parsedId)
	var item TreatmentDBResp
	err = row.Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Image,
		&item.Price,
		&item.Type,
		&item.Cure,
		&item.Certified,
		&item.CertifiedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt)
	if err != nil {
		d.logger.Error(fmt.Sprintf("failed to retrieve an item with id %v", itemId), zap.Error(err))
	}
	return item
}

// GetItems Retrieve all Item from the datastore
func (d *PostgresSQLDataStore) GetItems(ctx context.Context) TreatmentsDBRes {
	query := `SELECT * FROM treatments`

	// execute the computed query and handle any error
	rows, _ := d.pool.Query(ctx, query)
	items := TreatmentsDBRes{}
	for rows.Next() {
		item := TreatmentDBResp{}
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Image,
			&item.Price,
			&item.Type,
			&item.Cure,
			&item.Certified,
			&item.CertifiedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.DeletedAt)
		if err != nil {
			d.logger.Error(fmt.Sprintf("failed to retrieve all items"), zap.Error(err))
		}
		items = append(items, item)
	}
	return items
}

// UpdateItem - Update an Item in the datastore
func (d *PostgresSQLDataStore) UpdateItem(ctx context.Context, itemId string, item internal.Treatment) string {
	query := `
		UPDATE treatments SET 
		    name = $1,
		    description = $2,
		    image = $3,
		    price = $4,
		    type = $5,
		    cure = $6,
		    certified = $7,
		    certified_by = $8,
		    updated_at = now()
		WHERE id = $9
		RETURNING id AS res
		`

	parsedId, _ := uuid.Parse(itemId)

	// execute the computed query and handle any error
	row := d.pool.QueryRow(ctx, query,
		item.Name,
		item.Description,
		item.Image,
		item.Price,
		item.Type,
		item.Cure,
		item.Certified,
		item.CertifiedBy,
		parsedId,
	)
	var res uuid.UUID
	_ = row.Scan(&res)
	return res.String()
}

// DeleteItem - Remove/Purge an Item from the datastore

func (d *PostgresSQLDataStore) DeleteItem(ctx context.Context, itemId string) string {
	query := `DELETE FROM treatments 
       			WHERE id = $1
				RETURNING id AS res`

	parsedId, _ := uuid.Parse(itemId)

	// execute the computed query and handle any error
	row := d.pool.QueryRow(ctx, query, parsedId)
	var res uuid.UUID
	_ = row.Scan(&res)
	return res.String()
}

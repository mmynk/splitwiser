// Package storage provides abstractions for persistent data storage.
package storage

import (
	"context"

	"github.com/mmynk/splitwiser/internal/models"
)

// Store defines the interface for bill storage operations.
// This abstraction allows swapping storage backends (SQLite, PostgreSQL, etc.)
// without changing the service layer.
type Store interface {
	// CreateBill persists a new bill and returns the assigned ID.
	// The bill.ID field will be populated by the store.
	CreateBill(ctx context.Context, bill *models.Bill) error

	// GetBill retrieves a bill by its ID.
	// Returns nil and an error if the bill is not found.
	GetBill(ctx context.Context, billID string) (*models.Bill, error)

	// UpdateBill updates an existing bill.
	// Returns an error if the bill is not found.
	UpdateBill(ctx context.Context, bill *models.Bill) error

	// Close releases any resources held by the store.
	Close() error
}

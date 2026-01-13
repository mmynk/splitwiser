// Package storage provides abstractions for persistent data storage.
package storage

import (
	"context"

	"github.com/mmynk/splitwiser/internal/models"
)

// Store defines the interface for bill and group storage operations.
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

	// ListBillsByGroup retrieves all bills associated with a group.
	// Returns an empty slice if the group has no bills.
	ListBillsByGroup(ctx context.Context, groupID string) ([]*models.Bill, error)

	// CreateGroup persists a new group.
	// The group.ID field will be populated by the store.
	CreateGroup(ctx context.Context, group *models.Group) error

	// GetGroup retrieves a group by its ID.
	// Returns nil and an error if the group is not found.
	GetGroup(ctx context.Context, groupID string) (*models.Group, error)

	// ListGroups retrieves all groups.
	ListGroups(ctx context.Context) ([]*models.Group, error)

	// UpdateGroup updates an existing group.
	// Returns an error if the group is not found.
	UpdateGroup(ctx context.Context, group *models.Group) error

	// DeleteGroup removes a group by its ID.
	// Bills associated with the group will have their group_id set to NULL.
	DeleteGroup(ctx context.Context, groupID string) error

	// Close releases any resources held by the store.
	Close() error
}

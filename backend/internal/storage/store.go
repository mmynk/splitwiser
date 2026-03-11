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

	// DeleteBill removes a bill by its ID.
	// Returns an error if the bill is not found.
	DeleteBill(ctx context.Context, billID string) error

	// ListBillsByGroup retrieves all bills associated with a group.
	// Returns an empty slice if the group has no bills.
	ListBillsByGroup(ctx context.Context, groupID string) ([]*models.Bill, error)

	// ListBillsByUser retrieves all bills where the given user is the creator or a participant.
	// Returns an empty slice if the user has no bills.
	ListBillsByUser(ctx context.Context, userID string) ([]*models.Bill, error)

	// CreateGroup persists a new group.
	// The group.ID field will be populated by the store.
	CreateGroup(ctx context.Context, group *models.Group) error

	// GetGroup retrieves a group by its ID.
	// Returns nil and an error if the group is not found.
	GetGroup(ctx context.Context, groupID string) (*models.Group, error)

	// ListGroupsByUser retrieves all groups the given user belongs to.
	ListGroupsByUser(ctx context.Context, userID string) ([]*models.Group, error)

	// UpdateGroup updates an existing group.
	// Returns an error if the group is not found.
	UpdateGroup(ctx context.Context, group *models.Group) error

	// AddGroupMembers adds members to a group idempotently.
	// Members that already exist in the group are silently ignored.
	AddGroupMembers(ctx context.Context, groupID string, memberIDs []string) error

	// DeleteGroup removes a group by its ID.
	// Bills associated with the group will have their group_id set to NULL.
	DeleteGroup(ctx context.Context, groupID string) error

	// CreateSettlement persists a new settlement.
	// The settlement.ID field will be populated by the store.
	CreateSettlement(ctx context.Context, settlement *models.Settlement) error

	// GetSettlement retrieves a settlement by its ID.
	// Returns nil and an error if the settlement is not found.
	GetSettlement(ctx context.Context, settlementID string) (*models.Settlement, error)

	// ListSettlementsByGroup retrieves all settlements for a group.
	// Returns an empty slice if the group has no settlements.
	ListSettlementsByGroup(ctx context.Context, groupID string) ([]*models.Settlement, error)

	// DeleteSettlement removes a settlement by its ID.
	// Returns an error if the settlement is not found.
	DeleteSettlement(ctx context.Context, settlementID string) error

	// GetUsersByIDs retrieves multiple users by their IDs. Missing IDs are omitted.
	GetUsersByIDs(ctx context.Context, ids []string) (map[string]*models.User, error)

	// SearchUsers finds a registered user by exact email address, excluding the caller.
	// Returns nil, nil when no matching user is found.
	SearchUsers(ctx context.Context, email string, callerID string) (*models.User, error)

	// AddGroupMembersWithIDs adds members (with optional user IDs) to a group idempotently.
	AddGroupMembersWithIDs(ctx context.Context, groupID string, members []models.GroupMember) error

	// SendFriendRequest persists a new friendship request.
	// Returns an error if a request already exists in either direction.
	SendFriendRequest(ctx context.Context, friendship *models.Friendship) error

	// GetFriendship retrieves a friendship by ID.
	GetFriendship(ctx context.Context, id string) (*models.Friendship, error)

	// UpdateFriendshipStatus updates the status of a friendship.
	UpdateFriendshipStatus(ctx context.Context, id string, status models.FriendshipStatus) error

	// ListFriendships lists friendships for a user.
	// If incoming is true, returns requests where userID is the addressee.
	// If incoming is false, returns requests where userID is the requester.
	ListFriendships(ctx context.Context, userID string, incoming bool, status models.FriendshipStatus) ([]*models.Friendship, error)

	// DeleteFriendship removes a friendship by ID.
	DeleteFriendship(ctx context.Context, id string) error

	// AreFriends returns true if the two users have an accepted friendship in either direction.
	AreFriends(ctx context.Context, userIDA, userIDB string) (bool, error)

	// GetFriends returns all accepted friends of a user (the other party in each friendship).
	GetFriends(ctx context.Context, userID string) ([]*models.User, error)

	// GetFriendshipBetween retrieves the friendship between two users in either direction.
	// Returns a not-found error if no row exists.
	GetFriendshipBetween(ctx context.Context, userIDA, userIDB string) (*models.Friendship, error)

	// SearchFriends finds accepted friends matching a partial display name query.
	SearchFriends(ctx context.Context, callerID string, query string) ([]*models.User, error)

	// Close releases any resources held by the store.
	Close() error
}

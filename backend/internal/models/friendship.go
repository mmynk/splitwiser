package models

// FriendshipStatus represents the state of a friend request.
type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
	FriendshipDeclined FriendshipStatus = "declined"
)

// Friendship represents a bidirectional friend relationship between two users.
type Friendship struct {
	ID          string
	RequesterID string
	AddresseeID string
	Status      FriendshipStatus
	CreatedAt   int64
	UpdatedAt   int64
}

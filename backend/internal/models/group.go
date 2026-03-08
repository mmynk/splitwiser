package models

// GroupMember represents a member of a group, linking display name to an optional user account.
type GroupMember struct {
	DisplayName string
	UserID      string // empty for guests
}

// Group represents a reusable participant list.
type Group struct {
	ID        string
	Name      string
	Members   []GroupMember
	CreatedAt int64
}

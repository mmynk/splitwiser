package models

// Group represents a reusable participant list.
// Groups can own bills, enabling group bill history.
//
// Future: When authentication is added, Members will reference User IDs
// instead of names. The model is designed to be extensible for this transition.
type Group struct {
	// ID is the unique identifier for the group (UUID format).
	ID string

	// Name is the display name of the group (e.g., "Roommates", "Work Lunch").
	Name string

	// Members is the list of participant names in this group.
	// Future: will reference User IDs when auth is added.
	Members []string

	// CreatedAt is the Unix timestamp when the group was created.
	CreatedAt int64
}

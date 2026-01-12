package models

// Group represents a recurring group of people who frequently split bills together.
//
// NOTE: This model is for FUTURE use when authentication is added.
// The MVP does not use groups - participants are entered manually for each bill.
//
// Future features:
//   - Recurring groups (e.g., "Roommates", "Hiking Crew", "Office Lunch Group")
//   - Default participants when creating a bill within a group
//   - Group bill history
//   - Group settings (default split method, currency, etc.)
type Group struct {
	// ID is the unique identifier for the group (UUID format).
	ID string

	// Name is the display name of the group (e.g., "Roommates", "Pizza Fridays").
	Name string

	// MemberIDs is the list of user IDs who belong to this group.
	// Using IDs instead of User pointers to avoid circular references.
	MemberIDs []string

	// CreatedBy is the user ID of the person who created the group.
	CreatedBy string

	// CreatedAt is the Unix timestamp when the group was created.
	CreatedAt int64

	// Future fields to consider:
	// - Description string (group purpose)
	// - Currency string (default currency for bills)
	// - DefaultSplitMethod string ("equal", "itemized", etc.)
	// - Archived bool (soft delete)
}

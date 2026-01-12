package models

// Bill represents a bill with items to be split among participants.
// It stores the complete bill information including items, totals, and calculated splits.
type Bill struct {
	// ID is the unique identifier for the bill (UUID format).
	ID string

	// Title is the human-readable name for the bill.
	// For MVP, auto-generated from participants or date.
	// Future: could be user-provided or fun generated names.
	Title string

	// Items are the individual line items on the bill.
	// Each item can be assigned to one or more participants.
	Items []Item

	// Total is the final bill amount including tax, tips, and fees.
	Total float64

	// Subtotal is the pre-tax amount (sum of all items before tax).
	Subtotal float64

	// Participants is the list of people splitting the bill.
	// For MVP, these are just names (strings).
	// Future: will reference User IDs when auth is added.
	Participants []string

	// CreatedAt is the Unix timestamp when the bill was created.
	CreatedAt int64
}

// Item represents a single line item on a bill.
// Items can be shared among multiple participants.
type Item struct {
	// ID is the unique identifier for the item (UUID format).
	ID string

	// Description is the name or description of the item (e.g., "Pizza", "Beer").
	Description string

	// Amount is the pre-tax price of this item.
	Amount float64

	// Participants is the list of participant names who should split this item.
	// If multiple people are assigned, the item is split equally among them.
	// For MVP, these are participant names (strings).
	// Future: will reference User IDs when auth is added.
	Participants []string
}

// PersonItem represents an item's share for one person.
type PersonItem struct {
	Description string
	Amount      float64 // This person's share of the item
}

// PersonSplit represents one person's calculated share of a bill.
// This is the output of the split calculation algorithm.
type PersonSplit struct {
	// Participant is the name of the person.
	// For MVP, this is just a string (no user accounts yet).
	// Future: will reference User ID when auth is added.
	Participant string

	// Subtotal is the sum of this person's assigned item amounts (pre-tax).
	Subtotal float64

	// Tax is this person's proportional share of taxes/fees.
	// Calculated as: subtotal × (total_tax / bill_subtotal)
	Tax float64

	// Total is the final amount this person owes (subtotal + tax).
	Total float64

	// Items are the specific items assigned to this person with their share amounts.
	Items []PersonItem
}

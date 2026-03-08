package models

// BillParticipant represents a participant on a bill, linking display name to an optional user account.
type BillParticipant struct {
	DisplayName string
	UserID      string // empty for guests
}

// Bill represents a bill with items to be split among participants.
type Bill struct {
	ID           string
	Title        string
	Items        []Item
	Total        float64
	Subtotal     float64
	Participants []BillParticipant
	CreatedAt    int64
	GroupID      string
	PayerID      string
}

// Item represents a single line item on a bill.
// Participants holds display names (used by the calculator).
type Item struct {
	ID           string
	Description  string
	Amount       float64
	Participants []string // display names
}

// PersonItem represents an item's share for one person.
type PersonItem struct {
	Description string
	Amount      float64
}

// PersonSplit represents one person's calculated share of a bill.
type PersonSplit struct {
	Participant string
	Subtotal    float64
	Tax         float64
	Total       float64
	Items       []PersonItem
}

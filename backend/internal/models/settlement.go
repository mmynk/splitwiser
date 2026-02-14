package models

// Settlement represents a payment between group members to clear debts.
type Settlement struct {
	// ID is the unique identifier for the settlement (UUID format).
	ID string

	// GroupID is the group this settlement belongs to.
	GroupID string

	// FromUserID is the user who paid (debtor settling up).
	FromUserID string

	// ToUserID is the user who received payment (creditor being paid).
	ToUserID string

	// Amount is the payment amount.
	Amount float64

	// CreatedAt is the Unix timestamp when the settlement was recorded.
	CreatedAt int64

	// CreatedBy is the user ID who recorded this settlement.
	CreatedBy string

	// Note is an optional description for the settlement.
	Note string
}

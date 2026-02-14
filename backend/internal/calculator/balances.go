package calculator

import "fmt"

// BillForBalance represents a bill with the minimal information needed for balance calculations.
type BillForBalance struct {
	Total        float64
	Subtotal     float64
	PayerID      string
	Items        []Item
	Participants []string
}

// MemberBalance represents the balance information for one group member.
type MemberBalance struct {
	MemberName string
	NetBalance float64 // Positive = owed money, Negative = owes money
	TotalPaid  float64 // Total amount paid across all bills
	TotalOwed  float64 // Total amount this person owes
}

// DebtEdge represents a debt from one person to another.
type DebtEdge struct {
	From   string  // Person who owes
	To     string  // Person who is owed
	Amount float64
}

// SettlementForBalance represents a settlement with the minimal information needed for balance calculations.
type SettlementForBalance struct {
	FromUserID string  // Who paid (debtor settling up)
	ToUserID   string  // Who received (creditor being paid)
	Amount     float64
}

// CalculateGroupBalances computes balances across multiple bills and settlements.
// It aggregates who paid what and who owes what, returning both individual
// member balances and a detailed debt matrix.
//
// Algorithm:
// - For each bill: payer contributed +total, each participant owes their split
// - For each settlement: payer's balance improves, receiver's balance decreases
// - Aggregate: net_balance = total_paid - total_owed
// - Debt matrix: simplified using greedy matching
func CalculateGroupBalances(bills []BillForBalance, settlements []SettlementForBalance) ([]MemberBalance, []DebtEdge, error) {
	// Track balances per member
	balances := make(map[string]*MemberBalance)

	// Track debts: debts[debtor][creditor] = amount
	debts := make(map[string]map[string]float64)

	for _, bill := range bills {
		// Skip bills without payer (can't calculate balances)
		if bill.PayerID == "" {
			continue
		}

		// Calculate splits for this bill
		splitResult, err := CalculateSplit(bill.Items, bill.Total, bill.Subtotal, bill.Participants)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate split: %w", err)
		}

		// Initialize payer's balance if needed
		if _, exists := balances[bill.PayerID]; !exists {
			balances[bill.PayerID] = &MemberBalance{MemberName: bill.PayerID}
		}

		// Payer paid the full amount
		balances[bill.PayerID].TotalPaid += bill.Total

		// Each participant owes their share
		for participant, personSplit := range splitResult {
			if _, exists := balances[participant]; !exists {
				balances[participant] = &MemberBalance{MemberName: participant}
			}

			balances[participant].TotalOwed += personSplit.Total

			// If not the payer, record debt
			if participant != bill.PayerID {
				if _, exists := debts[participant]; !exists {
					debts[participant] = make(map[string]float64)
				}
				debts[participant][bill.PayerID] += personSplit.Total
			}
		}
	}

	// Apply settlements to balances
	for _, s := range settlements {
		// Initialize from user's balance if needed
		if _, exists := balances[s.FromUserID]; !exists {
			balances[s.FromUserID] = &MemberBalance{MemberName: s.FromUserID}
		}
		// Initialize to user's balance if needed
		if _, exists := balances[s.ToUserID]; !exists {
			balances[s.ToUserID] = &MemberBalance{MemberName: s.ToUserID}
		}
		// Payer's balance improves (they effectively "paid" to settle debt)
		balances[s.FromUserID].TotalPaid += s.Amount
		// Receiver's balance decreases (they received payment)
		balances[s.ToUserID].TotalOwed += s.Amount
	}

	// Compute net balances
	for _, bal := range balances {
		bal.NetBalance = bal.TotalPaid - bal.TotalOwed
	}

	// Convert to slices
	var memberBalances []MemberBalance
	for _, bal := range balances {
		memberBalances = append(memberBalances, *bal)
	}

	// Simplify debts using net balances
	// Create lists of creditors (owed money) and debtors (owe money)
	var creditors []MemberBalance
	var debtors []MemberBalance
	for _, bal := range balances {
		if bal.NetBalance > 0 {
			creditors = append(creditors, *bal)
		} else if bal.NetBalance < 0 {
			debtors = append(debtors, *bal)
		}
	}

	// Match debtors with creditors to minimize transactions
	var debtEdges []DebtEdge
	i, j := 0, 0
	debtorBalance := make(map[string]float64)
	creditorBalance := make(map[string]float64)

	for _, debtor := range debtors {
		debtorBalance[debtor.MemberName] = -debtor.NetBalance // Make positive
	}
	for _, creditor := range creditors {
		creditorBalance[creditor.MemberName] = creditor.NetBalance
	}

	// Greedy algorithm: match largest debts with largest credits
	for i < len(debtors) && j < len(creditors) {
		debtor := debtors[i].MemberName
		creditor := creditors[j].MemberName

		// Amount to settle is minimum of what debtor owes and creditor is owed
		amount := debtorBalance[debtor]
		if creditorBalance[creditor] < amount {
			amount = creditorBalance[creditor]
		}

		if amount > 0.01 { // Avoid floating point noise
			debtEdges = append(debtEdges, DebtEdge{
				From:   debtor,
				To:     creditor,
				Amount: amount,
			})
		}

		// Update balances
		debtorBalance[debtor] -= amount
		creditorBalance[creditor] -= amount

		// Move to next debtor/creditor if fully settled
		if debtorBalance[debtor] < 0.01 {
			i++
		}
		if creditorBalance[creditor] < 0.01 {
			j++
		}
	}

	return memberBalances, debtEdges, nil
}

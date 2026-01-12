package calculator

import (
	"fmt"
)

// PersonItem represents an item's share for one person
type PersonItem struct {
	Description string
	Amount      float64 // This person's share of the item
}

// PersonSplit represents the calculated split for one person
type PersonSplit struct {
	Subtotal float64
	Tax      float64
	Total    float64
	Items    []PersonItem // Items assigned to this person with their share
}

// Item represents a single item on the bill
type Item struct {
	Description  string
	Amount       float64
	Participants []string // was: AssignedTo
}

// CalculateSplit computes how much each person owes including proportional tax
// Based on the algorithm: person_total = person_subtotal × (1 + (total_tax / bill_subtotal))
func CalculateSplit(items []Item, billTotal float64, billSubtotal float64, participants []string) (map[string]*PersonSplit, error) {
	if billSubtotal == 0 {
		return nil, fmt.Errorf("subtotal cannot be zero")
	}
	if len(participants) == 0 {
		return nil, fmt.Errorf("must have at least one participant")
	}

	tax := billTotal - billSubtotal
	splits := make(map[string]*PersonSplit)

	// Initialize splits for all participants
	for _, p := range participants {
		splits[p] = &PersonSplit{
			Subtotal: 0,
			Tax:      0,
			Total:    0,
			Items:    []PersonItem{},
		}
	}

	// If no items, split total equally among all participants
	if len(items) == 0 {
		perPersonTotal := billTotal / float64(len(participants))
		perPersonSubtotal := billSubtotal / float64(len(participants))
		perPersonTax := tax / float64(len(participants))

		for _, split := range splits {
			split.Subtotal = perPersonSubtotal
			split.Tax = perPersonTax
			split.Total = perPersonTotal
		}
		return splits, nil
	}

	// Calculate each person's subtotal based on assigned items
	itemsTotal := 0.0
	for _, item := range items {
		if len(item.Participants) == 0 {
			continue
		}

		itemsTotal += item.Amount

		// Split item among assigned people
		perPersonAmount := item.Amount / float64(len(item.Participants))
		for _, person := range item.Participants {
			if split, exists := splits[person]; exists {
				split.Subtotal += perPersonAmount
				split.Items = append(split.Items, PersonItem{
					Description: item.Description,
					Amount:      perPersonAmount,
				})
			}
		}
	}

	// If items don't account for full subtotal, split remainder equally
	if itemsTotal < billSubtotal {
		remainder := billSubtotal - itemsTotal
		perPersonShare := remainder / float64(len(participants))
		for _, split := range splits {
			split.Subtotal += perPersonShare
			split.Items = append(split.Items, PersonItem{
				Description: "Shared",
				Amount:      perPersonShare,
			})
		}
	}

	// Apply proportional tax and calculate total
	for _, split := range splits {
		split.Tax = split.Subtotal * (tax / billSubtotal)
		split.Total = split.Subtotal + split.Tax
	}

	return splits, nil
}

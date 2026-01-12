package calculator

import (
	"math"
	"testing"
)

func TestCalculateSplit(t *testing.T) {
	tests := []struct {
		name          string
		items         []Item
		billTotal     float64
		billSubtotal  float64
		participants  []string
		wantErr       bool
		validateFunc  func(t *testing.T, splits map[string]*PersonSplit)
	}{
		{
			name: "simple two-person split with tax",
			items: []Item{
				{Description: "Pizza", Amount: 20.0, Participants: []string{"Alice", "Bob"}},
				{Description: "Salad", Amount: 10.0, Participants: []string{"Alice"}},
			},
			billTotal:    33.0,
			billSubtotal: 30.0,
			participants: []string{"Alice", "Bob"},
			wantErr:      false,
			validateFunc: func(t *testing.T, splits map[string]*PersonSplit) {
				// Alice: subtotal = 10 + 10 = 20, tax = 20 * (3/30) = 2, total = 22
				// Bob: subtotal = 10, tax = 10 * (3/30) = 1, total = 11
				alice := splits["Alice"]
				if math.Abs(alice.Subtotal-20.0) > 0.01 {
					t.Errorf("Alice subtotal = %v, want 20.0", alice.Subtotal)
				}
				if math.Abs(alice.Tax-2.0) > 0.01 {
					t.Errorf("Alice tax = %v, want 2.0", alice.Tax)
				}
				if math.Abs(alice.Total-22.0) > 0.01 {
					t.Errorf("Alice total = %v, want 22.0", alice.Total)
				}

				bob := splits["Bob"]
				if math.Abs(bob.Subtotal-10.0) > 0.01 {
					t.Errorf("Bob subtotal = %v, want 10.0", bob.Subtotal)
				}
				if math.Abs(bob.Total-11.0) > 0.01 {
					t.Errorf("Bob total = %v, want 11.0", bob.Total)
				}
			},
		},
		{
			name:         "zero subtotal should error",
			items:        []Item{{Description: "Item", Amount: 10.0, Participants: []string{"Alice"}}},
			billTotal:    10.0,
			billSubtotal: 0.0,
			participants: []string{"Alice"},
			wantErr:      true,
		},
		{
			name:         "no participants should error",
			items:        []Item{{Description: "Item", Amount: 10.0, Participants: []string{"Alice"}}},
			billTotal:    10.0,
			billSubtotal: 10.0,
			participants: []string{},
			wantErr:      true,
		},
		{
			name:         "no items - split equally among participants",
			items:        []Item{},
			billTotal:    33.0,
			billSubtotal: 30.0,
			participants: []string{"Alice", "Bob"},
			wantErr:      false,
			validateFunc: func(t *testing.T, splits map[string]*PersonSplit) {
				// Total bill = 33, split between 2 people = 16.50 each
				// Subtotal = 30, split between 2 = 15 each
				// Tax = 3, split between 2 = 1.50 each
				for _, person := range []string{"Alice", "Bob"} {
					split := splits[person]
					if math.Abs(split.Subtotal-15.0) > 0.01 {
						t.Errorf("%s subtotal = %v, want 15.0", person, split.Subtotal)
					}
					if math.Abs(split.Tax-1.5) > 0.01 {
						t.Errorf("%s tax = %v, want 1.5", person, split.Tax)
					}
					if math.Abs(split.Total-16.5) > 0.01 {
						t.Errorf("%s total = %v, want 16.5", person, split.Total)
					}
				}
			},
		},
		{
			name:         "no items - three people split",
			items:        []Item{},
			billTotal:    90.0,
			billSubtotal: 75.0,
			participants: []string{"Alice", "Bob", "Charlie"},
			wantErr:      false,
			validateFunc: func(t *testing.T, splits map[string]*PersonSplit) {
				// Total = 90 / 3 = 30 each
				// Subtotal = 75 / 3 = 25 each
				// Tax = 15 / 3 = 5 each
				for _, person := range []string{"Alice", "Bob", "Charlie"} {
					split := splits[person]
					if math.Abs(split.Subtotal-25.0) > 0.01 {
						t.Errorf("%s subtotal = %v, want 25.0", person, split.Subtotal)
					}
					if math.Abs(split.Tax-5.0) > 0.01 {
						t.Errorf("%s tax = %v, want 5.0", person, split.Tax)
					}
					if math.Abs(split.Total-30.0) > 0.01 {
						t.Errorf("%s total = %v, want 30.0", person, split.Total)
					}
				}
			},
		},
		{
			name: "items don't cover full subtotal - remainder split equally",
			items: []Item{
				{Description: "Banana", Amount: 10.0, Participants: []string{"Ree"}},
			},
			billTotal:    100.0,
			billSubtotal: 90.0,
			participants: []string{"Mo", "Ree"},
			wantErr:      false,
			validateFunc: func(t *testing.T, splits map[string]*PersonSplit) {
				// Banana ($10) -> Ree only
				// Remainder ($80) split equally -> $40 each
				// Mo: subtotal = 40, Ree: subtotal = 10 + 40 = 50
				// Tax = 10, distributed proportionally
				// Mo: tax = 40 * (10/90) = 4.44, total = 44.44
				// Ree: tax = 50 * (10/90) = 5.56, total = 55.56
				mo := splits["Mo"]
				if math.Abs(mo.Subtotal-40.0) > 0.01 {
					t.Errorf("Mo subtotal = %v, want 40.0", mo.Subtotal)
				}
				if len(mo.Items) != 1 || mo.Items[0].Description != "Shared" {
					t.Errorf("Mo should have 1 'Shared' item, got %v", mo.Items)
				}

				ree := splits["Ree"]
				if math.Abs(ree.Subtotal-50.0) > 0.01 {
					t.Errorf("Ree subtotal = %v, want 50.0", ree.Subtotal)
				}
				if len(ree.Items) != 2 {
					t.Errorf("Ree should have 2 items (Banana + Shared), got %d", len(ree.Items))
				}

				// Verify totals add up to bill total
				totalOwed := mo.Total + ree.Total
				if math.Abs(totalOwed-100.0) > 0.01 {
					t.Errorf("Total owed = %v, want 100.0", totalOwed)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splits, err := CalculateSplit(tt.items, tt.billTotal, tt.billSubtotal, tt.participants)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateSplit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validateFunc != nil {
				tt.validateFunc(t, splits)
			}
		})
	}
}

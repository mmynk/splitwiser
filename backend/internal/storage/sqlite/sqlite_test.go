package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmynk/splitwiser/internal/models"
)

func TestSQLiteStore(t *testing.T) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "splitwiser-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("CreateBill generates ID and title", func(t *testing.T) {
		bill := &models.Bill{
			Total:        33.0,
			Subtotal:     30.0,
			Participants: []string{"Alice", "Bob"},
			Items: []models.Item{
				{Description: "Pizza", Amount: 20.0, AssignedTo: []string{"Alice", "Bob"}},
				{Description: "Beer", Amount: 10.0, AssignedTo: []string{"Bob"}},
			},
		}

		err := store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		if bill.ID == "" {
			t.Error("Expected bill ID to be generated")
		}
		if bill.Title == "" {
			t.Error("Expected bill title to be generated")
		}
		if bill.CreatedAt == 0 {
			t.Error("Expected CreatedAt to be set")
		}

		t.Logf("Created bill: ID=%s, Title=%s", bill.ID, bill.Title)
	})

	t.Run("GetBill retrieves complete bill", func(t *testing.T) {
		// Create a bill first
		original := &models.Bill{
			Title:        "Test Dinner",
			Total:        55.0,
			Subtotal:     50.0,
			Participants: []string{"Charlie", "Diana"},
			Items: []models.Item{
				{Description: "Steak", Amount: 30.0, AssignedTo: []string{"Charlie"}},
				{Description: "Salad", Amount: 20.0, AssignedTo: []string{"Diana"}},
			},
		}

		err := store.CreateBill(ctx, original)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		// Retrieve it
		retrieved, err := store.GetBill(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetBill failed: %v", err)
		}

		// Verify fields
		if retrieved.ID != original.ID {
			t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, original.ID)
		}
		if retrieved.Title != original.Title {
			t.Errorf("Title mismatch: got %s, want %s", retrieved.Title, original.Title)
		}
		if retrieved.Total != original.Total {
			t.Errorf("Total mismatch: got %f, want %f", retrieved.Total, original.Total)
		}
		if retrieved.Subtotal != original.Subtotal {
			t.Errorf("Subtotal mismatch: got %f, want %f", retrieved.Subtotal, original.Subtotal)
		}
		if len(retrieved.Participants) != len(original.Participants) {
			t.Errorf("Participants count mismatch: got %d, want %d", len(retrieved.Participants), len(original.Participants))
		}
		if len(retrieved.Items) != len(original.Items) {
			t.Errorf("Items count mismatch: got %d, want %d", len(retrieved.Items), len(original.Items))
		}

		// Verify item assignments
		for i, item := range retrieved.Items {
			if len(item.AssignedTo) != len(original.Items[i].AssignedTo) {
				t.Errorf("Item %d assignments mismatch: got %d, want %d",
					i, len(item.AssignedTo), len(original.Items[i].AssignedTo))
			}
		}
	})

	t.Run("GetBill returns error for nonexistent bill", func(t *testing.T) {
		_, err := store.GetBill(ctx, "nonexistent-id")
		if err == nil {
			t.Error("Expected error for nonexistent bill, got nil")
		}
	})

	t.Run("CreateBill with no items (equal split)", func(t *testing.T) {
		bill := &models.Bill{
			Total:        100.0,
			Subtotal:     100.0,
			Participants: []string{"Eve", "Frank", "Grace"},
			Items:        []models.Item{}, // No items - equal split
		}

		err := store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		retrieved, err := store.GetBill(ctx, bill.ID)
		if err != nil {
			t.Fatalf("GetBill failed: %v", err)
		}

		if len(retrieved.Items) != 0 {
			t.Errorf("Expected 0 items, got %d", len(retrieved.Items))
		}
		if len(retrieved.Participants) != 3 {
			t.Errorf("Expected 3 participants, got %d", len(retrieved.Participants))
		}
	})

	t.Run("Auto-generated title format", func(t *testing.T) {
		// Two participants
		bill1 := &models.Bill{
			Total:        20.0,
			Subtotal:     20.0,
			Participants: []string{"Alice", "Bob"},
		}
		store.CreateBill(ctx, bill1)
		if bill1.Title != "Split with Alice, Bob" {
			t.Errorf("Unexpected title for 2 participants: %s", bill1.Title)
		}

		// Three participants
		bill2 := &models.Bill{
			Total:        30.0,
			Subtotal:     30.0,
			Participants: []string{"Alice", "Bob", "Charlie"},
		}
		store.CreateBill(ctx, bill2)
		if bill2.Title != "Split with Alice, Bob, Charlie" {
			t.Errorf("Unexpected title for 3 participants: %s", bill2.Title)
		}

		// Four+ participants (shows "and X others")
		bill3 := &models.Bill{
			Total:        40.0,
			Subtotal:     40.0,
			Participants: []string{"Alice", "Bob", "Charlie", "Diana"},
		}
		store.CreateBill(ctx, bill3)
		if bill3.Title != "Split with Alice, Bob and 2 others" {
			t.Errorf("Unexpected title for 4 participants: %s", bill3.Title)
		}
	})
}

func TestGenerateTitle(t *testing.T) {
	tests := []struct {
		participants []string
		wantContains string
	}{
		{[]string{}, "Bill -"},
		{[]string{"Alice"}, "Split with Alice"},
		{[]string{"Alice", "Bob"}, "Split with Alice, Bob"},
		{[]string{"Alice", "Bob", "Charlie"}, "Split with Alice, Bob, Charlie"},
		{[]string{"Alice", "Bob", "Charlie", "Diana"}, "and 2 others"},
	}

	for _, tt := range tests {
		t.Run(tt.wantContains, func(t *testing.T) {
			got := generateTitle(tt.participants)
			if !contains(got, tt.wantContains) {
				t.Errorf("generateTitle(%v) = %q, want to contain %q", tt.participants, got, tt.wantContains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

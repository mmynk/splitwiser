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
				{Description: "Pizza", Amount: 20.0, Participants: []string{"Alice", "Bob"}},
				{Description: "Beer", Amount: 10.0, Participants: []string{"Bob"}},
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
				{Description: "Steak", Amount: 30.0, Participants: []string{"Charlie"}},
				{Description: "Salad", Amount: 20.0, Participants: []string{"Diana"}},
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
			if len(item.Participants) != len(original.Items[i].Participants) {
				t.Errorf("Item %d assignments mismatch: got %d, want %d",
					i, len(item.Participants), len(original.Items[i].Participants))
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

	t.Run("UpdateBill modifies existing bill", func(t *testing.T) {
		// Create a bill first
		original := &models.Bill{
			Title:        "Original Dinner",
			Total:        50.0,
			Subtotal:     45.0,
			Participants: []string{"Alice", "Bob"},
			Items: []models.Item{
				{Description: "Pasta", Amount: 25.0, Participants: []string{"Alice"}},
				{Description: "Wine", Amount: 20.0, Participants: []string{"Bob"}},
			},
		}

		err := store.CreateBill(ctx, original)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		// Update the bill
		updated := &models.Bill{
			ID:           original.ID,
			Title:        "Updated Dinner",
			Total:        75.0,
			Subtotal:     70.0,
			Participants: []string{"Alice", "Bob", "Charlie"},
			Items: []models.Item{
				{Description: "Pizza", Amount: 30.0, Participants: []string{"Alice", "Bob"}},
				{Description: "Beer", Amount: 20.0, Participants: []string{"Charlie"}},
				{Description: "Dessert", Amount: 20.0, Participants: []string{"Alice", "Bob", "Charlie"}},
			},
		}

		err = store.UpdateBill(ctx, updated)
		if err != nil {
			t.Fatalf("UpdateBill failed: %v", err)
		}

		// Retrieve and verify
		retrieved, err := store.GetBill(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetBill after update failed: %v", err)
		}

		if retrieved.Title != "Updated Dinner" {
			t.Errorf("Title not updated: got %s, want Updated Dinner", retrieved.Title)
		}
		if retrieved.Total != 75.0 {
			t.Errorf("Total not updated: got %f, want 75.0", retrieved.Total)
		}
		if retrieved.Subtotal != 70.0 {
			t.Errorf("Subtotal not updated: got %f, want 70.0", retrieved.Subtotal)
		}
		if len(retrieved.Participants) != 3 {
			t.Errorf("Participants count mismatch: got %d, want 3", len(retrieved.Participants))
		}
		if len(retrieved.Items) != 3 {
			t.Errorf("Items count mismatch: got %d, want 3", len(retrieved.Items))
		}
	})

	t.Run("UpdateBill returns error for nonexistent bill", func(t *testing.T) {
		bill := &models.Bill{
			ID:           "nonexistent-id",
			Title:        "Test",
			Total:        10.0,
			Subtotal:     10.0,
			Participants: []string{"Alice"},
		}

		err := store.UpdateBill(ctx, bill)
		if err == nil {
			t.Error("Expected error for nonexistent bill, got nil")
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
		if bill3.Title != "Split with Alice, Bob & 2 others" {
			t.Errorf("Unexpected title for 4 participants: %s", bill3.Title)
		}
	})
}

func TestGenerateTitle(t *testing.T) {
	tests := []struct {
		items        []models.Item
		participants []string
		wantContains string
	}{
		{nil, []string{}, "Bill -"},
		{nil, []string{"Alice"}, "Split with Alice"},
		{nil, []string{"Alice", "Bob"}, "Split with Alice, Bob"},
		{nil, []string{"Alice", "Bob", "Charlie"}, "Split with Alice, Bob, Charlie"},
		{nil, []string{"Alice", "Bob", "Charlie", "Diana"}, "& 2 others"},
		{[]models.Item{{Description: "Pizza"}}, []string{"Alice", "Bob"}, "Pizza - Alice, Bob"},
		{[]models.Item{{Description: "Pizza"}, {Description: "Beer"}}, []string{"Alice", "Bob"}, "Pizza, Beer - Alice, Bob"},
		{[]models.Item{{Description: "Pizza"}, {Description: "Beer"}, {Description: "Fries"}, {Description: "Coke"}}, []string{"Alice"}, "Pizza, Beer & 2 more - Alice"},
		{[]models.Item{{Description: "Pizza"}}, nil, "Pizza"},
	}

	for _, tt := range tests {
		t.Run(tt.wantContains, func(t *testing.T) {
			got := generateTitle(tt.items, tt.participants)
			if !contains(got, tt.wantContains) {
				t.Errorf("generateTitle(items=%d, participants=%v) = %q, want to contain %q", len(tt.items), tt.participants, got, tt.wantContains)
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

func TestGroupStorage(t *testing.T) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "splitwiser-group-test-*")
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

	t.Run("CreateGroup generates ID and timestamp", func(t *testing.T) {
		group := &models.Group{
			Name:    "Roommates",
			Members: []string{"Alice", "Bob", "Charlie"},
		}

		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		if group.ID == "" {
			t.Error("Expected group ID to be generated")
		}
		if group.CreatedAt == 0 {
			t.Error("Expected CreatedAt to be set")
		}

		t.Logf("Created group: ID=%s, Name=%s", group.ID, group.Name)
	})

	t.Run("GetGroup retrieves complete group", func(t *testing.T) {
		original := &models.Group{
			Name:    "Work Lunch",
			Members: []string{"Diana", "Eve", "Frank"},
		}

		err := store.CreateGroup(ctx, original)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		retrieved, err := store.GetGroup(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetGroup failed: %v", err)
		}

		if retrieved.ID != original.ID {
			t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, original.ID)
		}
		if retrieved.Name != original.Name {
			t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, original.Name)
		}
		if len(retrieved.Members) != len(original.Members) {
			t.Errorf("Members count mismatch: got %d, want %d", len(retrieved.Members), len(original.Members))
		}
	})

	t.Run("GetGroup returns error for nonexistent group", func(t *testing.T) {
		_, err := store.GetGroup(ctx, "nonexistent-id")
		if err == nil {
			t.Error("Expected error for nonexistent group, got nil")
		}
	})

	t.Run("ListGroups returns all groups", func(t *testing.T) {
		// Create a few more groups
		store.CreateGroup(ctx, &models.Group{Name: "Group A", Members: []string{"A1", "A2"}})
		store.CreateGroup(ctx, &models.Group{Name: "Group B", Members: []string{"B1", "B2"}})

		groups, err := store.ListGroups(ctx)
		if err != nil {
			t.Fatalf("ListGroups failed: %v", err)
		}

		if len(groups) < 2 {
			t.Errorf("Expected at least 2 groups, got %d", len(groups))
		}

		// Verify groups have members populated
		for _, g := range groups {
			if len(g.Members) == 0 {
				t.Errorf("Group %s has no members", g.Name)
			}
		}
	})

	t.Run("UpdateGroup modifies existing group", func(t *testing.T) {
		original := &models.Group{
			Name:    "Original Name",
			Members: []string{"X", "Y"},
		}

		err := store.CreateGroup(ctx, original)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		updated := &models.Group{
			ID:      original.ID,
			Name:    "Updated Name",
			Members: []string{"X", "Y", "Z"},
		}

		err = store.UpdateGroup(ctx, updated)
		if err != nil {
			t.Fatalf("UpdateGroup failed: %v", err)
		}

		retrieved, err := store.GetGroup(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetGroup after update failed: %v", err)
		}

		if retrieved.Name != "Updated Name" {
			t.Errorf("Name not updated: got %s, want Updated Name", retrieved.Name)
		}
		if len(retrieved.Members) != 3 {
			t.Errorf("Members count mismatch: got %d, want 3", len(retrieved.Members))
		}
	})

	t.Run("UpdateGroup returns error for nonexistent group", func(t *testing.T) {
		group := &models.Group{
			ID:      "nonexistent-id",
			Name:    "Test",
			Members: []string{"A"},
		}

		err := store.UpdateGroup(ctx, group)
		if err == nil {
			t.Error("Expected error for nonexistent group, got nil")
		}
	})

	t.Run("DeleteGroup removes group", func(t *testing.T) {
		group := &models.Group{
			Name:    "To Be Deleted",
			Members: []string{"Delete", "Me"},
		}

		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		err = store.DeleteGroup(ctx, group.ID)
		if err != nil {
			t.Fatalf("DeleteGroup failed: %v", err)
		}

		_, err = store.GetGroup(ctx, group.ID)
		if err == nil {
			t.Error("Expected error getting deleted group, got nil")
		}
	})

	t.Run("DeleteGroup returns error for nonexistent group", func(t *testing.T) {
		err := store.DeleteGroup(ctx, "nonexistent-id")
		if err == nil {
			t.Error("Expected error for nonexistent group, got nil")
		}
	})
}

func TestBillWithGroup(t *testing.T) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "splitwiser-bill-group-test-*")
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

	t.Run("Bill with group_id", func(t *testing.T) {
		// Create a group first
		group := &models.Group{
			Name:    "Test Group",
			Members: []string{"Alice", "Bob"},
		}
		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		// Create bill with group_id
		bill := &models.Bill{
			Title:        "Group Dinner",
			Total:        50.0,
			Subtotal:     45.0,
			Participants: []string{"Alice", "Bob"},
			GroupID:      group.ID,
		}

		err = store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		// Retrieve and verify group_id
		retrieved, err := store.GetBill(ctx, bill.ID)
		if err != nil {
			t.Fatalf("GetBill failed: %v", err)
		}

		if retrieved.GroupID != group.ID {
			t.Errorf("GroupID mismatch: got %s, want %s", retrieved.GroupID, group.ID)
		}
	})

	t.Run("Bill without group_id", func(t *testing.T) {
		bill := &models.Bill{
			Title:        "No Group Dinner",
			Total:        30.0,
			Subtotal:     27.0,
			Participants: []string{"Charlie"},
		}

		err := store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		retrieved, err := store.GetBill(ctx, bill.ID)
		if err != nil {
			t.Fatalf("GetBill failed: %v", err)
		}

		if retrieved.GroupID != "" {
			t.Errorf("Expected empty GroupID, got %s", retrieved.GroupID)
		}
	})

	t.Run("UpdateBill with group_id", func(t *testing.T) {
		// Create a group
		group := &models.Group{
			Name:    "Update Test Group",
			Members: []string{"Diana"},
		}
		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		// Create bill without group
		bill := &models.Bill{
			Title:        "Update Test",
			Total:        20.0,
			Subtotal:     18.0,
			Participants: []string{"Diana"},
		}
		err = store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		// Update to add group_id
		bill.GroupID = group.ID
		err = store.UpdateBill(ctx, bill)
		if err != nil {
			t.Fatalf("UpdateBill failed: %v", err)
		}

		retrieved, err := store.GetBill(ctx, bill.ID)
		if err != nil {
			t.Fatalf("GetBill failed: %v", err)
		}

		if retrieved.GroupID != group.ID {
			t.Errorf("GroupID not updated: got %s, want %s", retrieved.GroupID, group.ID)
		}
	})

	t.Run("Deleting group sets bill group_id to null", func(t *testing.T) {
		// Create a group
		group := &models.Group{
			Name:    "Cascade Test Group",
			Members: []string{"Eve"},
		}
		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		// Create bill with group
		bill := &models.Bill{
			Title:        "Cascade Test",
			Total:        15.0,
			Subtotal:     14.0,
			Participants: []string{"Eve"},
			GroupID:      group.ID,
		}
		err = store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		// Delete the group
		err = store.DeleteGroup(ctx, group.ID)
		if err != nil {
			t.Fatalf("DeleteGroup failed: %v", err)
		}

		// Bill should still exist but with empty group_id
		retrieved, err := store.GetBill(ctx, bill.ID)
		if err != nil {
			t.Fatalf("GetBill after group delete failed: %v", err)
		}

		if retrieved.GroupID != "" {
			t.Errorf("Expected empty GroupID after group deletion, got %s", retrieved.GroupID)
		}
	})
}

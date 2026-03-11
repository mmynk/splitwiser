package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmynk/splitwiser/internal/models"
)

// bp is a helper to build []models.BillParticipant from display names (no user_id).
func bp(names ...string) []models.BillParticipant {
	out := make([]models.BillParticipant, len(names))
	for i, n := range names {
		out[i] = models.BillParticipant{DisplayName: n}
	}
	return out
}

// bpWithID is a helper to build a single BillParticipant with a user_id.
func bpWithID(name, userID string) models.BillParticipant {
	return models.BillParticipant{DisplayName: name, UserID: userID}
}

// gm is a helper to build []models.GroupMember from display names (no user_id).
func gm(names ...string) []models.GroupMember {
	out := make([]models.GroupMember, len(names))
	for i, n := range names {
		out[i] = models.GroupMember{DisplayName: n}
	}
	return out
}

// gmWithID is a helper to build a single GroupMember with a user_id.
func gmWithID(name, userID string) models.GroupMember {
	return models.GroupMember{DisplayName: name, UserID: userID}
}

func TestSQLiteStore(t *testing.T) {
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
			Participants: bp("Alice", "Bob"),
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
		original := &models.Bill{
			Title:        "Test Dinner",
			Total:        55.0,
			Subtotal:     50.0,
			Participants: bp("Charlie", "Diana"),
			Items: []models.Item{
				{Description: "Steak", Amount: 30.0, Participants: []string{"Charlie"}},
				{Description: "Salad", Amount: 20.0, Participants: []string{"Diana"}},
			},
		}

		err := store.CreateBill(ctx, original)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		retrieved, err := store.GetBill(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetBill failed: %v", err)
		}

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
			Participants: bp("Eve", "Frank", "Grace"),
			Items:        []models.Item{},
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
		original := &models.Bill{
			Title:        "Original Dinner",
			Total:        50.0,
			Subtotal:     45.0,
			Participants: bp("Alice", "Bob"),
			Items: []models.Item{
				{Description: "Pasta", Amount: 25.0, Participants: []string{"Alice"}},
				{Description: "Wine", Amount: 20.0, Participants: []string{"Bob"}},
			},
		}

		err := store.CreateBill(ctx, original)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		updated := &models.Bill{
			ID:           original.ID,
			Title:        "Updated Dinner",
			Total:        75.0,
			Subtotal:     70.0,
			Participants: bp("Alice", "Bob", "Charlie"),
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
			Participants: bp("Alice"),
		}

		err := store.UpdateBill(ctx, bill)
		if err == nil {
			t.Error("Expected error for nonexistent bill, got nil")
		}
	})

	t.Run("Auto-generated title format", func(t *testing.T) {
		bill1 := &models.Bill{
			Total:        20.0,
			Subtotal:     20.0,
			Participants: bp("Alice", "Bob"),
		}
		store.CreateBill(ctx, bill1)
		if bill1.Title != "Split with Alice, Bob" {
			t.Errorf("Unexpected title for 2 participants: %s", bill1.Title)
		}

		bill2 := &models.Bill{
			Total:        30.0,
			Subtotal:     30.0,
			Participants: bp("Alice", "Bob", "Charlie"),
		}
		store.CreateBill(ctx, bill2)
		if bill2.Title != "Split with Alice, Bob, Charlie" {
			t.Errorf("Unexpected title for 3 participants: %s", bill2.Title)
		}

		bill3 := &models.Bill{
			Total:        40.0,
			Subtotal:     40.0,
			Participants: bp("Alice", "Bob", "Charlie", "Diana"),
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
		participants []models.BillParticipant
		wantContains string
	}{
		{nil, []models.BillParticipant{}, "Bill -"},
		{nil, bp("Alice"), "Split with Alice"},
		{nil, bp("Alice", "Bob"), "Split with Alice, Bob"},
		{nil, bp("Alice", "Bob", "Charlie"), "Split with Alice, Bob, Charlie"},
		{nil, bp("Alice", "Bob", "Charlie", "Diana"), "& 2 others"},
		{[]models.Item{{Description: "Pizza"}}, bp("Alice", "Bob"), "Pizza - Alice, Bob"},
		{[]models.Item{{Description: "Pizza"}, {Description: "Beer"}}, bp("Alice", "Bob"), "Pizza, Beer - Alice, Bob"},
		{[]models.Item{{Description: "Pizza"}, {Description: "Beer"}, {Description: "Fries"}, {Description: "Coke"}}, bp("Alice"), "Pizza, Beer & 2 more - Alice"},
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
			Members: gm("Alice", "Bob", "Charlie"),
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
			Members: gm("Diana", "Eve", "Frank"),
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

	t.Run("ListGroupsByUser filters by user_id", func(t *testing.T) {
		// Groups must have user_id set to be found by ListGroupsByUser
		store.CreateGroup(ctx, &models.Group{Name: "Group A", Members: []models.GroupMember{
			gmWithID("A1", "uuid-a1"),
			gmWithID("A2", "uuid-a2"),
		}})
		store.CreateGroup(ctx, &models.Group{Name: "Group B", Members: []models.GroupMember{
			gmWithID("B1", "uuid-b1"),
			gmWithID("B2", "uuid-b2"),
		}})

		groupsA, err := store.ListGroupsByUser(ctx, "uuid-a1")
		if err != nil {
			t.Fatalf("ListGroupsByUser failed: %v", err)
		}
		if len(groupsA) < 1 {
			t.Errorf("Expected at least 1 group for uuid-a1, got %d", len(groupsA))
		}
		for _, g := range groupsA {
			if len(g.Members) == 0 {
				t.Errorf("Group %s has no members", g.Name)
			}
		}

		groupsB, err := store.ListGroupsByUser(ctx, "uuid-b1")
		if err != nil {
			t.Fatalf("ListGroupsByUser failed: %v", err)
		}
		if len(groupsB) < 1 {
			t.Errorf("Expected at least 1 group for uuid-b1, got %d", len(groupsB))
		}
		for _, g := range groupsB {
			for _, m := range g.Members {
				if m.UserID == "uuid-a1" || m.UserID == "uuid-a2" {
					t.Errorf("uuid-b1 should not see Group A members")
				}
			}
		}
	})

	t.Run("UpdateGroup modifies existing group", func(t *testing.T) {
		original := &models.Group{
			Name:    "Original Name",
			Members: gm("X", "Y"),
		}

		err := store.CreateGroup(ctx, original)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		updated := &models.Group{
			ID:      original.ID,
			Name:    "Updated Name",
			Members: gm("X", "Y", "Z"),
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
			Members: gm("A"),
		}

		err := store.UpdateGroup(ctx, group)
		if err == nil {
			t.Error("Expected error for nonexistent group, got nil")
		}
	})

	t.Run("DeleteGroup removes group", func(t *testing.T) {
		group := &models.Group{
			Name:    "To Be Deleted",
			Members: gm("Delete", "Me"),
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

func TestAddGroupMembers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "splitwiser-addmembers-test-*")
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

	group := &models.Group{
		Name:    "Test Group",
		Members: gm("Alice", "Bob"),
	}
	err = store.CreateGroup(ctx, group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	t.Run("adds new members to group", func(t *testing.T) {
		err := store.AddGroupMembers(ctx, group.ID, []string{"Charlie", "Diana"})
		if err != nil {
			t.Fatalf("AddGroupMembers failed: %v", err)
		}

		retrieved, err := store.GetGroup(ctx, group.ID)
		if err != nil {
			t.Fatalf("GetGroup failed: %v", err)
		}

		if len(retrieved.Members) != 4 {
			t.Errorf("Expected 4 members, got %d: %v", len(retrieved.Members), retrieved.Members)
		}
	})

	t.Run("idempotent - no duplicates", func(t *testing.T) {
		err := store.AddGroupMembers(ctx, group.ID, []string{"Alice", "Charlie", "Eve"})
		if err != nil {
			t.Fatalf("AddGroupMembers failed: %v", err)
		}

		retrieved, err := store.GetGroup(ctx, group.ID)
		if err != nil {
			t.Fatalf("GetGroup failed: %v", err)
		}

		if len(retrieved.Members) != 5 {
			t.Errorf("Expected 5 members (no duplicates), got %d: %v", len(retrieved.Members), retrieved.Members)
		}
	})

	t.Run("empty member list is no-op", func(t *testing.T) {
		err := store.AddGroupMembers(ctx, group.ID, []string{})
		if err != nil {
			t.Fatalf("AddGroupMembers with empty list failed: %v", err)
		}
	})
}

func TestBillWithGroup(t *testing.T) {
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
		group := &models.Group{
			Name:    "Test Group",
			Members: gm("Alice", "Bob"),
		}
		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		bill := &models.Bill{
			Title:        "Group Dinner",
			Total:        50.0,
			Subtotal:     45.0,
			Participants: bp("Alice", "Bob"),
			GroupID:      group.ID,
		}

		err = store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

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
			Participants: bp("Charlie"),
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
		group := &models.Group{
			Name:    "Update Test Group",
			Members: gm("Diana"),
		}
		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		bill := &models.Bill{
			Title:        "Update Test",
			Total:        20.0,
			Subtotal:     18.0,
			Participants: bp("Diana"),
		}
		err = store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

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
		group := &models.Group{
			Name:    "Cascade Test Group",
			Members: gm("Eve"),
		}
		err := store.CreateGroup(ctx, group)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		bill := &models.Bill{
			Title:        "Cascade Test",
			Total:        15.0,
			Subtotal:     14.0,
			Participants: bp("Eve"),
			GroupID:      group.ID,
		}
		err = store.CreateBill(ctx, bill)
		if err != nil {
			t.Fatalf("CreateBill failed: %v", err)
		}

		err = store.DeleteGroup(ctx, group.ID)
		if err != nil {
			t.Fatalf("DeleteGroup failed: %v", err)
		}

		retrieved, err := store.GetBill(ctx, bill.ID)
		if err != nil {
			t.Fatalf("GetBill after group delete failed: %v", err)
		}

		if retrieved.GroupID != "" {
			t.Errorf("Expected empty GroupID after group deletion, got %s", retrieved.GroupID)
		}
	})
}

func TestListBillsByUser(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "splitwiser-listbills-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := New(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	const aliceID = "uuid-alice"
	const bobID = "uuid-bob"

	// Bill where Alice is a participant
	bill1 := &models.Bill{
		Title:    "Dinner",
		Total:    22.0,
		Subtotal: 20.0,
		Participants: []models.BillParticipant{
			bpWithID("Alice", aliceID),
			bpWithID("Bob", bobID),
		},
	}
	if err := store.CreateBill(ctx, bill1); err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	// Bill where Bob is the only participant (Alice not involved at all)
	bill2 := &models.Bill{
		Title:    "Bob Only",
		Total:    10.0,
		Subtotal: 10.0,
		Participants: []models.BillParticipant{
			bpWithID("Bob", bobID),
		},
	}
	if err := store.CreateBill(ctx, bill2); err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

	// Bill where Alice is a solo participant
	bill3 := &models.Bill{
		Title:    "Alice Solo",
		Total:    5.0,
		Subtotal: 5.0,
		Participants: []models.BillParticipant{
			bpWithID("Alice", aliceID),
		},
	}
	if err := store.CreateBill(ctx, bill3); err != nil {
		t.Fatalf("CreateBill 3 failed: %v", err)
	}

	// Bill where Alice is the creator but NOT a participant
	bill4 := &models.Bill{
		Title:     "Alice Created, Bob Pays",
		Total:     15.0,
		Subtotal:  15.0,
		CreatorID: aliceID,
		Participants: []models.BillParticipant{
			bpWithID("Bob", bobID),
		},
	}
	if err := store.CreateBill(ctx, bill4); err != nil {
		t.Fatalf("CreateBill 4 failed: %v", err)
	}

	t.Run("returns bills where user is participant", func(t *testing.T) {
		bills, err := store.ListBillsByUser(ctx, aliceID)
		if err != nil {
			t.Fatalf("ListBillsByUser failed: %v", err)
		}

		// Alice should see bill1 (participant), bill3 (participant), bill4 (creator)
		if len(bills) != 3 {
			t.Fatalf("expected 3 bills for Alice, got %d", len(bills))
		}

		billIDs := map[string]bool{bill1.ID: false, bill3.ID: false, bill4.ID: false}
		for _, b := range bills {
			if _, ok := billIDs[b.ID]; ok {
				billIDs[b.ID] = true
			}
		}
		for id, found := range billIDs {
			if !found {
				t.Errorf("expected bill %s in Alice's bills", id)
			}
		}

		for _, b := range bills {
			if b.ID == bill2.ID {
				t.Error("Bob-only bill (no Alice) should not appear in Alice's bills")
			}
		}
	})

	t.Run("creator-only bill appears without participant entry", func(t *testing.T) {
		bills, err := store.ListBillsByUser(ctx, aliceID)
		if err != nil {
			t.Fatalf("ListBillsByUser failed: %v", err)
		}
		var found bool
		for _, b := range bills {
			if b.ID == bill4.ID {
				found = true
				// Alice should not appear in the participants list of this bill
				for _, p := range b.Participants {
					if p.UserID == aliceID {
						t.Error("Alice should not be a participant on bill4, only creator")
					}
				}
			}
		}
		if !found {
			t.Error("creator-only bill4 not found in Alice's bills")
		}
	})

	t.Run("returns empty slice for nonexistent user_id", func(t *testing.T) {
		bills, err := store.ListBillsByUser(ctx, "nonexistent-uuid")
		if err != nil {
			t.Fatalf("ListBillsByUser failed: %v", err)
		}
		if len(bills) != 0 {
			t.Errorf("expected 0 bills for nonexistent uuid, got %d", len(bills))
		}
	})

	t.Run("participants are populated on returned bills", func(t *testing.T) {
		bills, err := store.ListBillsByUser(ctx, aliceID)
		if err != nil {
			t.Fatalf("ListBillsByUser failed: %v", err)
		}
		for _, b := range bills {
			if len(b.Participants) == 0 {
				t.Errorf("bill %s has no participants", b.ID)
			}
		}
	})
}

func TestSettlementStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "splitwiser-settlement-test-*")
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

	aliceUser := &models.User{
		ID:           "alice-id",
		Email:        "alice@test.com",
		DisplayName:  "Alice",
		PasswordHash: "hash",
		CreatedAt:    1000,
		UpdatedAt:    1000,
	}
	bobUser := &models.User{
		ID:           "bob-id",
		Email:        "bob@test.com",
		DisplayName:  "Bob",
		PasswordHash: "hash",
		CreatedAt:    1000,
		UpdatedAt:    1000,
	}

	err = store.CreateUser(ctx, aliceUser)
	if err != nil {
		t.Fatalf("CreateUser (Alice) failed: %v", err)
	}
	err = store.CreateUser(ctx, bobUser)
	if err != nil {
		t.Fatalf("CreateUser (Bob) failed: %v", err)
	}

	group := &models.Group{
		Name:    "Settlement Test Group",
		Members: gm(aliceUser.ID, bobUser.ID),
	}
	err = store.CreateGroup(ctx, group)
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	t.Run("CreateSettlement generates ID and timestamp", func(t *testing.T) {
		settlement := &models.Settlement{
			GroupID:    group.ID,
			FromUserID: bobUser.ID,
			ToUserID:   aliceUser.ID,
			Amount:     50.0,
			CreatedBy:  bobUser.ID,
			Note:       "Venmo payment",
		}

		err := store.CreateSettlement(ctx, settlement)
		if err != nil {
			t.Fatalf("CreateSettlement failed: %v", err)
		}

		if settlement.ID == "" {
			t.Error("Expected settlement ID to be generated")
		}
		if settlement.CreatedAt == 0 {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("GetSettlement retrieves complete settlement", func(t *testing.T) {
		original := &models.Settlement{
			GroupID:    group.ID,
			FromUserID: aliceUser.ID,
			ToUserID:   bobUser.ID,
			Amount:     25.0,
			CreatedBy:  aliceUser.ID,
			Note:       "Cash payment",
		}

		err := store.CreateSettlement(ctx, original)
		if err != nil {
			t.Fatalf("CreateSettlement failed: %v", err)
		}

		retrieved, err := store.GetSettlement(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetSettlement failed: %v", err)
		}

		if retrieved.ID != original.ID {
			t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, original.ID)
		}
		if retrieved.Amount != original.Amount {
			t.Errorf("Amount mismatch: got %f, want %f", retrieved.Amount, original.Amount)
		}
	})

	t.Run("GetSettlement returns error for nonexistent settlement", func(t *testing.T) {
		_, err := store.GetSettlement(ctx, "nonexistent-id")
		if err == nil {
			t.Error("Expected error for nonexistent settlement, got nil")
		}
	})

	t.Run("ListSettlementsByGroup returns settlements for group", func(t *testing.T) {
		charlieUser := &models.User{
			ID:           "charlie-id",
			Email:        "charlie@test.com",
			DisplayName:  "Charlie",
			PasswordHash: "hash",
			CreatedAt:    1000,
			UpdatedAt:    1000,
		}
		err = store.CreateUser(ctx, charlieUser)
		if err != nil {
			t.Fatalf("CreateUser (Charlie) failed: %v", err)
		}

		group2 := &models.Group{
			Name:    "Another Group",
			Members: gm(charlieUser.ID, aliceUser.ID),
		}
		err = store.CreateGroup(ctx, group2)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		store.CreateSettlement(ctx, &models.Settlement{
			GroupID:    group2.ID,
			FromUserID: charlieUser.ID,
			ToUserID:   aliceUser.ID,
			Amount:     10.0,
			CreatedBy:  charlieUser.ID,
		})
		store.CreateSettlement(ctx, &models.Settlement{
			GroupID:    group2.ID,
			FromUserID: charlieUser.ID,
			ToUserID:   aliceUser.ID,
			Amount:     20.0,
			CreatedBy:  charlieUser.ID,
		})

		settlements, err := store.ListSettlementsByGroup(ctx, group2.ID)
		if err != nil {
			t.Fatalf("ListSettlementsByGroup failed: %v", err)
		}

		if len(settlements) != 2 {
			t.Errorf("Expected 2 settlements, got %d", len(settlements))
		}
	})

	t.Run("DeleteSettlement removes settlement", func(t *testing.T) {
		settlement := &models.Settlement{
			GroupID:    group.ID,
			FromUserID: bobUser.ID,
			ToUserID:   aliceUser.ID,
			Amount:     15.0,
			CreatedBy:  bobUser.ID,
		}

		err := store.CreateSettlement(ctx, settlement)
		if err != nil {
			t.Fatalf("CreateSettlement failed: %v", err)
		}

		err = store.DeleteSettlement(ctx, settlement.ID)
		if err != nil {
			t.Fatalf("DeleteSettlement failed: %v", err)
		}

		_, err = store.GetSettlement(ctx, settlement.ID)
		if err == nil {
			t.Error("Expected error getting deleted settlement, got nil")
		}
	})

	t.Run("Settlements cascade delete when group is deleted", func(t *testing.T) {
		cascadeGroup := &models.Group{
			Name:    "Cascade Test Group",
			Members: gm(aliceUser.ID, bobUser.ID),
		}
		err = store.CreateGroup(ctx, cascadeGroup)
		if err != nil {
			t.Fatalf("CreateGroup failed: %v", err)
		}

		settlement := &models.Settlement{
			GroupID:    cascadeGroup.ID,
			FromUserID: bobUser.ID,
			ToUserID:   aliceUser.ID,
			Amount:     100.0,
			CreatedBy:  bobUser.ID,
		}
		err = store.CreateSettlement(ctx, settlement)
		if err != nil {
			t.Fatalf("CreateSettlement failed: %v", err)
		}

		err = store.DeleteGroup(ctx, cascadeGroup.ID)
		if err != nil {
			t.Fatalf("DeleteGroup failed: %v", err)
		}

		_, err = store.GetSettlement(ctx, settlement.ID)
		if err == nil {
			t.Error("Expected error getting settlement after group deletion, got nil")
		}
	})
}

func TestFriendshipStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "splitwiser-friendship-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := New(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	alice := &models.User{ID: "alice-id", Email: "alice@test.com", DisplayName: "Alice", PasswordHash: "h", CreatedAt: 1, UpdatedAt: 1}
	bob := &models.User{ID: "bob-id", Email: "bob@test.com", DisplayName: "Bob", PasswordHash: "h", CreatedAt: 1, UpdatedAt: 1}
	if err := store.CreateUser(ctx, alice); err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	if err := store.CreateUser(ctx, bob); err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}

	t.Run("AreFriends_PendingRequest_ReturnsFalse", func(t *testing.T) {
		f := &models.Friendship{RequesterID: alice.ID, AddresseeID: bob.ID, Status: models.FriendshipPending}
		if err := store.SendFriendRequest(ctx, f); err != nil {
			t.Fatalf("SendFriendRequest failed: %v", err)
		}

		ok, err := store.AreFriends(ctx, alice.ID, bob.ID)
		if err != nil {
			t.Fatalf("AreFriends failed: %v", err)
		}
		if ok {
			t.Error("Expected AreFriends to be false for pending request")
		}
	})

	t.Run("SendFriendRequest_Duplicate_Error", func(t *testing.T) {
		// Request already sent in previous subtest
		f2 := &models.Friendship{RequesterID: alice.ID, AddresseeID: bob.ID, Status: models.FriendshipPending}
		err := store.SendFriendRequest(ctx, f2)
		if err == nil {
			t.Error("Expected error for duplicate friend request, got nil")
		}
	})

	t.Run("SendFriendRequest_ReverseDirection_Error", func(t *testing.T) {
		f := &models.Friendship{RequesterID: bob.ID, AddresseeID: alice.ID, Status: models.FriendshipPending}
		err := store.SendFriendRequest(ctx, f)
		if err == nil {
			t.Error("Expected error for reverse direction friend request, got nil")
		}
	})

	t.Run("AreFriends_AcceptedRequest_ReturnsTrue", func(t *testing.T) {
		// Get the pending friendship created above
		friendships, err := store.ListFriendships(ctx, alice.ID, false, models.FriendshipPending)
		if err != nil {
			t.Fatalf("ListFriendships failed: %v", err)
		}
		if len(friendships) == 0 {
			t.Fatal("Expected at least one pending friendship")
		}
		fID := friendships[0].ID

		if err := store.UpdateFriendshipStatus(ctx, fID, models.FriendshipAccepted); err != nil {
			t.Fatalf("UpdateFriendshipStatus failed: %v", err)
		}

		// Test both directions
		ok, err := store.AreFriends(ctx, alice.ID, bob.ID)
		if err != nil || !ok {
			t.Errorf("Expected AreFriends(alice, bob)=true, got %v (err: %v)", ok, err)
		}
		ok, err = store.AreFriends(ctx, bob.ID, alice.ID)
		if err != nil || !ok {
			t.Errorf("Expected AreFriends(bob, alice)=true, got %v (err: %v)", ok, err)
		}
	})

	t.Run("ListFriendships_Incoming", func(t *testing.T) {
		// Bob has an incoming accepted request from Alice
		friendships, err := store.ListFriendships(ctx, bob.ID, true, models.FriendshipAccepted)
		if err != nil {
			t.Fatalf("ListFriendships incoming failed: %v", err)
		}
		if len(friendships) != 1 {
			t.Errorf("Expected 1 incoming friendship for Bob, got %d", len(friendships))
		}
	})

	t.Run("ListFriendships_Outgoing", func(t *testing.T) {
		friendships, err := store.ListFriendships(ctx, alice.ID, false, models.FriendshipAccepted)
		if err != nil {
			t.Fatalf("ListFriendships outgoing failed: %v", err)
		}
		if len(friendships) != 1 {
			t.Errorf("Expected 1 outgoing friendship for Alice, got %d", len(friendships))
		}
	})

	t.Run("GetFriends_ReturnsFriendUser", func(t *testing.T) {
		friends, err := store.GetFriends(ctx, alice.ID)
		if err != nil {
			t.Fatalf("GetFriends failed: %v", err)
		}
		if len(friends) != 1 || friends[0].ID != bob.ID {
			t.Errorf("Expected Bob as Alice's friend, got %v", friends)
		}
	})

	t.Run("DeleteFriendship_RemovesFriendship", func(t *testing.T) {
		friendships, err := store.ListFriendships(ctx, alice.ID, false, models.FriendshipAccepted)
		if err != nil || len(friendships) == 0 {
			t.Fatalf("Could not get friendship to delete")
		}
		if err := store.DeleteFriendship(ctx, friendships[0].ID); err != nil {
			t.Fatalf("DeleteFriendship failed: %v", err)
		}
		ok, err := store.AreFriends(ctx, alice.ID, bob.ID)
		if err != nil || ok {
			t.Errorf("Expected AreFriends=false after deletion, got %v (err: %v)", ok, err)
		}
	})
}

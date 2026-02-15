package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/middleware"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// testAuthInterceptor returns a Connect interceptor that sets a test user ID in the context.
func testAuthInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx = context.WithValue(ctx, middleware.UserIDKey, "Alice")
			return next(ctx, req)
		}
	}
}

// setupTestServer creates a test server with an in-memory SQLite database
func setupTestServer(t *testing.T) (protoconnect.SplitServiceClient, func()) {
	splitClient, _, cleanup := setupTestServerWithGroupService(t)
	return splitClient, cleanup
}

// setupTestServerWithGroupService creates a test server with both Split and Group services
func setupTestServerWithGroupService(t *testing.T) (protoconnect.SplitServiceClient, protoconnect.GroupServiceClient, func()) {
	t.Helper()

	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	store, err := sqlite.New(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create store: %v", err)
	}

	// Create services and handlers with test auth interceptor
	authInterceptor := connect.WithInterceptors(testAuthInterceptor())
	splitSvc := NewSplitService(store)
	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(splitSvc, authInterceptor)

	groupSvc := NewGroupService(store)
	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(groupSvc, authInterceptor)

	mux := http.NewServeMux()
	mux.Handle(splitPath, splitHandler)
	mux.Handle(groupPath, groupHandler)

	server := httptest.NewServer(mux)

	splitClient := protoconnect.NewSplitServiceClient(
		http.DefaultClient,
		server.URL,
	)

	groupClient := protoconnect.NewGroupServiceClient(
		http.DefaultClient,
		server.URL,
	)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return splitClient, groupClient, cleanup
}

func TestCalculateSplit_EqualSplit(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CalculateSplit(context.Background(), connect.NewRequest(&pb.CalculateSplitRequest{
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CalculateSplit failed: %v", err)
	}

	if len(resp.Msg.Splits) != 2 {
		t.Errorf("expected 2 splits, got %d", len(resp.Msg.Splits))
	}

	for _, name := range []string{"Alice", "Bob"} {
		split, ok := resp.Msg.Splits[name]
		if !ok {
			t.Errorf("missing split for %s", name)
			continue
		}
		if split.Total != 50 {
			t.Errorf("expected %s total to be 50, got %f", name, split.Total)
		}
	}
}

func TestCalculateSplit_WithItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CalculateSplit(context.Background(), connect.NewRequest(&pb.CalculateSplitRequest{
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice"}},
			{Description: "Salad", Amount: 10, ParticipantIds: []string{"Bob"}},
		},
		Total:        33, // $3 tax
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CalculateSplit failed: %v", err)
	}

	// Alice: $20 subtotal, $2 tax (20/30 * 3), $22 total
	// Bob: $10 subtotal, $1 tax (10/30 * 3), $11 total
	alice := resp.Msg.Splits["Alice"]
	if alice.Subtotal != 20 {
		t.Errorf("Alice subtotal: expected 20, got %f", alice.Subtotal)
	}
	if alice.Tax != 2 {
		t.Errorf("Alice tax: expected 2, got %f", alice.Tax)
	}
	if alice.Total != 22 {
		t.Errorf("Alice total: expected 22, got %f", alice.Total)
	}
	// Verify itemized breakdown
	if len(alice.Items) != 1 {
		t.Errorf("Alice items: expected 1, got %d", len(alice.Items))
	} else {
		if alice.Items[0].Description != "Pizza" {
			t.Errorf("Alice item description: expected 'Pizza', got '%s'", alice.Items[0].Description)
		}
		if alice.Items[0].Amount != 20 {
			t.Errorf("Alice item amount: expected 20, got %f", alice.Items[0].Amount)
		}
	}

	bob := resp.Msg.Splits["Bob"]
	if bob.Subtotal != 10 {
		t.Errorf("Bob subtotal: expected 10, got %f", bob.Subtotal)
	}
	if bob.Tax != 1 {
		t.Errorf("Bob tax: expected 1, got %f", bob.Tax)
	}
	if bob.Total != 11 {
		t.Errorf("Bob total: expected 11, got %f", bob.Total)
	}
	// Verify itemized breakdown
	if len(bob.Items) != 1 {
		t.Errorf("Bob items: expected 1, got %d", len(bob.Items))
	} else {
		if bob.Items[0].Description != "Salad" {
			t.Errorf("Bob item description: expected 'Salad', got '%s'", bob.Items[0].Description)
		}
		if bob.Items[0].Amount != 10 {
			t.Errorf("Bob item amount: expected 10, got %f", bob.Items[0].Amount)
		}
	}

	if resp.Msg.TaxAmount != 3 {
		t.Errorf("TaxAmount: expected 3, got %f", resp.Msg.TaxAmount)
	}
}

func TestCalculateSplit_SharedItem(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CalculateSplit(context.Background(), connect.NewRequest(&pb.CalculateSplitRequest{
		Items: []*pb.Item{
			{Description: "Shared Pizza", Amount: 30, ParticipantIds: []string{"Alice", "Bob", "Charlie"}},
		},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob", "Charlie"},
	}))

	if err != nil {
		t.Fatalf("CalculateSplit failed: %v", err)
	}

	// Each person: $10 subtotal, $1 tax, $11 total
	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		split := resp.Msg.Splits[name]
		if split.Total != 11 {
			t.Errorf("%s total: expected 11, got %f", name, split.Total)
		}
	}
}

func TestCreateBill_And_GetBill(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a bill
	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "Dinner",
		Items: []*pb.Item{
			{Description: "Burger", Amount: 15, ParticipantIds: []string{"Alice"}},
			{Description: "Fries", Amount: 5, ParticipantIds: []string{"Alice", "Bob"}},
		},
		Total:        22,
		Subtotal:     20,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	if createResp.Msg.BillId == "" {
		t.Error("expected non-empty bill ID")
	}

	if createResp.Msg.Split == nil {
		t.Fatal("expected split in response")
	}

	// Retrieve the bill
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: createResp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	if getResp.Msg.Title != "Dinner" {
		t.Errorf("title: expected 'Dinner', got '%s'", getResp.Msg.Title)
	}

	if getResp.Msg.Total != 22 {
		t.Errorf("total: expected 22, got %f", getResp.Msg.Total)
	}

	if len(getResp.Msg.Items) != 2 {
		t.Errorf("items: expected 2, got %d", len(getResp.Msg.Items))
	}

	if len(getResp.Msg.ParticipantIds) != 2 {
		t.Errorf("participants: expected 2, got %d", len(getResp.Msg.ParticipantIds))
	}

	if getResp.Msg.Split == nil {
		t.Error("expected split in GetBill response")
	}
}

func TestGetBill_NotFound(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: "nonexistent-id",
	}))

	if err == nil {
		t.Error("expected error for nonexistent bill")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestCalculateSplit_NoParticipants(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.CalculateSplit(context.Background(), connect.NewRequest(&pb.CalculateSplitRequest{
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		ParticipantIds: []string{},
	}))

	if err == nil {
		t.Error("expected error for no participants")
	}
}

func TestUpdateBill(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// First create a bill
	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "Original Dinner",
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
		},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	billId := createResp.Msg.BillId

	// Update the bill
	updateResp, err := client.UpdateBill(context.Background(), connect.NewRequest(&pb.UpdateBillRequest{
		BillId: billId,
		Title:  "Updated Dinner",
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
			{Description: "Beer", Amount: 15, ParticipantIds: []string{"Bob"}},
		},
		Total:        44,
		Subtotal:     35,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("UpdateBill failed: %v", err)
	}

	if updateResp.Msg.BillId != billId {
		t.Errorf("expected bill ID %s, got %s", billId, updateResp.Msg.BillId)
	}

	if updateResp.Msg.Split == nil {
		t.Fatal("expected split in response")
	}

	// Verify splits are recalculated correctly
	// Alice: $10 (half of pizza), Bob: $10 (half of pizza) + $15 (beer) = $25
	// Tax ratio: Alice 10/35, Bob 25/35
	// Tax = $9 total. Alice tax = 9*10/35 ≈ 2.57, Bob tax = 9*25/35 ≈ 6.43
	aliceSplit := updateResp.Msg.Split.Splits["Alice"]
	if aliceSplit == nil {
		t.Fatal("expected Alice in splits")
	}
	if aliceSplit.Subtotal != 10 {
		t.Errorf("Alice subtotal: expected 10, got %f", aliceSplit.Subtotal)
	}

	bobSplit := updateResp.Msg.Split.Splits["Bob"]
	if bobSplit == nil {
		t.Fatal("expected Bob in splits")
	}
	if bobSplit.Subtotal != 25 {
		t.Errorf("Bob subtotal: expected 25, got %f", bobSplit.Subtotal)
	}

	// Retrieve and verify persisted changes
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: billId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	if getResp.Msg.Title != "Updated Dinner" {
		t.Errorf("title not updated: expected 'Updated Dinner', got '%s'", getResp.Msg.Title)
	}

	if getResp.Msg.Total != 44 {
		t.Errorf("total not updated: expected 44, got %f", getResp.Msg.Total)
	}

	if len(getResp.Msg.Items) != 2 {
		t.Errorf("items count: expected 2, got %d", len(getResp.Msg.Items))
	}
}

func TestUpdateBill_NotFound(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.UpdateBill(context.Background(), connect.NewRequest(&pb.UpdateBillRequest{
		BillId:       "nonexistent-id",
		Title:        "Test",
		Total:        10,
		Subtotal:     10,
		ParticipantIds: []string{"Alice"},
	}))

	if err == nil {
		t.Error("expected error for nonexistent bill")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestListBillsByGroup(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	// First, create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	// Create first bill in group
	bill1Resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Dinner 1",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
		GroupId:      &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	// Create second bill in same group
	bill2Resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Lunch",
		Items:        []*pb.Item{{Description: "Burgers", Amount: 20}},
		Total:        22,
		Subtotal:     20,
		ParticipantIds: []string{"Alice", "Bob"},
		GroupId:      &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

	// Create bill without group (should not appear in results)
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Individual Bill",
		Items:        []*pb.Item{{Description: "Coffee", Amount: 5}},
		Total:        5,
		Subtotal:     5,
		ParticipantIds: []string{"Alice"},
	}))
	if err != nil {
		t.Fatalf("CreateBill 3 failed: %v", err)
	}

	// List bills by group
	listResp, err := splitClient.ListBillsByGroup(context.Background(), connect.NewRequest(&pb.ListBillsByGroupRequest{
		GroupId: groupID,
	}))
	if err != nil {
		t.Fatalf("ListBillsByGroup failed: %v", err)
	}

	// Verify we got exactly 2 bills
	if len(listResp.Msg.Bills) != 2 {
		t.Fatalf("expected 2 bills, got %d", len(listResp.Msg.Bills))
	}

	// Verify bills are in the response (order is by created_at DESC)
	billIDs := map[string]bool{
		bill1Resp.Msg.BillId: false,
		bill2Resp.Msg.BillId: false,
	}

	for _, summary := range listResp.Msg.Bills {
		if _, exists := billIDs[summary.BillId]; exists {
			billIDs[summary.BillId] = true
		}

		// Verify summary fields
		if summary.Title == "" {
			t.Error("expected non-empty title")
		}
		if summary.Total <= 0 {
			t.Error("expected positive total")
		}
		if summary.ParticipantCount != 2 {
			t.Errorf("expected 2 participants, got %d", summary.ParticipantCount)
		}
		if summary.CreatedAt == 0 {
			t.Error("expected non-zero created_at")
		}
	}

	// Verify both bills were found
	for billID, found := range billIDs {
		if !found {
			t.Errorf("bill %s not found in list response", billID)
		}
	}
}

func TestListBillsByGroup_EmptyGroup(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	// Create an actual empty group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Empty Group",
		MemberIds: []string{"Alice"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// List bills for a group with no bills
	listResp, err := splitClient.ListBillsByGroup(context.Background(), connect.NewRequest(&pb.ListBillsByGroupRequest{
		GroupId: groupResp.Msg.Group.Id,
	}))
	if err != nil {
		t.Fatalf("ListBillsByGroup failed: %v", err)
	}

	// Should return empty list, not error
	if len(listResp.Msg.Bills) != 0 {
		t.Errorf("expected 0 bills, got %d", len(listResp.Msg.Bills))
	}
}

func TestCreateBill_AutoGenerateTitle_WithItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Create bill with empty title - should auto-generate from items and participants
	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "", // Empty title
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
		},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Retrieve the bill
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	// Title should be auto-generated as "Pizza - Alice, Bob"
	expectedTitle := "Pizza - Alice, Bob"
	if getResp.Msg.Title != expectedTitle {
		t.Errorf("expected title '%s', got '%s'", expectedTitle, getResp.Msg.Title)
	}
}

func TestCreateBill_AutoGenerateTitle_MultipleItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Create bill with multiple items
	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "", // Empty title
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
			{Description: "Beer", Amount: 15, ParticipantIds: []string{"Bob"}},
			{Description: "Salad", Amount: 10, ParticipantIds: []string{"Alice"}},
		},
		Total:        50,
		Subtotal:     45,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Retrieve the bill
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	// Title should be "Pizza, Beer, Salad - Alice, Bob"
	expectedTitle := "Pizza, Beer, Salad - Alice, Bob"
	if getResp.Msg.Title != expectedTitle {
		t.Errorf("expected title '%s', got '%s'", expectedTitle, getResp.Msg.Title)
	}
}

func TestCreateBill_AutoGenerateTitle_NoItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Create bill with no items
	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "", // Empty title
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		ParticipantIds: []string{"Alice", "Bob", "Charlie"},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Retrieve the bill
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	// Title should be "Split with Alice, Bob, Charlie"
	expectedTitle := "Split with Alice, Bob, Charlie"
	if getResp.Msg.Title != expectedTitle {
		t.Errorf("expected title '%s', got '%s'", expectedTitle, getResp.Msg.Title)
	}
}

func TestCreateBill_WithPayer(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	payerID := "Alice"
	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
		PayerId:      &payerID,
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Retrieve and verify payer is saved
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	if getResp.Msg.GetPayerId() != "Alice" {
		t.Errorf("expected payerId 'Alice', got '%s'", getResp.Msg.GetPayerId())
	}
}

func TestCreateBill_InvalidPayer(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Try to create bill with payer that's not a participant
	invalidPayer := "Charlie"
	_, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
		PayerId:      &invalidPayer,
	}))

	if err == nil {
		t.Error("expected error for invalid payer")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connectErr.Code())
	}
}

func TestUpdateBill_ChangePayer(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Create bill with Alice as payer
	alicePayer := "Alice"
	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
		PayerId:      &alicePayer,
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Update to change payer to Bob
	bobPayer := "Bob"
	_, err = client.UpdateBill(context.Background(), connect.NewRequest(&pb.UpdateBillRequest{
		BillId:       createResp.Msg.BillId,
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
		PayerId:      &bobPayer,
	}))

	if err != nil {
		t.Fatalf("UpdateBill failed: %v", err)
	}

	// Verify payer changed
	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: createResp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	if getResp.Msg.GetPayerId() != "Bob" {
		t.Errorf("expected payerId 'Bob', got '%s'", getResp.Msg.GetPayerId())
	}
}

func TestDeleteBill(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a bill
	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}}},
		Total:        33,
		Subtotal:     30,
		ParticipantIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	billID := createResp.Msg.BillId

	// Delete the bill
	_, err = client.DeleteBill(context.Background(), connect.NewRequest(&pb.DeleteBillRequest{
		BillId: billID,
	}))
	if err != nil {
		t.Fatalf("DeleteBill failed: %v", err)
	}

	// Verify bill is deleted
	_, err = client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: billID,
	}))
	if err == nil {
		t.Error("Expected error when getting deleted bill, got nil")
	}
}

func TestDeleteBill_NotFound(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Try to delete non-existent bill
	_, err := client.DeleteBill(context.Background(), connect.NewRequest(&pb.DeleteBillRequest{
		BillId: "nonexistent-id",
	}))
	if err == nil {
		t.Error("Expected error for nonexistent bill")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("Expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("Expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestCreateBill_AutoAddsParticipantsToGroup(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	// Create a group with 2 members
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Auto-Add Test Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	// Create a bill with a new participant "Charlie" not in the group
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:          "Dinner",
		Items:          []*pb.Item{{Description: "Pizza", Amount: 30, ParticipantIds: []string{"Alice", "Bob", "Charlie"}}},
		Total:          33,
		Subtotal:       30,
		ParticipantIds: []string{"Alice", "Bob", "Charlie"},
		GroupId:        &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Verify Charlie was auto-added to the group
	getResp, err := groupClient.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: groupID,
	}))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	members := getResp.Msg.Group.MemberIds
	if len(members) != 3 {
		t.Fatalf("Expected 3 members after auto-add, got %d: %v", len(members), members)
	}

	// Verify Charlie is in the members list
	found := false
	for _, m := range members {
		if m == "Charlie" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Charlie not found in group members: %v", members)
	}
}

func TestCreateBill_AutoAddsPayerToGroup(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	// Create a group with 2 members
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Payer Auto-Add Test",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	// Create a bill where Alice pays but Charlie is also a participant (and payer is Alice, already in group)
	// The real test: payer "Diana" who is NOT a participant and NOT a group member
	// Actually, payer must be a participant per validation - so test that a new participant who is also the payer gets added
	payerID := "Charlie"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:          "Dinner",
		Items:          []*pb.Item{},
		Total:          100,
		Subtotal:       100,
		ParticipantIds: []string{"Alice", "Bob", "Charlie"},
		GroupId:        &groupID,
		PayerId:        &payerID,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Verify Charlie (the payer and participant) was auto-added to the group
	getResp, err := groupClient.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: groupID,
	}))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	members := getResp.Msg.Group.MemberIds
	if len(members) != 3 {
		t.Fatalf("Expected 3 members after auto-add, got %d: %v", len(members), members)
	}

	found := false
	for _, m := range members {
		if m == "Charlie" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Charlie (payer) not found in group members: %v", members)
	}
}

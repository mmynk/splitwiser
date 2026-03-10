package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/middleware"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

const testUserID = "test-user-uuid-alice"

// testAuthInterceptor returns a Connect interceptor that sets a test user UUID in the context.
func testAuthInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx = context.WithValue(ctx, middleware.UserIDKey, testUserID)
			return next(ctx, req)
		}
	}
}

// aliceBP returns a BillParticipant for the test auth user (Alice).
func aliceBP() *pb.BillParticipant {
	uid := testUserID
	return &pb.BillParticipant{DisplayName: "Alice", UserId: &uid}
}

// guestBP returns a BillParticipant for a guest participant (no user_id).
func guestBP(name string) *pb.BillParticipant {
	return &pb.BillParticipant{DisplayName: name}
}

// setupTestServerWithGroupService creates a test server with both Split and Group services.
// It also creates the test user (Alice) in the DB so resolveDisplayName works.
func setupTestServerWithGroupService(t *testing.T) (protoconnect.SplitServiceClient, protoconnect.GroupServiceClient, func()) {
	t.Helper()

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

	// Create the test user so resolveDisplayName returns "Alice"
	if err := store.CreateUser(context.Background(), &models.User{
		ID:           testUserID,
		Email:        "alice@test.com",
		DisplayName:  "Alice",
		PasswordHash: "testhash",
		CreatedAt:    1000,
		UpdatedAt:    1000,
	}); err != nil {
		store.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create test user: %v", err)
	}

	authInterceptor := connect.WithInterceptors(testAuthInterceptor())
	splitSvc := NewSplitService(store)
	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(splitSvc, authInterceptor)

	groupSvc := NewGroupService(store)
	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(groupSvc, authInterceptor)

	mux := http.NewServeMux()
	mux.Handle(splitPath, splitHandler)
	mux.Handle(groupPath, groupHandler)

	server := httptest.NewServer(mux)

	splitClient := protoconnect.NewSplitServiceClient(http.DefaultClient, server.URL)
	groupClient := protoconnect.NewGroupServiceClient(http.DefaultClient, server.URL)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return splitClient, groupClient, cleanup
}

// setupTestServer creates a split-only test server.
func setupTestServer(t *testing.T) (protoconnect.SplitServiceClient, func()) {
	splitClient, _, cleanup := setupTestServerWithGroupService(t)
	return splitClient, cleanup
}

func TestCalculateSplit_EqualSplit(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CalculateSplit(context.Background(), connect.NewRequest(&pb.CalculateSplitRequest{
		Items:          []*pb.Item{},
		Total:          100,
		Subtotal:       100,
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
		Total:          33,
		Subtotal:       30,
		ParticipantIds: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CalculateSplit failed: %v", err)
	}

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
	if len(alice.Items) != 1 {
		t.Errorf("Alice items: expected 1, got %d", len(alice.Items))
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
		Total:          33,
		Subtotal:       30,
		ParticipantIds: []string{"Alice", "Bob", "Charlie"},
	}))

	if err != nil {
		t.Fatalf("CalculateSplit failed: %v", err)
	}

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

	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "Dinner",
		Items: []*pb.Item{
			{Description: "Burger", Amount: 15, ParticipantIds: []string{"Alice"}},
			{Description: "Fries", Amount: 5, ParticipantIds: []string{"Alice", "Bob"}},
		},
		Total:        22,
		Subtotal:     20,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
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

	if len(getResp.Msg.Participants) != 2 {
		t.Errorf("participants: expected 2, got %d", len(getResp.Msg.Participants))
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
		Items:          []*pb.Item{},
		Total:          100,
		Subtotal:       100,
		ParticipantIds: []string{},
	}))

	if err == nil {
		t.Error("expected error for no participants")
	}
}

func TestUpdateBill(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "Original Dinner",
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
		},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	billId := createResp.Msg.BillId

	updateResp, err := client.UpdateBill(context.Background(), connect.NewRequest(&pb.UpdateBillRequest{
		BillId: billId,
		Title:  "Updated Dinner",
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
			{Description: "Beer", Amount: 15, ParticipantIds: []string{"Bob"}},
		},
		Total:        44,
		Subtotal:     35,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
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
		Participants: []*pb.BillParticipant{aliceBP()},
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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: []*pb.GroupMember{{DisplayName: "Bob"}},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	bill1Resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Dinner 1",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	bill2Resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Lunch",
		Items:        []*pb.Item{{Description: "Burgers", Amount: 20}},
		Total:        22,
		Subtotal:     20,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Individual Bill",
		Items:        []*pb.Item{{Description: "Coffee", Amount: 5}},
		Total:        5,
		Subtotal:     5,
		Participants: []*pb.BillParticipant{aliceBP()},
	}))
	if err != nil {
		t.Fatalf("CreateBill 3 failed: %v", err)
	}

	listResp, err := splitClient.ListBillsByGroup(context.Background(), connect.NewRequest(&pb.ListBillsByGroupRequest{
		GroupId: groupID,
	}))
	if err != nil {
		t.Fatalf("ListBillsByGroup failed: %v", err)
	}

	if len(listResp.Msg.Bills) != 2 {
		t.Fatalf("expected 2 bills, got %d", len(listResp.Msg.Bills))
	}

	billIDs := map[string]bool{
		bill1Resp.Msg.BillId: false,
		bill2Resp.Msg.BillId: false,
	}

	for _, summary := range listResp.Msg.Bills {
		if _, exists := billIDs[summary.BillId]; exists {
			billIDs[summary.BillId] = true
		}

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

	for billID, found := range billIDs {
		if !found {
			t.Errorf("bill %s not found in list response", billID)
		}
	}
}

func TestListBillsByGroup_EmptyGroup(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Empty Group",
		Members: []*pb.GroupMember{},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	listResp, err := splitClient.ListBillsByGroup(context.Background(), connect.NewRequest(&pb.ListBillsByGroupRequest{
		GroupId: groupResp.Msg.Group.Id,
	}))
	if err != nil {
		t.Fatalf("ListBillsByGroup failed: %v", err)
	}

	if len(listResp.Msg.Bills) != 0 {
		t.Errorf("expected 0 bills, got %d", len(listResp.Msg.Bills))
	}
}

func TestCreateBill_AutoGenerateTitle_WithItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "",
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
		},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	expectedTitle := "Pizza - Alice, Bob"
	if getResp.Msg.Title != expectedTitle {
		t.Errorf("expected title '%s', got '%s'", expectedTitle, getResp.Msg.Title)
	}
}

func TestCreateBill_AutoGenerateTitle_MultipleItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title: "",
		Items: []*pb.Item{
			{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}},
			{Description: "Beer", Amount: 15, ParticipantIds: []string{"Bob"}},
			{Description: "Salad", Amount: 10, ParticipantIds: []string{"Alice"}},
		},
		Total:        50,
		Subtotal:     45,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	expectedTitle := "Pizza, Beer, Salad - Alice, Bob"
	if getResp.Msg.Title != expectedTitle {
		t.Errorf("expected title '%s', got '%s'", expectedTitle, getResp.Msg.Title)
	}
}

func TestCreateBill_AutoGenerateTitle_NoItems(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "",
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob"), guestBP("Charlie")},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

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
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		PayerId:      &payerID,
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

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

	invalidPayer := "Charlie"
	_, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
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

	alicePayer := "Alice"
	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		PayerId:      &alicePayer,
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	bobPayer := "Bob"
	_, err = client.UpdateBill(context.Background(), connect.NewRequest(&pb.UpdateBillRequest{
		BillId:       createResp.Msg.BillId,
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30}},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		PayerId:      &bobPayer,
	}))

	if err != nil {
		t.Fatalf("UpdateBill failed: %v", err)
	}

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

	createResp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}}},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	billID := createResp.Msg.BillId

	_, err = client.DeleteBill(context.Background(), connect.NewRequest(&pb.DeleteBillRequest{
		BillId: billID,
	}))
	if err != nil {
		t.Fatalf("DeleteBill failed: %v", err)
	}

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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Auto-Add Test Group",
		Members: []*pb.GroupMember{{DisplayName: "Bob"}},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	// Alice (creator) + Bob = 2 members already. Charlie is new.
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 30, ParticipantIds: []string{"Alice", "Bob", "Charlie"}}},
		Total:        33,
		Subtotal:     30,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob"), guestBP("Charlie")},
		GroupId:      &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	getResp, err := groupClient.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: groupID,
	}))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	members := getResp.Msg.Group.Members
	// Expect: Alice (creator, UUID) + Bob + Charlie = 3
	if len(members) != 3 {
		t.Fatalf("Expected 3 members after auto-add, got %d", len(members))
	}

	found := false
	for _, m := range members {
		if m.DisplayName == "Charlie" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Charlie not found in group members")
	}
}

func TestCreateBill_AutoAddsPayerToGroup(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Payer Auto-Add Test",
		Members: []*pb.GroupMember{{DisplayName: "Bob"}},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	payerID := "Charlie"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob"), guestBP("Charlie")},
		GroupId:      &groupID,
		PayerId:      &payerID,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	getResp, err := groupClient.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: groupID,
	}))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	members := getResp.Msg.Group.Members
	if len(members) != 3 {
		t.Fatalf("Expected 3 members after auto-add, got %d", len(members))
	}

	found := false
	for _, m := range members {
		if m.DisplayName == "Charlie" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Charlie (payer) not found in group members")
	}
}

func TestListMyBills(t *testing.T) {
	splitClient, groupClient, cleanup := setupTestServerWithGroupService(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "My Bills Test Group",
		Members: []*pb.GroupMember{{DisplayName: "Bob"}},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupID := groupResp.Msg.Group.Id

	bill1Resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Dinner",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Bob"}}},
		Total:        22,
		Subtotal:     20,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupID,
	}))
	if err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	bill2Resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Solo Coffee",
		Items:        []*pb.Item{{Description: "Coffee", Amount: 5, ParticipantIds: []string{"Alice"}}},
		Total:        5,
		Subtotal:     5,
		Participants: []*pb.BillParticipant{aliceBP()},
	}))
	if err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

	// Alice must be a participant (auth check) — so she appears in all 3
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Bob Bill",
		Items:        []*pb.Item{{Description: "Beer", Amount: 10, ParticipantIds: []string{"Bob"}}},
		Total:        10,
		Subtotal:     10,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
	}))
	if err != nil {
		t.Fatalf("CreateBill 3 failed: %v", err)
	}

	listResp, err := splitClient.ListMyBills(context.Background(), connect.NewRequest(&pb.ListMyBillsRequest{}))
	if err != nil {
		t.Fatalf("ListMyBills failed: %v", err)
	}

	if len(listResp.Msg.Bills) < 3 {
		t.Fatalf("expected at least 3 bills, got %d", len(listResp.Msg.Bills))
	}

	var foundGroupBill, foundSoloBill bool
	for _, summary := range listResp.Msg.Bills {
		if summary.BillId == bill1Resp.Msg.BillId {
			foundGroupBill = true
			if summary.GroupName == nil || *summary.GroupName != "My Bills Test Group" {
				t.Errorf("expected group_name 'My Bills Test Group', got '%v'", summary.GroupName)
			}
		}
		if summary.BillId == bill2Resp.Msg.BillId {
			foundSoloBill = true
			if summary.GroupName != nil {
				t.Errorf("expected standalone bill to have no group_name, got '%s'", *summary.GroupName)
			}
		}
	}

	if !foundGroupBill {
		t.Errorf("group bill %s not found in ListMyBills", bill1Resp.Msg.BillId)
	}
	if !foundSoloBill {
		t.Errorf("standalone bill %s not found in ListMyBills", bill2Resp.Msg.BillId)
	}
}

func TestSearchUsers(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Search for "ali" (should match Alice); use IncludeNonFriends since Alice is the caller,
	// not her own friend, and the default mode filters to friends only.
	resp, err := client.SearchUsers(context.Background(), connect.NewRequest(&pb.SearchUsersRequest{
		Query:             "ali",
		IncludeNonFriends: true,
	}))
	if err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}

	if len(resp.Msg.Users) != 1 {
		t.Errorf("expected 1 result for 'ali', got %d", len(resp.Msg.Users))
	}
	if len(resp.Msg.Users) > 0 {
		if resp.Msg.Users[0].DisplayName != "Alice" {
			t.Errorf("expected 'Alice', got '%s'", resp.Msg.Users[0].DisplayName)
		}
		if resp.Msg.Users[0].UserId != testUserID {
			t.Errorf("expected UUID '%s', got '%s'", testUserID, resp.Msg.Users[0].UserId)
		}
	}
}

func TestSearchUsers_TooShort(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Single char query → empty result (min 2 chars)
	resp, err := client.SearchUsers(context.Background(), connect.NewRequest(&pb.SearchUsersRequest{
		Query: "a",
	}))
	if err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}

	if len(resp.Msg.Users) != 0 {
		t.Errorf("expected 0 results for single char query, got %d", len(resp.Msg.Users))
	}
}

func TestCreateBill_GuestParticipant(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	// Bill with both a registered (Alice with UUID) and a guest participant
	resp, err := client.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Mixed Bill",
		Items:        []*pb.Item{{Description: "Pizza", Amount: 20, ParticipantIds: []string{"Alice", "Guest"}}},
		Total:        22,
		Subtotal:     20,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Guest")},
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	getResp, err := client.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: resp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	if len(getResp.Msg.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(getResp.Msg.Participants))
	}

	// Find guest participant - should have no user_id
	var foundGuest bool
	for _, p := range getResp.Msg.Participants {
		if p.DisplayName == "Guest" {
			foundGuest = true
			if p.GetUserId() != "" {
				t.Errorf("guest participant should have no user_id, got '%s'", p.GetUserId())
			}
		}
		if p.DisplayName == "Alice" {
			if p.GetUserId() != testUserID {
				t.Errorf("Alice should have user_id '%s', got '%s'", testUserID, p.GetUserId())
			}
		}
	}
	if !foundGuest {
		t.Error("guest participant not found in response")
	}
}

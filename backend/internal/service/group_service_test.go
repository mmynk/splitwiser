package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// setupGroupTestServer creates a test server with both SplitService and GroupService
func setupGroupTestServer(t *testing.T) (protoconnect.GroupServiceClient, protoconnect.SplitServiceClient, func()) {
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

	// Create services and handlers
	splitSvc := NewSplitService(store)
	groupSvc := NewGroupService(store)

	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(splitSvc)
	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(groupSvc)

	mux := http.NewServeMux()
	mux.Handle(splitPath, splitHandler)
	mux.Handle(groupPath, groupHandler)

	server := httptest.NewServer(mux)

	groupClient := protoconnect.NewGroupServiceClient(
		http.DefaultClient,
		server.URL,
	)

	splitClient := protoconnect.NewSplitServiceClient(
		http.DefaultClient,
		server.URL,
	)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return groupClient, splitClient, cleanup
}

func TestCreateGroup(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	resp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Roommates",
		Members: []string{"Alice", "Bob", "Charlie"},
	}))

	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if resp.Msg.Group == nil {
		t.Fatal("expected group in response")
	}

	if resp.Msg.Group.Id == "" {
		t.Error("expected non-empty group ID")
	}

	if resp.Msg.Group.Name != "Roommates" {
		t.Errorf("name: expected 'Roommates', got '%s'", resp.Msg.Group.Name)
	}

	if len(resp.Msg.Group.Members) != 3 {
		t.Errorf("members: expected 3, got %d", len(resp.Msg.Group.Members))
	}

	if resp.Msg.Group.CreatedAt == 0 {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestGetGroup(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group first
	createResp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Work Lunch",
		Members: []string{"Diana", "Eve"},
	}))

	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	// Get the group
	getResp, err := client.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: createResp.Msg.Group.Id,
	}))

	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	if getResp.Msg.Group.Name != "Work Lunch" {
		t.Errorf("name: expected 'Work Lunch', got '%s'", getResp.Msg.Group.Name)
	}

	if len(getResp.Msg.Group.Members) != 2 {
		t.Errorf("members: expected 2, got %d", len(getResp.Msg.Group.Members))
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	_, err := client.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: "nonexistent-id",
	}))

	if err == nil {
		t.Error("expected error for nonexistent group")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestListGroups(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a few groups
	client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group A",
		Members: []string{"A1", "A2"},
	}))
	client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group B",
		Members: []string{"B1", "B2"},
	}))

	// List groups
	listResp, err := client.ListGroups(context.Background(), connect.NewRequest(&pb.ListGroupsRequest{}))

	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}

	if len(listResp.Msg.Groups) < 2 {
		t.Errorf("expected at least 2 groups, got %d", len(listResp.Msg.Groups))
	}

	// Verify groups have members
	for _, g := range listResp.Msg.Groups {
		if len(g.Members) == 0 {
			t.Errorf("group %s has no members", g.Name)
		}
	}
}

func TestListGroups_Empty(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	listResp, err := client.ListGroups(context.Background(), connect.NewRequest(&pb.ListGroupsRequest{}))

	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}

	if listResp.Msg.Groups == nil {
		// nil is acceptable for empty list
	} else if len(listResp.Msg.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(listResp.Msg.Groups))
	}
}

func TestUpdateGroup(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group first
	createResp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Original Name",
		Members: []string{"X", "Y"},
	}))

	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groupId := createResp.Msg.Group.Id

	// Update the group
	updateResp, err := client.UpdateGroup(context.Background(), connect.NewRequest(&pb.UpdateGroupRequest{
		GroupId: groupId,
		Name:    "Updated Name",
		Members: []string{"X", "Y", "Z"},
	}))

	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	if updateResp.Msg.Group.Name != "Updated Name" {
		t.Errorf("name not updated: expected 'Updated Name', got '%s'", updateResp.Msg.Group.Name)
	}

	if len(updateResp.Msg.Group.Members) != 3 {
		t.Errorf("members not updated: expected 3, got %d", len(updateResp.Msg.Group.Members))
	}

	// Verify by getting the group
	getResp, err := client.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: groupId,
	}))

	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	if getResp.Msg.Group.Name != "Updated Name" {
		t.Errorf("persisted name mismatch: expected 'Updated Name', got '%s'", getResp.Msg.Group.Name)
	}
}

func TestUpdateGroup_NotFound(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	_, err := client.UpdateGroup(context.Background(), connect.NewRequest(&pb.UpdateGroupRequest{
		GroupId: "nonexistent-id",
		Name:    "Test",
		Members: []string{"A"},
	}))

	if err == nil {
		t.Error("expected error for nonexistent group")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestDeleteGroup(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group first
	createResp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "To Be Deleted",
		Members: []string{"Delete", "Me"},
	}))

	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groupId := createResp.Msg.Group.Id

	// Delete the group
	_, err = client.DeleteGroup(context.Background(), connect.NewRequest(&pb.DeleteGroupRequest{
		GroupId: groupId,
	}))

	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	// Verify it's deleted
	_, err = client.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: groupId,
	}))

	if err == nil {
		t.Error("expected error getting deleted group")
	}
}

func TestDeleteGroup_NotFound(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	_, err := client.DeleteGroup(context.Background(), connect.NewRequest(&pb.DeleteGroupRequest{
		GroupId: "nonexistent-id",
	}))

	if err == nil {
		t.Error("expected error for nonexistent group")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestBillWithGroupId(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: []string{"Alice", "Bob"},
	}))

	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groupId := groupResp.Msg.Group.Id

	// Create a bill with group_id
	createResp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Dinner",
		Items:        []*pb.Item{},
		Total:        50,
		Subtotal:     50,
		Participants: []string{"Alice", "Bob"},
		GroupId:      &groupId,
	}))

	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Get the bill and verify group_id
	getResp, err := splitClient.GetBill(context.Background(), connect.NewRequest(&pb.GetBillRequest{
		BillId: createResp.Msg.BillId,
	}))

	if err != nil {
		t.Fatalf("GetBill failed: %v", err)
	}

	if getResp.Msg.GetGroupId() != groupId {
		t.Errorf("GroupId mismatch: expected %s, got %s", groupId, getResp.Msg.GetGroupId())
	}
}

func TestGetGroupBalances_SingleBill(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Create a bill where Alice paid $100
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		Participants: []string{"Alice", "Bob"},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Get balances
	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	// Verify balances
	if len(balResp.Msg.MemberBalances) != 2 {
		t.Fatalf("expected 2 member balances, got %d", len(balResp.Msg.MemberBalances))
	}

	// Alice paid $100, owes $50 → net = +$50
	// Bob paid $0, owes $50 → net = -$50
	var aliceBalance, bobBalance *pb.MemberBalance
	for _, bal := range balResp.Msg.MemberBalances {
		if bal.MemberName == "Alice" {
			aliceBalance = bal
		} else if bal.MemberName == "Bob" {
			bobBalance = bal
		}
	}

	if aliceBalance == nil {
		t.Fatal("Alice balance not found")
	}
	if aliceBalance.TotalPaid != 100 {
		t.Errorf("Alice total paid: expected 100, got %f", aliceBalance.TotalPaid)
	}
	if aliceBalance.TotalOwed != 50 {
		t.Errorf("Alice total owed: expected 50, got %f", aliceBalance.TotalOwed)
	}
	if aliceBalance.NetBalance != 50 {
		t.Errorf("Alice net balance: expected 50, got %f", aliceBalance.NetBalance)
	}

	if bobBalance == nil {
		t.Fatal("Bob balance not found")
	}
	if bobBalance.TotalPaid != 0 {
		t.Errorf("Bob total paid: expected 0, got %f", bobBalance.TotalPaid)
	}
	if bobBalance.TotalOwed != 50 {
		t.Errorf("Bob total owed: expected 50, got %f", bobBalance.TotalOwed)
	}
	if bobBalance.NetBalance != -50 {
		t.Errorf("Bob net balance: expected -50, got %f", bobBalance.NetBalance)
	}

	// Verify debt matrix - Bob owes Alice $50
	if len(balResp.Msg.DebtMatrix) != 1 {
		t.Fatalf("expected 1 debt edge, got %d", len(balResp.Msg.DebtMatrix))
	}
	debt := balResp.Msg.DebtMatrix[0]
	if debt.From != "Bob" || debt.To != "Alice" || debt.Amount != 50 {
		t.Errorf("debt: expected Bob→Alice $50, got %s→%s $%f", debt.From, debt.To, debt.Amount)
	}
}

func TestGetGroupBalances_MultipleBills(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group with 3 members
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: []string{"Alice", "Bob", "Charlie"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Bill 1: Alice paid $90 for all 3 people
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner 1",
		Total:        90,
		Subtotal:     90,
		Participants: []string{"Alice", "Bob", "Charlie"},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	// Bill 2: Bob paid $60 for all 3 people
	bobPayer := "Bob"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner 2",
		Total:        60,
		Subtotal:     60,
		Participants: []string{"Alice", "Bob", "Charlie"},
		GroupId:      &groupId,
		PayerId:      &bobPayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

	// Get balances
	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	// Alice: paid $90, owes $50 → net = +$40
	// Bob: paid $60, owes $50 → net = +$10
	// Charlie: paid $0, owes $50 → net = -$50
	var aliceBalance, bobBalance, charlieBalance *pb.MemberBalance
	for _, bal := range balResp.Msg.MemberBalances {
		switch bal.MemberName {
		case "Alice":
			aliceBalance = bal
		case "Bob":
			bobBalance = bal
		case "Charlie":
			charlieBalance = bal
		}
	}

	if aliceBalance == nil || bobBalance == nil || charlieBalance == nil {
		t.Fatal("missing member balances")
	}

	// Check net balances
	if aliceBalance.NetBalance != 40 {
		t.Errorf("Alice net: expected 40, got %f", aliceBalance.NetBalance)
	}
	if bobBalance.NetBalance != 10 {
		t.Errorf("Bob net: expected 10, got %f", bobBalance.NetBalance)
	}
	if charlieBalance.NetBalance != -50 {
		t.Errorf("Charlie net: expected -50, got %f", charlieBalance.NetBalance)
	}

	// Verify total debts add up (Charlie owes both Alice and Bob)
	if len(balResp.Msg.DebtMatrix) != 2 {
		t.Fatalf("expected 2 debt edges, got %d", len(balResp.Msg.DebtMatrix))
	}
}

func TestGetGroupBalances_NoBills(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group with no bills
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Empty Group",
		Members: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Get balances
	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	// Should return empty balances
	if len(balResp.Msg.MemberBalances) != 0 {
		t.Errorf("expected 0 balances, got %d", len(balResp.Msg.MemberBalances))
	}
	if len(balResp.Msg.DebtMatrix) != 0 {
		t.Errorf("expected 0 debts, got %d", len(balResp.Msg.DebtMatrix))
	}
}

func TestGetGroupBalances_BillsWithoutPayer(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Test Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Create a bill without payer
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:         "Dinner",
		Total:         100,
		Subtotal:      100,
		ParticipantIds: []string{"Alice", "Bob"},
		GroupId:       &groupId,
		// No PayerId
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Get balances - should skip bills without payer
	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	// Should return empty since bill has no payer
	if len(balResp.Msg.MemberBalances) != 0 {
		t.Errorf("expected 0 balances (no payer), got %d", len(balResp.Msg.MemberBalances))
	}
}

// Settlement Tests

func TestRecordSettlement(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Settlement Test Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Record a settlement: Bob pays Alice $30
	settlementResp, err := groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     30,
		Note:       "Venmo payment",
	}))
	if err != nil {
		t.Fatalf("RecordSettlement failed: %v", err)
	}

	if settlementResp.Msg.Settlement == nil {
		t.Fatal("expected settlement in response")
	}

	if settlementResp.Msg.Settlement.Id == "" {
		t.Error("expected non-empty settlement ID")
	}

	if settlementResp.Msg.Settlement.GroupId != groupId {
		t.Errorf("group_id: expected %s, got %s", groupId, settlementResp.Msg.Settlement.GroupId)
	}

	if settlementResp.Msg.Settlement.FromUserId != "Bob" {
		t.Errorf("from_user_id: expected 'Bob', got '%s'", settlementResp.Msg.Settlement.FromUserId)
	}

	if settlementResp.Msg.Settlement.ToUserId != "Alice" {
		t.Errorf("to_user_id: expected 'Alice', got '%s'", settlementResp.Msg.Settlement.ToUserId)
	}

	if settlementResp.Msg.Settlement.Amount != 30 {
		t.Errorf("amount: expected 30, got %f", settlementResp.Msg.Settlement.Amount)
	}

	if settlementResp.Msg.Settlement.Note != "Venmo payment" {
		t.Errorf("note: expected 'Venmo payment', got '%s'", settlementResp.Msg.Settlement.Note)
	}

	if settlementResp.Msg.Settlement.CreatedAt == 0 {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestRecordSettlement_Validation(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Validation Test Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Test: amount must be positive
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     0,
	}))
	if err == nil {
		t.Error("expected error for zero amount")
	}

	// Test: from and to must be different
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Alice",
		ToUserId:   "Alice",
		Amount:     10,
	}))
	if err == nil {
		t.Error("expected error when from_user_id == to_user_id")
	}

	// Test: users must be group members
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Charlie",
		ToUserId:   "Alice",
		Amount:     10,
	}))
	if err == nil {
		t.Error("expected error when from_user is not a group member")
	}
}

func TestListSettlements(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "List Settlements Group",
		MemberIds: []string{"Alice", "Bob", "Charlie"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Record multiple settlements
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     25,
	}))
	if err != nil {
		t.Fatalf("RecordSettlement 1 failed: %v", err)
	}

	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Charlie",
		ToUserId:   "Alice",
		Amount:     15,
	}))
	if err != nil {
		t.Fatalf("RecordSettlement 2 failed: %v", err)
	}

	// List settlements
	listResp, err := groupClient.ListSettlements(context.Background(), connect.NewRequest(&pb.ListSettlementsRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("ListSettlements failed: %v", err)
	}

	if len(listResp.Msg.Settlements) != 2 {
		t.Errorf("expected 2 settlements, got %d", len(listResp.Msg.Settlements))
	}
}

func TestDeleteSettlement(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Delete Settlement Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Record a settlement
	settlementResp, err := groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     50,
	}))
	if err != nil {
		t.Fatalf("RecordSettlement failed: %v", err)
	}

	settlementId := settlementResp.Msg.Settlement.Id

	// Delete the settlement
	_, err = groupClient.DeleteSettlement(context.Background(), connect.NewRequest(&pb.DeleteSettlementRequest{
		SettlementId: settlementId,
	}))
	if err != nil {
		t.Fatalf("DeleteSettlement failed: %v", err)
	}

	// Verify it's deleted
	listResp, err := groupClient.ListSettlements(context.Background(), connect.NewRequest(&pb.ListSettlementsRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("ListSettlements failed: %v", err)
	}

	if len(listResp.Msg.Settlements) != 0 {
		t.Errorf("expected 0 settlements after delete, got %d", len(listResp.Msg.Settlements))
	}
}

func TestGetGroupBalances_WithSettlements(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create a group
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:      "Balances With Settlements Group",
		MemberIds: []string{"Alice", "Bob"},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Create a bill where Alice paid $100, split equally
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:          "Dinner",
		Total:          100,
		Subtotal:       100,
		ParticipantIds: []string{"Alice", "Bob"},
		GroupId:        &groupId,
		PayerId:        &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Before settlement: Bob owes Alice $50
	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	// Find Bob's balance
	var bobBalanceBefore float64
	for _, bal := range balResp.Msg.MemberBalances {
		if bal.DisplayName == "Bob" {
			bobBalanceBefore = bal.NetBalance
		}
	}
	if bobBalanceBefore != -50 {
		t.Errorf("Bob's balance before settlement: expected -50, got %f", bobBalanceBefore)
	}

	// Record settlement: Bob pays Alice $30
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     30,
	}))
	if err != nil {
		t.Fatalf("RecordSettlement failed: %v", err)
	}

	// After settlement: Bob owes Alice $20 (50 - 30)
	balResp, err = groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	// Find Bob's balance after settlement
	var bobBalanceAfter float64
	for _, bal := range balResp.Msg.MemberBalances {
		if bal.DisplayName == "Bob" {
			bobBalanceAfter = bal.NetBalance
		}
	}

	// Bob's net balance should now be -20 (he still owes $20)
	// Before: TotalPaid=0, TotalOwed=50, Net=-50
	// After settlement: TotalPaid=30, TotalOwed=50, Net=-20
	if bobBalanceAfter != -20 {
		t.Errorf("Bob's balance after settlement: expected -20, got %f", bobBalanceAfter)
	}

	// Verify debt matrix shows $20 remaining
	if len(balResp.Msg.DebtMatrix) != 1 {
		t.Fatalf("expected 1 debt edge, got %d", len(balResp.Msg.DebtMatrix))
	}
	debt := balResp.Msg.DebtMatrix[0]
	if debt.FromUserId != "Bob" || debt.ToUserId != "Alice" || debt.Amount != 20 {
		t.Errorf("debt: expected Bob→Alice $20, got %s→%s $%f", debt.FromUserId, debt.ToUserId, debt.Amount)
	}
}

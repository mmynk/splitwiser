package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// setupGroupTestServer creates a test server with both SplitService and GroupService.
// It also creates the Alice user in the DB so resolveDisplayName returns "Alice".
func setupGroupTestServer(t *testing.T) (protoconnect.GroupServiceClient, protoconnect.SplitServiceClient, func()) {
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

	// Create the test user so resolveDisplayName returns "Alice" (needed for settlement validation)
	if err := store.CreateUser(context.Background(), &models.User{
		ID:           testUserID,
		Email:        "alice@example.com",
		DisplayName:  "Alice",
		PasswordHash: "hash",
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}); err != nil {
		store.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create test user: %v", err)
	}

	authInterceptor := connect.WithInterceptors(testAuthInterceptor())
	splitSvc := NewSplitService(store)
	groupSvc := NewGroupService(store)

	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(splitSvc, authInterceptor)
	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(groupSvc, authInterceptor)

	mux := http.NewServeMux()
	mux.Handle(splitPath, splitHandler)
	mux.Handle(groupPath, groupHandler)

	server := httptest.NewServer(mux)

	groupClient := protoconnect.NewGroupServiceClient(http.DefaultClient, server.URL)
	splitClient := protoconnect.NewSplitServiceClient(http.DefaultClient, server.URL)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return groupClient, splitClient, cleanup
}

// gm returns a slice of GroupMember protos with no user_id (guest members).
func gm(names ...string) []*pb.GroupMember {
	members := make([]*pb.GroupMember, len(names))
	for i, n := range names {
		members[i] = &pb.GroupMember{DisplayName: n}
	}
	return members
}

func TestCreateGroup(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	resp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Roommates",
		Members: gm("Alice", "Bob", "Charlie"),
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

	// Alice is already in the list, so creator is not re-added → 3 members
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

	createResp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Work Lunch",
		Members: gm("Alice", "Diana"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	getResp, err := client.GetGroup(context.Background(), connect.NewRequest(&pb.GetGroupRequest{
		GroupId: createResp.Msg.Group.Id,
	}))
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	if getResp.Msg.Group.Name != "Work Lunch" {
		t.Errorf("name: expected 'Work Lunch', got '%s'", getResp.Msg.Group.Name)
	}

	// Alice already in list → 2 members
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

	// Alice is auto-added as creator to both groups (not in A1/A2 or B1/B2 lists)
	client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group A",
		Members: gm("A1", "A2"),
	}))
	client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group B",
		Members: gm("B1", "B2"),
	}))

	listResp, err := client.ListGroups(context.Background(), connect.NewRequest(&pb.ListGroupsRequest{}))
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}

	if len(listResp.Msg.Groups) < 2 {
		t.Errorf("expected at least 2 groups, got %d", len(listResp.Msg.Groups))
	}

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

	if listResp.Msg.Groups != nil && len(listResp.Msg.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(listResp.Msg.Groups))
	}
}

func TestUpdateGroup(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	createResp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Original Name",
		Members: gm("X", "Y"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groupId := createResp.Msg.Group.Id

	// UpdateGroup does not auto-add creator — exactly 3 members
	updateResp, err := client.UpdateGroup(context.Background(), connect.NewRequest(&pb.UpdateGroupRequest{
		GroupId: groupId,
		Name:    "Updated Name",
		Members: gm("X", "Y", "Z"),
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
		Members: gm("A"),
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

	createResp, err := client.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "To Be Deleted",
		Members: gm("Delete", "Me"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groupId := createResp.Msg.Group.Id

	_, err = client.DeleteGroup(context.Background(), connect.NewRequest(&pb.DeleteGroupRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	groupId := groupResp.Msg.Group.Id

	createResp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Group Dinner",
		Items:        []*pb.Item{},
		Total:        50,
		Subtotal:     50,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupId,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Alice paid $100 for Alice and Bob
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	if len(balResp.Msg.MemberBalances) != 2 {
		t.Fatalf("expected 2 member balances, got %d", len(balResp.Msg.MemberBalances))
	}

	var aliceBalance, bobBalance *pb.MemberBalance
	for _, bal := range balResp.Msg.MemberBalances {
		if bal.DisplayName == "Alice" {
			aliceBalance = bal
		} else if bal.DisplayName == "Bob" {
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

	if len(balResp.Msg.DebtMatrix) != 1 {
		t.Fatalf("expected 1 debt edge, got %d", len(balResp.Msg.DebtMatrix))
	}
	debt := balResp.Msg.DebtMatrix[0]
	if debt.FromUserId != "Bob" || debt.ToUserId != "Alice" || debt.Amount != 50 {
		t.Errorf("debt: expected Bob→Alice $50, got %s→%s $%f", debt.FromUserId, debt.ToUserId, debt.Amount)
	}
}

func TestGetGroupBalances_MultipleBills(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: gm("Alice", "Bob", "Charlie"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Bill 1: Alice paid $90 for all 3
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner 1",
		Total:        90,
		Subtotal:     90,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob"), guestBP("Charlie")},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	// Bill 2: Bob paid $60 for all 3
	bobPayer := "Bob"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner 2",
		Total:        60,
		Subtotal:     60,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob"), guestBP("Charlie")},
		GroupId:      &groupId,
		PayerId:      &bobPayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

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
		switch bal.DisplayName {
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

	if aliceBalance.NetBalance != 40 {
		t.Errorf("Alice net: expected 40, got %f", aliceBalance.NetBalance)
	}
	if bobBalance.NetBalance != 10 {
		t.Errorf("Bob net: expected 10, got %f", bobBalance.NetBalance)
	}
	if charlieBalance.NetBalance != -50 {
		t.Errorf("Charlie net: expected -50, got %f", charlieBalance.NetBalance)
	}

	if len(balResp.Msg.DebtMatrix) != 2 {
		t.Fatalf("expected 2 debt edges, got %d", len(balResp.Msg.DebtMatrix))
	}
}

func TestGetGroupBalances_NoBills(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Empty Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupId,
		// No PayerId
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	balResp, err := groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	if len(balResp.Msg.MemberBalances) != 0 {
		t.Errorf("expected 0 balances (no payer), got %d", len(balResp.Msg.MemberBalances))
	}
}

// Settlement Tests

func TestRecordSettlement(t *testing.T) {
	groupClient, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Settlement Test Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Bob pays Alice $30
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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Validation Test Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// amount must be positive
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     0,
	}))
	if err == nil {
		t.Error("expected error for zero amount")
	}

	// from and to must differ
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Alice",
		ToUserId:   "Alice",
		Amount:     10,
	}))
	if err == nil {
		t.Error("expected error when from_user_id == to_user_id")
	}

	// users must be group members
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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "List Settlements Group",
		Members: gm("Alice", "Bob", "Charlie"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Delete Settlement Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

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

	_, err = groupClient.DeleteSettlement(context.Background(), connect.NewRequest(&pb.DeleteSettlementRequest{
		SettlementId: settlementId,
	}))
	if err != nil {
		t.Fatalf("DeleteSettlement failed: %v", err)
	}

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

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Balances With Settlements Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Alice paid $100, split equally with Bob
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
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

	var bobBalanceBefore float64
	for _, bal := range balResp.Msg.MemberBalances {
		if bal.DisplayName == "Bob" {
			bobBalanceBefore = bal.NetBalance
		}
	}
	if bobBalanceBefore != -50 {
		t.Errorf("Bob's balance before settlement: expected -50, got %f", bobBalanceBefore)
	}

	// Bob pays Alice $30
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     30,
	}))
	if err != nil {
		t.Fatalf("RecordSettlement failed: %v", err)
	}

	// After settlement: Bob owes Alice $20
	balResp, err = groupClient.GetGroupBalances(context.Background(), connect.NewRequest(&pb.GetGroupBalancesRequest{
		GroupId: groupId,
	}))
	if err != nil {
		t.Fatalf("GetGroupBalances failed: %v", err)
	}

	var bobBalanceAfter float64
	for _, bal := range balResp.Msg.MemberBalances {
		if bal.DisplayName == "Bob" {
			bobBalanceAfter = bal.NetBalance
		}
	}
	if bobBalanceAfter != -20 {
		t.Errorf("Bob's balance after settlement: expected -20, got %f", bobBalanceAfter)
	}

	if len(balResp.Msg.DebtMatrix) != 1 {
		t.Fatalf("expected 1 debt edge, got %d", len(balResp.Msg.DebtMatrix))
	}
	debt := balResp.Msg.DebtMatrix[0]
	if debt.FromUserId != "Bob" || debt.ToUserId != "Alice" || debt.Amount != 20 {
		t.Errorf("debt: expected Bob→Alice $20, got %s→%s $%f", debt.FromUserId, debt.ToUserId, debt.Amount)
	}
}

// GetMyBalances Tests

func TestGetMyBalances_NoGroups(t *testing.T) {
	client, _, cleanup := setupGroupTestServer(t)
	defer cleanup()

	resp, err := client.GetMyBalances(context.Background(), connect.NewRequest(&pb.GetMyBalancesRequest{}))
	if err != nil {
		t.Fatalf("GetMyBalances failed: %v", err)
	}

	if resp.Msg.TotalYouOwe != 0 {
		t.Errorf("total_you_owe: expected 0, got %f", resp.Msg.TotalYouOwe)
	}
	if resp.Msg.TotalOwedToYou != 0 {
		t.Errorf("total_owed_to_you: expected 0, got %f", resp.Msg.TotalOwedToYou)
	}
	if len(resp.Msg.PersonBalances) != 0 {
		t.Errorf("expected 0 person balances, got %d", len(resp.Msg.PersonBalances))
	}
}

func TestGetMyBalances_SingleGroup(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Create group with Alice (auth user) and Bob
	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Test Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Alice paid $100 for both
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	resp, err := groupClient.GetMyBalances(context.Background(), connect.NewRequest(&pb.GetMyBalancesRequest{}))
	if err != nil {
		t.Fatalf("GetMyBalances failed: %v", err)
	}

	// Bob owes Alice $50
	if resp.Msg.TotalOwedToYou != 50 {
		t.Errorf("total_owed_to_you: expected 50, got %f", resp.Msg.TotalOwedToYou)
	}
	if resp.Msg.TotalYouOwe != 0 {
		t.Errorf("total_you_owe: expected 0, got %f", resp.Msg.TotalYouOwe)
	}
	if len(resp.Msg.PersonBalances) != 1 {
		t.Fatalf("expected 1 person balance, got %d", len(resp.Msg.PersonBalances))
	}

	bob := resp.Msg.PersonBalances[0]
	if bob.DisplayName != "Bob" {
		t.Errorf("expected Bob, got %s", bob.DisplayName)
	}
	if bob.NetAmount != 50 {
		t.Errorf("Bob net_amount: expected 50 (owes Alice), got %f", bob.NetAmount)
	}
	if len(bob.GroupBalances) != 1 {
		t.Fatalf("expected 1 group balance for Bob, got %d", len(bob.GroupBalances))
	}
	if bob.GroupBalances[0].GroupId != groupId {
		t.Errorf("group_id mismatch")
	}
}

func TestGetMyBalances_MultipleGroups(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	// Group 1: Alice + Bob
	g1Resp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group 1",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup 1 failed: %v", err)
	}
	g1Id := g1Resp.Msg.Group.Id

	// Group 2: Alice + Bob + Charlie
	g2Resp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group 2",
		Members: gm("Alice", "Bob", "Charlie"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup 2 failed: %v", err)
	}
	g2Id := g2Resp.Msg.Group.Id

	// Group 1: Alice paid $60 for Alice + Bob → Bob owes $30
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Lunch",
		Total:        60,
		Subtotal:     60,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &g1Id,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill 1 failed: %v", err)
	}

	// Group 2: Bob paid $90 for all 3 → Alice owes $30, Charlie owes $30
	bobPayer := "Bob"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Total:        90,
		Subtotal:     90,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob"), guestBP("Charlie")},
		GroupId:      &g2Id,
		PayerId:      &bobPayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill 2 failed: %v", err)
	}

	resp, err := groupClient.GetMyBalances(context.Background(), connect.NewRequest(&pb.GetMyBalancesRequest{}))
	if err != nil {
		t.Fatalf("GetMyBalances failed: %v", err)
	}

	// Bob: group1 owes Alice $30, group2 Alice owes Bob $30 → net 0
	// Charlie: group2 Alice is owed $0 from Charlie (Charlie owes Bob, not Alice)
	// Actually, let me recalculate:
	// Group 1: Alice paid $60, split 2 ways. Bob owes Alice $30. Debt: Bob→Alice $30
	// Group 2: Bob paid $90, split 3 ways. Alice owes Bob $30, Charlie owes Bob $30. Debt: Alice→Bob $30, Charlie→Bob $30
	// For Alice: Bob owes me $30 (group1), I owe Bob $30 (group2). Net with Bob = 0.
	// Alice has no debt with Charlie in either group.

	// So net Bob balance for Alice: +30 - 30 = 0
	// total_you_owe = 30 (from group 2, Alice→Bob)
	// total_owed_to_you = 30 (from group 1, Bob→Alice)

	if resp.Msg.TotalYouOwe != 30 {
		t.Errorf("total_you_owe: expected 30, got %f", resp.Msg.TotalYouOwe)
	}
	if resp.Msg.TotalOwedToYou != 30 {
		t.Errorf("total_owed_to_you: expected 30, got %f", resp.Msg.TotalOwedToYou)
	}

	// Should have 1 person (Bob) with net 0 and 2 group_balances
	if len(resp.Msg.PersonBalances) != 1 {
		t.Fatalf("expected 1 person balance (Bob), got %d", len(resp.Msg.PersonBalances))
	}
	bob := resp.Msg.PersonBalances[0]
	if bob.DisplayName != "Bob" {
		t.Errorf("expected Bob, got %s", bob.DisplayName)
	}
	if bob.NetAmount != 0 {
		t.Errorf("Bob net: expected 0, got %f", bob.NetAmount)
	}
	if len(bob.GroupBalances) != 2 {
		t.Errorf("expected 2 group balances for Bob, got %d", len(bob.GroupBalances))
	}
}

func TestGetMyBalances_WithSettlements(t *testing.T) {
	groupClient, splitClient, cleanup := setupGroupTestServer(t)
	defer cleanup()

	groupResp, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Settlement Group",
		Members: gm("Alice", "Bob"),
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	groupId := groupResp.Msg.Group.Id

	// Alice paid $100 → Bob owes $50
	alicePayer := "Alice"
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:        "Dinner",
		Total:        100,
		Subtotal:     100,
		Participants: []*pb.BillParticipant{aliceBP(), guestBP("Bob")},
		GroupId:      &groupId,
		PayerId:      &alicePayer,
	}))
	if err != nil {
		t.Fatalf("CreateBill failed: %v", err)
	}

	// Bob settles $30
	_, err = groupClient.RecordSettlement(context.Background(), connect.NewRequest(&pb.RecordSettlementRequest{
		GroupId:    groupId,
		FromUserId: "Bob",
		ToUserId:   "Alice",
		Amount:     30,
	}))
	if err != nil {
		t.Fatalf("RecordSettlement failed: %v", err)
	}

	resp, err := groupClient.GetMyBalances(context.Background(), connect.NewRequest(&pb.GetMyBalancesRequest{}))
	if err != nil {
		t.Fatalf("GetMyBalances failed: %v", err)
	}

	// After settlement: Bob owes Alice $20
	if resp.Msg.TotalOwedToYou != 20 {
		t.Errorf("total_owed_to_you: expected 20, got %f", resp.Msg.TotalOwedToYou)
	}
	if resp.Msg.TotalYouOwe != 0 {
		t.Errorf("total_you_owe: expected 0, got %f", resp.Msg.TotalYouOwe)
	}
}

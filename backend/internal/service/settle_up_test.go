package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// setupSettleUpTest wraps setupTestServerWithFriendService and adds an accepted
// Alice–Bob friendship so group creation with Bob as a registered member succeeds.
func setupSettleUpTest(t *testing.T) (protoconnect.GroupServiceClient, *sqlite.SQLiteStore, func()) {
	t.Helper()
	_, groupClient, _, store, cleanup := setupTestServerWithFriendService(t)

	ctx := context.Background()
	friendship := &models.Friendship{
		ID:          "friendship-alice-bob",
		RequesterID: testUserID,
		AddresseeID: testBobID,
		Status:      models.FriendshipPending,
		CreatedAt:   1000,
		UpdatedAt:   1000,
	}
	if err := store.SendFriendRequest(ctx, friendship); err != nil {
		cleanup()
		t.Fatalf("failed to send friend request: %v", err)
	}
	if err := store.UpdateFriendshipStatus(ctx, friendship.ID, models.FriendshipAccepted); err != nil {
		cleanup()
		t.Fatalf("failed to accept friendship: %v", err)
	}

	return groupClient, store, cleanup
}

// bobMember returns a GroupMember proto for Bob (registered user).
func bobMember() *pb.GroupMember {
	return &pb.GroupMember{DisplayName: "Bob", UserId: strPtr(testBobID)}
}

func TestSettleUpWithPerson_TwoGroups(t *testing.T) {
	client, store, cleanup := setupSettleUpTest(t)
	defer cleanup()
	ctx := context.Background()

	g1Resp, err := client.CreateGroup(ctx, connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group 1",
		Members: []*pb.GroupMember{bobMember()},
	}))
	if err != nil {
		t.Fatalf("CreateGroup 1 failed: %v", err)
	}
	groupID1 := g1Resp.Msg.Group.Id

	g2Resp, err := client.CreateGroup(ctx, connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Group 2",
		Members: []*pb.GroupMember{bobMember()},
	}))
	if err != nil {
		t.Fatalf("CreateGroup 2 failed: %v", err)
	}
	groupID2 := g2Resp.Msg.Group.Id

	// Group 1: Alice pays $100 for both → Bob owes Alice $50.
	// Group 2: Bob pays $60 for both → Alice owes Bob $30.
	// SettleUpWithPerson must create one settlement per group.
	aliceBobIDs := []string{"Alice", "Bob"}
	if err := store.CreateBill(ctx, &models.Bill{
		Title:    "Dinner",
		Total:    100, Subtotal: 100,
		GroupID: groupID1, PayerID: "Alice",
		Participants: []models.BillParticipant{
			{DisplayName: "Alice", UserID: testUserID},
			{DisplayName: "Bob", UserID: testBobID},
		},
		Items: []models.Item{{Description: "Food", Amount: 100, Participants: aliceBobIDs}},
	}); err != nil {
		t.Fatalf("CreateBill group1 failed: %v", err)
	}
	if err := store.CreateBill(ctx, &models.Bill{
		Title:    "Lunch",
		Total:    60, Subtotal: 60,
		GroupID: groupID2, PayerID: "Bob",
		Participants: []models.BillParticipant{
			{DisplayName: "Alice", UserID: testUserID},
			{DisplayName: "Bob", UserID: testBobID},
		},
		Items: []models.Item{{Description: "Food", Amount: 60, Participants: aliceBobIDs}},
	}); err != nil {
		t.Fatalf("CreateBill group2 failed: %v", err)
	}

	settleResp, err := client.SettleUpWithPerson(ctx, connect.NewRequest(&pb.SettleUpWithPersonRequest{
		ToUserId: testBobID,
	}))
	if err != nil {
		t.Fatalf("SettleUpWithPerson failed: %v", err)
	}
	if len(settleResp.Msg.Settlements) != 2 {
		t.Errorf("expected 2 settlements, got %d", len(settleResp.Msg.Settlements))
	}
	for _, s := range settleResp.Msg.Settlements {
		if s.GroupId == nil {
			t.Errorf("expected group-scoped settlement, got nil group_id")
		}
	}

	balResp, err := client.GetMyBalances(ctx, connect.NewRequest(&pb.GetMyBalancesRequest{}))
	if err != nil {
		t.Fatalf("GetMyBalances failed: %v", err)
	}
	for _, p := range balResp.Msg.PersonBalances {
		if p.DisplayName == "Bob" && p.NetAmount != 0 {
			t.Errorf("expected zero net balance with Bob after settle up, got %.2f", p.NetAmount)
		}
	}
}

func TestSettleUpWithPerson_NoDebt(t *testing.T) {
	client, _, cleanup := setupSettleUpTest(t)
	defer cleanup()
	ctx := context.Background()

	_, err := client.CreateGroup(ctx, connect.NewRequest(&pb.CreateGroupRequest{
		Name:    "Empty Group",
		Members: []*pb.GroupMember{bobMember()},
	}))
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	_, err = client.SettleUpWithPerson(ctx, connect.NewRequest(&pb.SettleUpWithPersonRequest{
		ToUserId: testBobID,
	}))
	if err == nil {
		t.Fatal("expected error when no debt exists, got nil")
	}
}

func TestSettleUpWithPerson_Self(t *testing.T) {
	client, _, cleanup := setupSettleUpTest(t)
	defer cleanup()

	_, err := client.SettleUpWithPerson(context.Background(), connect.NewRequest(&pb.SettleUpWithPersonRequest{
		ToUserId: testUserID,
	}))
	if err == nil {
		t.Fatal("expected error settling up with self")
	}
}

func TestSettleUpWithPerson_UnknownUser(t *testing.T) {
	client, _, cleanup := setupSettleUpTest(t)
	defer cleanup()

	_, err := client.SettleUpWithPerson(context.Background(), connect.NewRequest(&pb.SettleUpWithPersonRequest{
		ToUserId: "nonexistent-user-id",
	}))
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}

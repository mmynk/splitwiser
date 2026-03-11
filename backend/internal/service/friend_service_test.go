package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage/sqlite"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

const testBobID = "test-user-uuid-bob"

// setupTestServerWithFriendService creates a test server with Split, Group, and Friend services.
// It creates Alice (auth user) and Bob as real DB users.
// The store is exposed to allow direct manipulation in tests (e.g. inserting pending friendships).
func setupTestServerWithFriendService(t *testing.T) (protoconnect.SplitServiceClient, protoconnect.GroupServiceClient, protoconnect.FriendServiceClient, *sqlite.SQLiteStore, func()) {
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

	// Create Alice (the auth user) and Bob
	for _, u := range []*models.User{
		{ID: testUserID, Email: "alice@test.com", DisplayName: "Alice", PasswordHash: "h", CreatedAt: 1, UpdatedAt: 1},
		{ID: testBobID, Email: "bob@test.com", DisplayName: "Bob", PasswordHash: "h", CreatedAt: 1, UpdatedAt: 1},
	} {
		if err := store.CreateUser(context.Background(), u); err != nil {
			store.Close()
			os.Remove(tmpFile.Name())
			t.Fatalf("failed to create user %s: %v", u.DisplayName, err)
		}
	}

	authInterceptor := connect.WithInterceptors(testAuthInterceptor())

	mux := http.NewServeMux()

	splitPath, splitHandler := protoconnect.NewSplitServiceHandler(NewSplitService(store), authInterceptor)
	mux.Handle(splitPath, splitHandler)

	groupPath, groupHandler := protoconnect.NewGroupServiceHandler(NewGroupService(store), authInterceptor)
	mux.Handle(groupPath, groupHandler)

	friendPath, friendHandler := protoconnect.NewFriendServiceHandler(NewFriendService(store), authInterceptor)
	mux.Handle(friendPath, friendHandler)

	server := httptest.NewServer(mux)

	splitClient := protoconnect.NewSplitServiceClient(http.DefaultClient, server.URL)
	groupClient := protoconnect.NewGroupServiceClient(http.DefaultClient, server.URL)
	friendClient := protoconnect.NewFriendServiceClient(http.DefaultClient, server.URL)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return splitClient, groupClient, friendClient, store, cleanup
}

func TestSendFriendRequest(t *testing.T) {
	_, _, friendClient, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	resp, err := friendClient.SendFriendRequest(context.Background(), connect.NewRequest(&pb.SendFriendRequestRequest{
		AddresseeId: testBobID,
	}))
	if err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}

	if resp.Msg.Request.RequesterId != testUserID {
		t.Errorf("Expected requester_id=%s, got %s", testUserID, resp.Msg.Request.RequesterId)
	}
	if resp.Msg.Request.AddresseeId != testBobID {
		t.Errorf("Expected addressee_id=%s, got %s", testBobID, resp.Msg.Request.AddresseeId)
	}
	if resp.Msg.Request.Status != "pending" {
		t.Errorf("Expected status=pending, got %s", resp.Msg.Request.Status)
	}
	if resp.Msg.Request.RequesterDisplayName != "Alice" {
		t.Errorf("Expected requester display name=Alice, got %s", resp.Msg.Request.RequesterDisplayName)
	}
}

func TestSendFriendRequest_ToSelf_Rejected(t *testing.T) {
	_, _, friendClient, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	_, err := friendClient.SendFriendRequest(context.Background(), connect.NewRequest(&pb.SendFriendRequestRequest{
		AddresseeId: testUserID, // Alice sending to herself
	}))
	if err == nil {
		t.Fatal("Expected error when sending friend request to self, got nil")
	}
}

func TestSendFriendRequest_Duplicate_Rejected(t *testing.T) {
	_, _, friendClient, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	req := connect.NewRequest(&pb.SendFriendRequestRequest{AddresseeId: testBobID})
	if _, err := friendClient.SendFriendRequest(context.Background(), req); err != nil {
		t.Fatalf("First SendFriendRequest failed: %v", err)
	}

	// Second request should fail
	_, err := friendClient.SendFriendRequest(context.Background(), connect.NewRequest(&pb.SendFriendRequestRequest{AddresseeId: testBobID}))
	if err == nil {
		t.Fatal("Expected error for duplicate friend request, got nil")
	}
}

// TestRespondToFriendRequest_NonAddresseeRejected verifies that the requester cannot
// accept their own outgoing request (only the addressee may respond).
func TestRespondToFriendRequest_NonAddresseeRejected(t *testing.T) {
	_, _, friendClient, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Alice sends request to Bob — Alice is the requester, not the addressee.
	resp, err := friendClient.SendFriendRequest(context.Background(), connect.NewRequest(&pb.SendFriendRequestRequest{
		AddresseeId: testBobID,
	}))
	if err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}
	requestID := resp.Msg.Request.Id

	// Alice tries to accept — she is the requester, not the addressee; should be denied.
	_, err = friendClient.RespondToFriendRequest(context.Background(), connect.NewRequest(&pb.RespondToFriendRequestRequest{
		RequestId: requestID,
		Accept:    true,
	}))
	if err == nil {
		t.Fatal("Expected permission denied when non-addressee tries to accept, got nil")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied, got %v", err)
	}
}

// TestRespondToFriendRequest_AcceptsSuccessfully verifies that the addressee can accept
// a pending request and that the friendship becomes active.
func TestRespondToFriendRequest_AcceptsSuccessfully(t *testing.T) {
	splitClient, _, friendClient, store, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Insert a Bob→Alice pending request directly (Bob is requester, Alice is addressee).
	// The test interceptor authenticates as Alice, so she can accept it via the API.
	f := &models.Friendship{
		RequesterID: testBobID,
		AddresseeID: testUserID,
		Status:      models.FriendshipPending,
	}
	if err := store.SendFriendRequest(context.Background(), f); err != nil {
		t.Fatalf("failed to insert pending friendship: %v", err)
	}

	// Alice (addressee) accepts.
	resp, err := friendClient.RespondToFriendRequest(context.Background(), connect.NewRequest(&pb.RespondToFriendRequestRequest{
		RequestId: f.ID,
		Accept:    true,
	}))
	if err != nil {
		t.Fatalf("RespondToFriendRequest failed: %v", err)
	}
	if resp.Msg.Request.Status != "accepted" {
		t.Errorf("Expected status=accepted, got %s", resp.Msg.Request.Status)
	}

	// Verify Bob is now addable to Alice's bill.
	_, err = splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:    "Dinner",
		Total:    100,
		Subtotal: 100,
		Participants: []*pb.BillParticipant{
			aliceBP(),
			{DisplayName: "Bob", UserId: func() *string { s := testBobID; return &s }()},
		},
	}))
	if err != nil {
		t.Errorf("Expected accepted friend to be addable to bill, got: %v", err)
	}
}

func TestListFriendRequests(t *testing.T) {
	_, _, friendClient, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Send a request
	if _, err := friendClient.SendFriendRequest(context.Background(), connect.NewRequest(&pb.SendFriendRequestRequest{
		AddresseeId: testBobID,
	})); err != nil {
		t.Fatalf("SendFriendRequest failed: %v", err)
	}

	// List outgoing requests (Alice as requester)
	listResp, err := friendClient.ListFriendRequests(context.Background(), connect.NewRequest(&pb.ListFriendRequestsRequest{
		Incoming: false,
	}))
	if err != nil {
		t.Fatalf("ListFriendRequests failed: %v", err)
	}
	if len(listResp.Msg.Requests) != 1 {
		t.Errorf("Expected 1 outgoing request, got %d", len(listResp.Msg.Requests))
	}

	// List incoming — Alice has no incoming requests
	listResp, err = friendClient.ListFriendRequests(context.Background(), connect.NewRequest(&pb.ListFriendRequestsRequest{
		Incoming: true,
	}))
	if err != nil {
		t.Fatalf("ListFriendRequests (incoming) failed: %v", err)
	}
	if len(listResp.Msg.Requests) != 0 {
		t.Errorf("Expected 0 incoming requests for Alice, got %d", len(listResp.Msg.Requests))
	}
}

func TestCreateBill_NonFriendRegisteredUser_Rejected(t *testing.T) {
	splitClient, _, _, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Bob is registered but not Alice's friend — adding Bob should fail
	_, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:    "Dinner",
		Total:    100,
		Subtotal: 100,
		Participants: []*pb.BillParticipant{
			aliceBP(),
			{DisplayName: "Bob", UserId: func() *string { s := testBobID; return &s }()},
		},
	}))
	if err == nil {
		t.Fatal("Expected error when adding non-friend registered user to bill, got nil")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied, got %v", err)
	}
}

func TestCreateBill_GuestParticipant_AlwaysAllowed(t *testing.T) {
	splitClient, _, _, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Charlie is a guest (no user_id) — always allowed
	resp, err := splitClient.CreateBill(context.Background(), connect.NewRequest(&pb.CreateBillRequest{
		Title:    "Dinner",
		Total:    100,
		Subtotal: 100,
		Participants: []*pb.BillParticipant{
			aliceBP(),
			guestBP("Charlie"),
		},
	}))
	if err != nil {
		t.Fatalf("Expected guest participant to be allowed, got error: %v", err)
	}
	if resp.Msg.BillId == "" {
		t.Error("Expected bill ID in response")
	}
}

func TestCreateGroup_NonFriendMember_Rejected(t *testing.T) {
	_, groupClient, _, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	_, err := groupClient.CreateGroup(context.Background(), connect.NewRequest(&pb.CreateGroupRequest{
		Name: "Test Group",
		Members: []*pb.GroupMember{
			{DisplayName: "Bob", UserId: func() *string { s := testBobID; return &s }()},
		},
	}))
	if err == nil {
		t.Fatal("Expected error when adding non-friend registered user to group, got nil")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied, got %v", err)
	}
}

func TestSearchUsers_ExactEmail_ReturnsUser(t *testing.T) {
	splitClient, _, _, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Exact email for Bob (not a friend) — should still return a result for the "add friend" flow
	resp, err := splitClient.SearchUsers(context.Background(), connect.NewRequest(&pb.SearchUsersRequest{
		Query: "bob@test.com",
	}))
	if err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}
	if len(resp.Msg.Users) != 1 {
		t.Errorf("Expected 1 result for exact email, got %d", len(resp.Msg.Users))
	}
	if len(resp.Msg.Users) > 0 && resp.Msg.Users[0].UserId != testBobID {
		t.Errorf("Expected Bob's UUID, got %s", resp.Msg.Users[0].UserId)
	}
}

func TestSearchUsers_PartialName_NoResults(t *testing.T) {
	splitClient, _, _, _, cleanup := setupTestServerWithFriendService(t)
	defer cleanup()

	// Name query (not an email) → no results; exact email required
	resp, err := splitClient.SearchUsers(context.Background(), connect.NewRequest(&pb.SearchUsersRequest{
		Query: "Bob",
	}))
	if err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}
	if len(resp.Msg.Users) != 0 {
		t.Errorf("Expected 0 results for name query, got %d", len(resp.Msg.Users))
	}
}

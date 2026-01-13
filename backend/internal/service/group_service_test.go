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

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

// setupTestServer creates a test server with an in-memory SQLite database
func setupTestServer(t *testing.T) (protoconnect.SplitServiceClient, func()) {
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

	// Create service and handler
	svc := NewSplitService(store)
	path, handler := protoconnect.NewSplitServiceHandler(svc)

	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := httptest.NewServer(mux)

	client := protoconnect.NewSplitServiceClient(
		http.DefaultClient,
		server.URL,
	)

	cleanup := func() {
		server.Close()
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return client, cleanup
}

func TestCalculateSplit_EqualSplit(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.CalculateSplit(context.Background(), connect.NewRequest(&pb.CalculateSplitRequest{
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		Participants: []string{"Alice", "Bob"},
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
			{Description: "Pizza", Amount: 20, AssignedTo: []string{"Alice"}},
			{Description: "Salad", Amount: 10, AssignedTo: []string{"Bob"}},
		},
		Total:        33, // $3 tax
		Subtotal:     30,
		Participants: []string{"Alice", "Bob"},
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
			{Description: "Shared Pizza", Amount: 30, AssignedTo: []string{"Alice", "Bob", "Charlie"}},
		},
		Total:        33,
		Subtotal:     30,
		Participants: []string{"Alice", "Bob", "Charlie"},
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
			{Description: "Burger", Amount: 15, AssignedTo: []string{"Alice"}},
			{Description: "Fries", Amount: 5, AssignedTo: []string{"Alice", "Bob"}},
		},
		Total:        22,
		Subtotal:     20,
		Participants: []string{"Alice", "Bob"},
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
		Items:        []*pb.Item{},
		Total:        100,
		Subtotal:     100,
		Participants: []string{},
	}))

	if err == nil {
		t.Error("expected error for no participants")
	}
}

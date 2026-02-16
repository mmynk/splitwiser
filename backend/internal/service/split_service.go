package service

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/calculator"
	"github.com/mmynk/splitwiser/internal/middleware"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// SplitService implements the Connect SplitService
type SplitService struct {
	protoconnect.UnimplementedSplitServiceHandler
	store storage.Store
}

// NewSplitService creates a new SplitService with the given storage backend.
func NewSplitService(store storage.Store) *SplitService {
	return &SplitService{store: store}
}

// validatePayerID checks if the payer is one of the participants.
func validatePayerID(payerID string, participants []string) error {
	if payerID == "" {
		return nil // Optional field
	}
	for _, p := range participants {
		if p == payerID {
			return nil
		}
	}
	return fmt.Errorf("payer_id '%s' must be one of the participants", payerID)
}

// isParticipant checks if the user is in the participants list.
func isParticipant(userID string, participants []string) bool {
	for _, p := range participants {
		if p == userID {
			return true
		}
	}
	return false
}

// findNewParticipants returns participants that are not already in existingMembers.
func findNewParticipants(participants, existingMembers []string) []string {
	memberSet := make(map[string]bool, len(existingMembers))
	for _, m := range existingMembers {
		memberSet[m] = true
	}
	var newOnes []string
	for _, p := range participants {
		if !memberSet[p] {
			newOnes = append(newOnes, p)
		}
	}
	return newOnes
}

// autoAddParticipantsToGroup adds any bill participants (and payer) not already in the group.
func (s *SplitService) autoAddParticipantsToGroup(ctx context.Context, groupID string, participants []string, payerID string) {
	if groupID == "" {
		return
	}
	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Warn("autoAddParticipantsToGroup: failed to get group", "group_id", groupID, "error", err)
		return
	}

	// Collect all people to potentially add (participants + payer)
	allPeople := make([]string, 0, len(participants)+1)
	allPeople = append(allPeople, participants...)
	if payerID != "" && !isParticipant(payerID, participants) {
		allPeople = append(allPeople, payerID)
	}

	newMembers := findNewParticipants(allPeople, group.Members)
	if len(newMembers) == 0 {
		return
	}

	if err := s.store.AddGroupMembers(ctx, groupID, newMembers); err != nil {
		slog.Error("autoAddParticipantsToGroup: failed to add members", "group_id", groupID, "error", err)
		return
	}
	slog.Info("Auto-added participants to group", "group_id", groupID, "new_members", newMembers)
}

// CalculateSplit handles bill split calculation
func (s *SplitService) CalculateSplit(ctx context.Context, req *connect.Request[pb.CalculateSplitRequest]) (*connect.Response[pb.CalculateSplitResponse], error) {
	// Convert proto items to calculator items
	items := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		slog.Debug("Processing item",
			"index", i+1,
			"description", item.Description,
			"amount", item.Amount,
			"participants", item.ParticipantIds,
		)
		items[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	splits, err := calculator.CalculateSplit(items, req.Msg.Total, req.Msg.Subtotal, req.Msg.ParticipantIds)
	if err != nil {
		slog.Error("CalculateSplit failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		protoItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			protoItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		slog.Debug("Person split",
			"person", person,
			"subtotal", split.Subtotal,
			"tax", split.Tax,
			"total", split.Total,
			"items_count", len(split.Items),
		)
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    protoItems,
		}
	}

	return connect.NewResponse(&pb.CalculateSplitResponse{
		Splits:    protoSplits,
		TaxAmount: req.Msg.Total - req.Msg.Subtotal,
		Subtotal:  req.Msg.Subtotal,
	}), nil
}

// CreateBill creates a new bill and persists it to storage.
func (s *SplitService) CreateBill(ctx context.Context, req *connect.Request[pb.CreateBillRequest]) (*connect.Response[pb.CreateBillResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Check if user is one of the participants
	if !isParticipant(userID, req.Msg.ParticipantIds) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to create this bill"))
	}

	// Convert proto items to models
	items := make([]models.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		items[i] = models.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	// Validate payer
	if err := validatePayerID(req.Msg.GetPayerId(), req.Msg.ParticipantIds); err != nil {
		slog.Error("CreateBill payer validation failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Create bill model
	bill := &models.Bill{
		Title:        req.Msg.Title,
		Items:        items,
		Total:        req.Msg.Total,
		Subtotal:     req.Msg.Subtotal,
		Participants: req.Msg.ParticipantIds,
	}
	if req.Msg.GetGroupId() != "" {
		bill.GroupID = req.Msg.GetGroupId()
	}
	if req.Msg.GetPayerId() != "" {
		bill.PayerID = req.Msg.GetPayerId()
	}

	// Save to storage (generates ID and CreatedAt)
	if err := s.store.CreateBill(ctx, bill); err != nil {
		slog.Error("CreateBill failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Auto-add bill participants to group
	s.autoAddParticipantsToGroup(ctx, bill.GroupID, bill.Participants, bill.PayerID)

	// Calculate splits
	calcItems := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, req.Msg.Total, req.Msg.Subtotal, req.Msg.ParticipantIds)
	if err != nil {
		slog.Error("CalculateSplit failed during CreateBill", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		protoItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			protoItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    protoItems,
		}
	}

	return connect.NewResponse(&pb.CreateBillResponse{
		BillId: bill.ID,
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: req.Msg.Total - req.Msg.Subtotal,
			Subtotal:  req.Msg.Subtotal,
		},
	}), nil
}

// GetBill retrieves a bill by ID from storage.
func (s *SplitService) GetBill(ctx context.Context, req *connect.Request[pb.GetBillRequest]) (*connect.Response[pb.GetBillResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Retrieve from storage
	bill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("GetBill failed", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Check if user is a participant
	if !isParticipant(userID, bill.Participants) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to view this bill"))
	}

	// Convert items to proto format
	protoItems := make([]*pb.Item, len(bill.Items))
	for i, item := range bill.Items {
		protoItems[i] = &pb.Item{
			Description:    item.Description,
			Amount:         item.Amount,
			ParticipantIds: item.Participants,
		}
	}

	// Recalculate splits
	calcItems := make([]calculator.Item, len(bill.Items))
	for i, item := range bill.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, bill.Total, bill.Subtotal, bill.Participants)
	if err != nil {
		slog.Error("CalculateSplit failed during GetBill", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		personItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			personItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    personItems,
		}
	}

	resp := &pb.GetBillResponse{
		BillId:         bill.ID,
		Title:          bill.Title,
		Items:          protoItems,
		Total:          bill.Total,
		Subtotal:       bill.Subtotal,
		ParticipantIds: bill.Participants,
		PayerId:        bill.PayerID, // Now non-optional
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: bill.Total - bill.Subtotal,
			Subtotal:  bill.Subtotal,
		},
		CreatedAt: bill.CreatedAt,
	}
	if bill.GroupID != "" {
		resp.GroupId = &bill.GroupID
		// Fetch group name
		group, err := s.store.GetGroup(ctx, bill.GroupID)
		if err == nil && group != nil {
			resp.GroupName = &group.Name
		}
	}
	return connect.NewResponse(resp), nil
}

// UpdateBill updates an existing bill.
func (s *SplitService) UpdateBill(ctx context.Context, req *connect.Request[pb.UpdateBillRequest]) (*connect.Response[pb.UpdateBillResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// First, get the existing bill to check permissions
	existingBill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("UpdateBill: failed to get existing bill", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Check if user is a participant in the existing bill
	if !isParticipant(userID, existingBill.Participants) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to update this bill"))
	}

	// Convert proto items to models
	items := make([]models.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		items[i] = models.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	// Validate payer
	if err := validatePayerID(req.Msg.GetPayerId(), req.Msg.ParticipantIds); err != nil {
		slog.Error("UpdateBill payer validation failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Create bill model
	bill := &models.Bill{
		ID:           req.Msg.BillId,
		Title:        req.Msg.Title,
		Items:        items,
		Total:        req.Msg.Total,
		Subtotal:     req.Msg.Subtotal,
		Participants: req.Msg.ParticipantIds,
	}
	if req.Msg.GetGroupId() != "" {
		bill.GroupID = req.Msg.GetGroupId()
	}
	if req.Msg.GetPayerId() != "" {
		bill.PayerID = req.Msg.GetPayerId()
	}

	// Update in storage
	if err := s.store.UpdateBill(ctx, bill); err != nil {
		slog.Error("UpdateBill failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Auto-add bill participants to group
	s.autoAddParticipantsToGroup(ctx, bill.GroupID, bill.Participants, bill.PayerID)

	// Calculate splits
	calcItems := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, req.Msg.Total, req.Msg.Subtotal, req.Msg.ParticipantIds)
	if err != nil {
		slog.Error("CalculateSplit failed during UpdateBill", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		protoItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			protoItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    protoItems,
		}
	}

	return connect.NewResponse(&pb.UpdateBillResponse{
		BillId: bill.ID,
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: req.Msg.Total - req.Msg.Subtotal,
			Subtotal:  req.Msg.Subtotal,
		},
	}), nil
}

// DeleteBill deletes a bill.
func (s *SplitService) DeleteBill(ctx context.Context, req *connect.Request[pb.DeleteBillRequest]) (*connect.Response[pb.DeleteBillResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if req.Msg.BillId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bill_id required"))
	}

	// First, get the existing bill to check permissions
	existingBill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("DeleteBill: failed to get existing bill", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Check if user is a participant in the bill
	if !isParticipant(userID, existingBill.Participants) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to delete this bill"))
	}

	if err := s.store.DeleteBill(ctx, req.Msg.BillId); err != nil {
		slog.Error("DeleteBill failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&pb.DeleteBillResponse{}), nil
}

// ListBillsByGroup retrieves all bills associated with a group.
func (s *SplitService) ListBillsByGroup(ctx context.Context, req *connect.Request[pb.ListBillsByGroupRequest]) (*connect.Response[pb.ListBillsByGroupResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Check if user is a member of the group
	group, err := s.store.GetGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("ListBillsByGroup: failed to get group", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Check if user is a member
	if !isParticipant(userID, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a member of this group"))
	}

	// Retrieve bills from storage
	bills, err := s.store.ListBillsByGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("ListBillsByGroup failed", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to bill summaries
	summaries := make([]*pb.BillSummary, len(bills))
	for i, bill := range bills {
		summaries[i] = &pb.BillSummary{
			BillId:           bill.ID,
			Title:            bill.Title,
			Total:            bill.Total,
			PayerId:          bill.PayerID, // Now non-optional
			CreatedAt:        bill.CreatedAt,
			ParticipantCount: int32(len(bill.Participants)),
		}
	}

	return connect.NewResponse(&pb.ListBillsByGroupResponse{
		Bills: summaries,
	}), nil
}

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

// validatePayerID checks if the payer is one of the participant display names.
func validatePayerID(payerID string, participants []models.BillParticipant) error {
	if payerID == "" {
		return nil
	}
	for _, p := range participants {
		if p.DisplayName == payerID {
			return nil
		}
	}
	return fmt.Errorf("payer_id '%s' must be one of the participants", payerID)
}

// isParticipant checks if the user (by UUID) is in the participants list.
func isParticipant(userID string, participants []models.BillParticipant) bool {
	for _, p := range participants {
		if p.UserID == userID {
			return true
		}
	}
	return false
}

// participantDisplayNames extracts just the display names (for calculator input).
func participantDisplayNames(participants []models.BillParticipant) []string {
	names := make([]string, len(participants))
	for i, p := range participants {
		names[i] = p.DisplayName
	}
	return names
}

// pbToModelParticipants converts proto BillParticipants to model BillParticipants.
func pbToModelParticipants(pbParticipants []*pb.BillParticipant) []models.BillParticipant {
	result := make([]models.BillParticipant, len(pbParticipants))
	for i, p := range pbParticipants {
		result[i] = models.BillParticipant{
			DisplayName: p.DisplayName,
			UserID:      p.GetUserId(),
		}
	}
	return result
}

// modelToPbParticipants converts model BillParticipants to proto BillParticipants.
func modelToPbParticipants(participants []models.BillParticipant) []*pb.BillParticipant {
	result := make([]*pb.BillParticipant, len(participants))
	for i, p := range participants {
		pbp := &pb.BillParticipant{DisplayName: p.DisplayName}
		if p.UserID != "" {
			uid := p.UserID
			pbp.UserId = &uid
		}
		result[i] = pbp
	}
	return result
}

// findNewParticipants returns participants whose display names are not already in existingMembers.
func findNewParticipants(participants []models.BillParticipant, existingMembers []models.GroupMember) []models.GroupMember {
	memberSet := make(map[string]bool, len(existingMembers))
	for _, m := range existingMembers {
		memberSet[m.DisplayName] = true
	}
	var newOnes []models.GroupMember
	for _, p := range participants {
		if !memberSet[p.DisplayName] {
			newOnes = append(newOnes, models.GroupMember{
				DisplayName: p.DisplayName,
				UserID:      p.UserID,
			})
		}
	}
	return newOnes
}

// autoAddParticipantsToGroup adds any bill participants (and payer) not already in the group.
func (s *SplitService) autoAddParticipantsToGroup(ctx context.Context, groupID string, participants []models.BillParticipant, payerID string) {
	if groupID == "" {
		return
	}
	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Warn("autoAddParticipantsToGroup: failed to get group", "group_id", groupID, "error", err)
		return
	}

	// Include payer as a participant if not already listed
	allParticipants := participants
	payerIsParticipant := false
	for _, p := range participants {
		if p.DisplayName == payerID {
			payerIsParticipant = true
			break
		}
	}
	if payerID != "" && !payerIsParticipant {
		allParticipants = append(allParticipants, models.BillParticipant{DisplayName: payerID})
	}

	newMembers := findNewParticipants(allParticipants, group.Members)
	if len(newMembers) == 0 {
		return
	}

	if err := s.store.AddGroupMembersWithIDs(ctx, groupID, newMembers); err != nil {
		slog.Error("autoAddParticipantsToGroup: failed to add members", "group_id", groupID, "error", err)
		return
	}
	slog.Info("Auto-added participants to group", "group_id", groupID, "count", len(newMembers))
}

// CalculateSplit handles bill split calculation
func (s *SplitService) CalculateSplit(ctx context.Context, req *connect.Request[pb.CalculateSplitRequest]) (*connect.Response[pb.CalculateSplitResponse], error) {
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

	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
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

	return connect.NewResponse(&pb.CalculateSplitResponse{
		Splits:    protoSplits,
		TaxAmount: req.Msg.Total - req.Msg.Subtotal,
		Subtotal:  req.Msg.Subtotal,
	}), nil
}

// CreateBill creates a new bill and persists it to storage.
func (s *SplitService) CreateBill(ctx context.Context, req *connect.Request[pb.CreateBillRequest]) (*connect.Response[pb.CreateBillResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	participants := pbToModelParticipants(req.Msg.Participants)

	if !isParticipant(userID, participants) {
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

	if err := validatePayerID(req.Msg.GetPayerId(), participants); err != nil {
		slog.Error("CreateBill payer validation failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	bill := &models.Bill{
		Title:        req.Msg.Title,
		Items:        items,
		Total:        req.Msg.Total,
		Subtotal:     req.Msg.Subtotal,
		Participants: participants,
	}
	if req.Msg.GetGroupId() != "" {
		bill.GroupID = req.Msg.GetGroupId()
	}
	if req.Msg.GetPayerId() != "" {
		bill.PayerID = req.Msg.GetPayerId()
	}

	if err := s.store.CreateBill(ctx, bill); err != nil {
		slog.Error("CreateBill failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.autoAddParticipantsToGroup(ctx, bill.GroupID, bill.Participants, bill.PayerID)

	displayNames := participantDisplayNames(participants)
	calcItems := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, req.Msg.Total, req.Msg.Subtotal, displayNames)
	if err != nil {
		slog.Error("CalculateSplit failed during CreateBill", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
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
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	bill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("GetBill failed", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if !isParticipant(userID, bill.Participants) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to view this bill"))
	}

	protoItems := make([]*pb.Item, len(bill.Items))
	for i, item := range bill.Items {
		protoItems[i] = &pb.Item{
			Description:    item.Description,
			Amount:         item.Amount,
			ParticipantIds: item.Participants,
		}
	}

	displayNames := participantDisplayNames(bill.Participants)
	calcItems := make([]calculator.Item, len(bill.Items))
	for i, item := range bill.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, bill.Total, bill.Subtotal, displayNames)
	if err != nil {
		slog.Error("CalculateSplit failed during GetBill", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
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
		BillId:       bill.ID,
		Title:        bill.Title,
		Items:        protoItems,
		Total:        bill.Total,
		Subtotal:     bill.Subtotal,
		Participants: modelToPbParticipants(bill.Participants),
		PayerId:      bill.PayerID,
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: bill.Total - bill.Subtotal,
			Subtotal:  bill.Subtotal,
		},
		CreatedAt: bill.CreatedAt,
	}
	if bill.GroupID != "" {
		resp.GroupId = &bill.GroupID
		group, err := s.store.GetGroup(ctx, bill.GroupID)
		if err == nil && group != nil {
			resp.GroupName = &group.Name
		}
	}
	return connect.NewResponse(resp), nil
}

// UpdateBill updates an existing bill.
func (s *SplitService) UpdateBill(ctx context.Context, req *connect.Request[pb.UpdateBillRequest]) (*connect.Response[pb.UpdateBillResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	existingBill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("UpdateBill: failed to get existing bill", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if !isParticipant(userID, existingBill.Participants) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to update this bill"))
	}

	participants := pbToModelParticipants(req.Msg.Participants)

	items := make([]models.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		items[i] = models.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	if err := validatePayerID(req.Msg.GetPayerId(), participants); err != nil {
		slog.Error("UpdateBill payer validation failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	bill := &models.Bill{
		ID:           req.Msg.BillId,
		Title:        req.Msg.Title,
		Items:        items,
		Total:        req.Msg.Total,
		Subtotal:     req.Msg.Subtotal,
		Participants: participants,
	}
	if req.Msg.GetGroupId() != "" {
		bill.GroupID = req.Msg.GetGroupId()
	}
	if req.Msg.GetPayerId() != "" {
		bill.PayerID = req.Msg.GetPayerId()
	}

	if err := s.store.UpdateBill(ctx, bill); err != nil {
		slog.Error("UpdateBill failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	s.autoAddParticipantsToGroup(ctx, bill.GroupID, bill.Participants, bill.PayerID)

	displayNames := participantDisplayNames(participants)
	calcItems := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.ParticipantIds,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, req.Msg.Total, req.Msg.Subtotal, displayNames)
	if err != nil {
		slog.Error("CalculateSplit failed during UpdateBill", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
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
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if req.Msg.BillId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bill_id required"))
	}

	existingBill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("DeleteBill: failed to get existing bill", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if !isParticipant(userID, existingBill.Participants) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a participant to delete this bill"))
	}

	if err := s.store.DeleteBill(ctx, req.Msg.BillId); err != nil {
		slog.Error("DeleteBill failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&pb.DeleteBillResponse{}), nil
}

// ListMyBills retrieves all bills where the authenticated user is a participant.
func (s *SplitService) ListMyBills(ctx context.Context, req *connect.Request[pb.ListMyBillsRequest]) (*connect.Response[pb.ListMyBillsResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	bills, err := s.store.ListBillsByParticipant(ctx, userID)
	if err != nil {
		slog.Error("ListMyBills failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Collect unique group IDs to fetch names
	groupIDs := make(map[string]struct{})
	for _, bill := range bills {
		if bill.GroupID != "" {
			groupIDs[bill.GroupID] = struct{}{}
		}
	}
	groupNames := make(map[string]string, len(groupIDs))
	for gid := range groupIDs {
		if group, err := s.store.GetGroup(ctx, gid); err == nil && group != nil {
			groupNames[gid] = group.Name
		}
	}

	summaries := make([]*pb.BillSummary, len(bills))
	for i, bill := range bills {
		s := &pb.BillSummary{
			BillId:           bill.ID,
			Title:            bill.Title,
			Total:            bill.Total,
			PayerId:          bill.PayerID,
			CreatedAt:        bill.CreatedAt,
			ParticipantCount: int32(len(bill.Participants)),
		}
		if name, ok := groupNames[bill.GroupID]; ok {
			s.GroupName = &name
		}
		summaries[i] = s
	}

	return connect.NewResponse(&pb.ListMyBillsResponse{Bills: summaries}), nil
}

// ListBillsByGroup retrieves all bills associated with a group.
func (s *SplitService) ListBillsByGroup(ctx context.Context, req *connect.Request[pb.ListBillsByGroupRequest]) (*connect.Response[pb.ListBillsByGroupResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	group, err := s.store.GetGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("ListBillsByGroup: failed to get group", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if !isMember(userID, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you must be a member of this group"))
	}

	bills, err := s.store.ListBillsByGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("ListBillsByGroup failed", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	summaries := make([]*pb.BillSummary, len(bills))
	for i, bill := range bills {
		summaries[i] = &pb.BillSummary{
			BillId:           bill.ID,
			Title:            bill.Title,
			Total:            bill.Total,
			PayerId:          bill.PayerID,
			CreatedAt:        bill.CreatedAt,
			ParticipantCount: int32(len(bill.Participants)),
		}
	}

	return connect.NewResponse(&pb.ListBillsByGroupResponse{
		Bills: summaries,
	}), nil
}

// SearchUsers finds registered users by name or email prefix (min 2 chars).
func (s *SplitService) SearchUsers(ctx context.Context, req *connect.Request[pb.SearchUsersRequest]) (*connect.Response[pb.SearchUsersResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	query := req.Msg.Query
	if len(query) < 2 {
		return connect.NewResponse(&pb.SearchUsersResponse{}), nil
	}

	users, err := s.store.SearchUsers(ctx, query, 10)
	if err != nil {
		slog.Error("SearchUsers failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	results := make([]*pb.UserSearchResult, len(users))
	for i, u := range users {
		results[i] = &pb.UserSearchResult{
			UserId:      u.ID,
			DisplayName: u.DisplayName,
			Email:       u.Email,
		}
	}

	return connect.NewResponse(&pb.SearchUsersResponse{Users: results}), nil
}

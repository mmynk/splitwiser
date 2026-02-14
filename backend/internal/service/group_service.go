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

// GroupService implements the Connect GroupService
type GroupService struct {
	protoconnect.UnimplementedGroupServiceHandler
	store storage.Store
}

// NewGroupService creates a new GroupService with the given storage backend.
func NewGroupService(store storage.Store) *GroupService {
	return &GroupService{store: store}
}

// isMember checks if the user is in the members list.
func isMember(userID string, members []string) bool {
	for _, m := range members {
		if m == userID {
			return true
		}
	}
	return false
}

// CreateGroup creates a new group.
func (s *GroupService) CreateGroup(ctx context.Context, req *connect.Request[pb.CreateGroupRequest]) (*connect.Response[pb.CreateGroupResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	slog.Info("CreateGroup request received",
		"user_id", userID,
		"name", req.Msg.Name,
		"members_count", len(req.Msg.MemberIds),
	)

	// Add creator to members if not already present
	members := req.Msg.MemberIds
	if !isMember(userID, members) {
		members = append([]string{userID}, members...)
	}

	// Create group model
	group := &models.Group{
		Name:    req.Msg.Name,
		Members: members,
	}

	// Save to storage (generates ID and CreatedAt)
	if err := s.store.CreateGroup(ctx, group); err != nil {
		slog.Error("CreateGroup failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("Group created", "group_id", group.ID)

	return connect.NewResponse(&pb.CreateGroupResponse{
		Group: &pb.Group{
			Id:        group.ID,
			Name:      group.Name,
			MemberIds: group.Members,
			CreatedAt: group.CreatedAt,
		},
	}), nil
}

// GetGroup retrieves a group by ID.
func (s *GroupService) GetGroup(ctx context.Context, req *connect.Request[pb.GetGroupRequest]) (*connect.Response[pb.GetGroupResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	slog.Info("GetGroup request received", "user_id", userID, "group_id", req.Msg.GroupId)

	group, err := s.store.GetGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("GetGroup failed", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	slog.Info("GetGroup successful", "group_id", group.ID, "name", group.Name)

	return connect.NewResponse(&pb.GetGroupResponse{
		Group: &pb.Group{
			Id:        group.ID,
			Name:      group.Name,
			MemberIds: group.Members,
			CreatedAt: group.CreatedAt,
		},
	}), nil
}

// ListGroups retrieves all groups.
func (s *GroupService) ListGroups(ctx context.Context, req *connect.Request[pb.ListGroupsRequest]) (*connect.Response[pb.ListGroupsResponse], error) {
	slog.Info("ListGroups request received")

	groups, err := s.store.ListGroups(ctx)
	if err != nil {
		slog.Error("ListGroups failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to proto format
	protoGroups := make([]*pb.Group, len(groups))
	for i, group := range groups {
		protoGroups[i] = &pb.Group{
			Id:        group.ID,
			Name:      group.Name,
			MemberIds: group.Members,
			CreatedAt: group.CreatedAt,
		}
	}

	slog.Info("ListGroups successful", "count", len(groups))

	return connect.NewResponse(&pb.ListGroupsResponse{
		Groups: protoGroups,
	}), nil
}

// UpdateGroup updates an existing group.
func (s *GroupService) UpdateGroup(ctx context.Context, req *connect.Request[pb.UpdateGroupRequest]) (*connect.Response[pb.UpdateGroupResponse], error) {
	slog.Info("UpdateGroup request received",
		"group_id", req.Msg.GroupId,
		"name", req.Msg.Name,
		"members_count", len(req.Msg.MemberIds),
	)

	// Create group model
	group := &models.Group{
		ID:      req.Msg.GroupId,
		Name:    req.Msg.Name,
		Members: req.Msg.MemberIds,
	}

	// Update in storage
	if err := s.store.UpdateGroup(ctx, group); err != nil {
		slog.Error("UpdateGroup failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Fetch updated group to get CreatedAt
	updatedGroup, err := s.store.GetGroup(ctx, group.ID)
	if err != nil {
		slog.Error("Failed to fetch updated group", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("Group updated", "group_id", group.ID)

	return connect.NewResponse(&pb.UpdateGroupResponse{
		Group: &pb.Group{
			Id:        updatedGroup.ID,
			Name:      updatedGroup.Name,
			MemberIds: updatedGroup.Members,
			CreatedAt: updatedGroup.CreatedAt,
		},
	}), nil
}

// DeleteGroup removes a group by ID.
func (s *GroupService) DeleteGroup(ctx context.Context, req *connect.Request[pb.DeleteGroupRequest]) (*connect.Response[pb.DeleteGroupResponse], error) {
	slog.Info("DeleteGroup request received", "group_id", req.Msg.GroupId)

	if err := s.store.DeleteGroup(ctx, req.Msg.GroupId); err != nil {
		slog.Error("DeleteGroup failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	slog.Info("Group deleted", "group_id", req.Msg.GroupId)

	return connect.NewResponse(&pb.DeleteGroupResponse{}), nil
}

// GetGroupBalances calculates balances across all bills in a group.
func (s *GroupService) GetGroupBalances(ctx context.Context, req *connect.Request[pb.GetGroupBalancesRequest]) (*connect.Response[pb.GetGroupBalancesResponse], error) {
	groupID := req.Msg.GetGroupId()
	slog.Info("GetGroupBalances request received", "group_id", groupID)

	if groupID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id required"))
	}

	// Verify group exists
	_, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Error("GetGroupBalances failed - group not found", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}

	// Get all bills for this group
	billSummaries, err := s.store.ListBillsByGroup(ctx, groupID)
	if err != nil {
		slog.Error("GetGroupBalances failed - could not list bills", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Fetch full bill details for each
	var bills []calculator.BillForBalance
	for _, summary := range billSummaries {
		bill, err := s.store.GetBill(ctx, summary.ID)
		if err != nil {
			slog.Error("GetGroupBalances failed - could not get bill", "bill_id", summary.ID, "error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Convert to calculator format
		calcItems := make([]calculator.Item, len(bill.Items))
		for i, item := range bill.Items {
			calcItems[i] = calculator.Item{
				Description:  item.Description,
				Amount:       item.Amount,
				Participants: item.Participants,
			}
		}

		bills = append(bills, calculator.BillForBalance{
			Total:        bill.Total,
			Subtotal:     bill.Subtotal,
			PayerID:      bill.PayerID,
			Items:        calcItems,
			Participants: bill.Participants,
		})
	}

	// Fetch settlements for this group
	settlementsList, err := s.store.ListSettlementsByGroup(ctx, groupID)
	if err != nil {
		slog.Error("GetGroupBalances failed - could not list settlements", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert settlements to calculator format
	calcSettlements := make([]calculator.SettlementForBalance, len(settlementsList))
	for i, settlement := range settlementsList {
		calcSettlements[i] = calculator.SettlementForBalance{
			FromUserID: settlement.FromUserID,
			ToUserID:   settlement.ToUserID,
			Amount:     settlement.Amount,
		}
	}

	// Calculate balances
	memberBalances, debtEdges, err := calculator.CalculateGroupBalances(bills, calcSettlements)
	if err != nil {
		slog.Error("GetGroupBalances failed - calculation error", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to proto messages
	pbBalances := make([]*pb.MemberBalance, len(memberBalances))
	for i, bal := range memberBalances {
		pbBalances[i] = &pb.MemberBalance{
			DisplayName: bal.MemberName,
			NetBalance:  bal.NetBalance,
			TotalPaid:   bal.TotalPaid,
			TotalOwed:   bal.TotalOwed,
		}
	}

	pbDebts := make([]*pb.DebtEdge, len(debtEdges))
	for i, debt := range debtEdges {
		pbDebts[i] = &pb.DebtEdge{
			FromUserId: debt.From,
			ToUserId:   debt.To,
			Amount:     debt.Amount,
		}
	}

	slog.Info("GetGroupBalances successful",
		"group_id", groupID,
		"bills_count", len(bills),
		"members_count", len(memberBalances),
		"debts_count", len(debtEdges),
	)

	return connect.NewResponse(&pb.GetGroupBalancesResponse{
		MemberBalances: pbBalances,
		DebtMatrix:     pbDebts,
	}), nil
}

// RecordSettlement records a payment between group members.
func (s *GroupService) RecordSettlement(ctx context.Context, req *connect.Request[pb.RecordSettlementRequest]) (*connect.Response[pb.RecordSettlementResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	groupID := req.Msg.GetGroupId()
	fromUserID := req.Msg.GetFromUserId()
	toUserID := req.Msg.GetToUserId()
	amount := req.Msg.GetAmount()
	note := req.Msg.GetNote()

	slog.Info("RecordSettlement request received",
		"user_id", userID,
		"group_id", groupID,
		"from_user_id", fromUserID,
		"to_user_id", toUserID,
		"amount", amount,
	)

	// Validation
	if groupID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id required"))
	}
	if fromUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("from_user_id required"))
	}
	if toUserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("to_user_id required"))
	}
	if amount <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount must be positive"))
	}
	if fromUserID == toUserID {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("from_user_id and to_user_id must be different"))
	}

	// Verify group exists and user is a member
	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Error("RecordSettlement failed - group not found", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}

	if !isMember(userID, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this group"))
	}

	// Verify from_user and to_user are members
	if !isMember(fromUserID, group.Members) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("from_user is not a member of this group"))
	}
	if !isMember(toUserID, group.Members) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("to_user is not a member of this group"))
	}

	// Create settlement
	settlement := &models.Settlement{
		GroupID:    groupID,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Amount:     amount,
		CreatedBy:  userID,
		Note:       note,
	}

	if err := s.store.CreateSettlement(ctx, settlement); err != nil {
		slog.Error("RecordSettlement failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("Settlement recorded", "settlement_id", settlement.ID)

	// Get display names for response
	fromName := fromUserID
	toName := toUserID

	return connect.NewResponse(&pb.RecordSettlementResponse{
		Settlement: &pb.Settlement{
			Id:         settlement.ID,
			GroupId:    settlement.GroupID,
			FromUserId: settlement.FromUserID,
			ToUserId:   settlement.ToUserID,
			Amount:     settlement.Amount,
			CreatedAt:  settlement.CreatedAt,
			CreatedBy:  settlement.CreatedBy,
			Note:       settlement.Note,
			FromName:   fromName,
			ToName:     toName,
		},
	}), nil
}

// ListSettlements lists all settlements for a group.
func (s *GroupService) ListSettlements(ctx context.Context, req *connect.Request[pb.ListSettlementsRequest]) (*connect.Response[pb.ListSettlementsResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	groupID := req.Msg.GetGroupId()

	slog.Info("ListSettlements request received", "user_id", userID, "group_id", groupID)

	if groupID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id required"))
	}

	// Verify group exists and user is a member
	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Error("ListSettlements failed - group not found", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}

	if !isMember(userID, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this group"))
	}

	// Get settlements
	settlements, err := s.store.ListSettlementsByGroup(ctx, groupID)
	if err != nil {
		slog.Error("ListSettlements failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to proto
	pbSettlements := make([]*pb.Settlement, len(settlements))
	for i, settlement := range settlements {
		pbSettlements[i] = &pb.Settlement{
			Id:         settlement.ID,
			GroupId:    settlement.GroupID,
			FromUserId: settlement.FromUserID,
			ToUserId:   settlement.ToUserID,
			Amount:     settlement.Amount,
			CreatedAt:  settlement.CreatedAt,
			CreatedBy:  settlement.CreatedBy,
			Note:       settlement.Note,
			FromName:   settlement.FromUserID, // Use ID as fallback
			ToName:     settlement.ToUserID,
		}
	}

	slog.Info("ListSettlements successful", "group_id", groupID, "count", len(settlements))

	return connect.NewResponse(&pb.ListSettlementsResponse{
		Settlements: pbSettlements,
	}), nil
}

// DeleteSettlement removes a settlement.
func (s *GroupService) DeleteSettlement(ctx context.Context, req *connect.Request[pb.DeleteSettlementRequest]) (*connect.Response[pb.DeleteSettlementResponse], error) {
	// Get authenticated user ID from context
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	settlementID := req.Msg.GetSettlementId()

	slog.Info("DeleteSettlement request received", "user_id", userID, "settlement_id", settlementID)

	if settlementID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("settlement_id required"))
	}

	// Get settlement to check group membership
	settlement, err := s.store.GetSettlement(ctx, settlementID)
	if err != nil {
		slog.Error("DeleteSettlement failed - settlement not found", "settlement_id", settlementID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("settlement not found"))
	}

	// Verify user is a member of the group
	group, err := s.store.GetGroup(ctx, settlement.GroupID)
	if err != nil {
		slog.Error("DeleteSettlement failed - group not found", "group_id", settlement.GroupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("group not found"))
	}

	if !isMember(userID, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this group"))
	}

	// Delete settlement
	if err := s.store.DeleteSettlement(ctx, settlementID); err != nil {
		slog.Error("DeleteSettlement failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("Settlement deleted", "settlement_id", settlementID)

	return connect.NewResponse(&pb.DeleteSettlementResponse{}), nil
}

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

// isMember checks if the user (by UUID) is in the members list.
func isMember(userID string, members []models.GroupMember) bool {
	for _, m := range members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

// isMemberByName checks if a display name is in the members list.
func isMemberByName(name string, members []models.GroupMember) bool {
	for _, m := range members {
		if m.DisplayName == name {
			return true
		}
	}
	return false
}

// resolveDisplayName looks up a user's display name by their ID.
// Falls back to the ID itself if lookup fails (e.g. in tests).
func (s *GroupService) resolveDisplayName(ctx context.Context, userID string) string {
	users, err := s.store.GetUsersByIDs(ctx, []string{userID})
	if err != nil || users[userID] == nil {
		return userID
	}
	return users[userID].DisplayName
}

// modelToPbMembers converts model GroupMembers to proto GroupMembers.
func modelToPbMembers(members []models.GroupMember) []*pb.GroupMember {
	result := make([]*pb.GroupMember, len(members))
	for i, m := range members {
		pbm := &pb.GroupMember{DisplayName: m.DisplayName}
		if m.UserID != "" {
			uid := m.UserID
			pbm.UserId = &uid
		}
		result[i] = pbm
	}
	return result
}

// pbToModelMembers converts proto GroupMembers to model GroupMembers.
func pbToModelMembers(pbMembers []*pb.GroupMember) []models.GroupMember {
	result := make([]models.GroupMember, len(pbMembers))
	for i, m := range pbMembers {
		result[i] = models.GroupMember{
			DisplayName: m.DisplayName,
			UserID:      m.GetUserId(),
		}
	}
	return result
}

// CreateGroup creates a new group.
func (s *GroupService) CreateGroup(ctx context.Context, req *connect.Request[pb.CreateGroupRequest]) (*connect.Response[pb.CreateGroupResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	creatorName := s.resolveDisplayName(ctx, userID)

	members := pbToModelMembers(req.Msg.Members)

	// Add creator with their user_id if not already present
	if !isMemberByName(creatorName, members) {
		members = append([]models.GroupMember{{DisplayName: creatorName, UserID: userID}}, members...)
	}

	group := &models.Group{
		Name:    req.Msg.Name,
		Members: members,
	}

	if err := s.store.CreateGroup(ctx, group); err != nil {
		slog.Error("CreateGroup failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.CreateGroupResponse{
		Group: &pb.Group{
			Id:        group.ID,
			Name:      group.Name,
			Members:   modelToPbMembers(group.Members),
			CreatedAt: group.CreatedAt,
		},
	}), nil
}

// GetGroup retrieves a group by ID.
func (s *GroupService) GetGroup(ctx context.Context, req *connect.Request[pb.GetGroupRequest]) (*connect.Response[pb.GetGroupResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	group, err := s.store.GetGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("GetGroup failed", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&pb.GetGroupResponse{
		Group: &pb.Group{
			Id:        group.ID,
			Name:      group.Name,
			Members:   modelToPbMembers(group.Members),
			CreatedAt: group.CreatedAt,
		},
	}), nil
}

// ListGroups retrieves all groups the authenticated user belongs to.
func (s *GroupService) ListGroups(ctx context.Context, req *connect.Request[pb.ListGroupsRequest]) (*connect.Response[pb.ListGroupsResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	groups, err := s.store.ListGroupsByUser(ctx, userID)
	if err != nil {
		slog.Error("ListGroups failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoGroups := make([]*pb.Group, len(groups))
	for i, group := range groups {
		protoGroups[i] = &pb.Group{
			Id:        group.ID,
			Name:      group.Name,
			Members:   modelToPbMembers(group.Members),
			CreatedAt: group.CreatedAt,
		}
	}

	return connect.NewResponse(&pb.ListGroupsResponse{
		Groups: protoGroups,
	}), nil
}

// UpdateGroup updates an existing group.
func (s *GroupService) UpdateGroup(ctx context.Context, req *connect.Request[pb.UpdateGroupRequest]) (*connect.Response[pb.UpdateGroupResponse], error) {
	group := &models.Group{
		ID:      req.Msg.GroupId,
		Name:    req.Msg.Name,
		Members: pbToModelMembers(req.Msg.Members),
	}

	if err := s.store.UpdateGroup(ctx, group); err != nil {
		slog.Error("UpdateGroup failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	updatedGroup, err := s.store.GetGroup(ctx, group.ID)
	if err != nil {
		slog.Error("Failed to fetch updated group", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.UpdateGroupResponse{
		Group: &pb.Group{
			Id:        updatedGroup.ID,
			Name:      updatedGroup.Name,
			Members:   modelToPbMembers(updatedGroup.Members),
			CreatedAt: updatedGroup.CreatedAt,
		},
	}), nil
}

// DeleteGroup removes a group by ID.
func (s *GroupService) DeleteGroup(ctx context.Context, req *connect.Request[pb.DeleteGroupRequest]) (*connect.Response[pb.DeleteGroupResponse], error) {
	if err := s.store.DeleteGroup(ctx, req.Msg.GroupId); err != nil {
		slog.Error("DeleteGroup failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&pb.DeleteGroupResponse{}), nil
}

// GetGroupBalances calculates balances across all bills in a group.
func (s *GroupService) GetGroupBalances(ctx context.Context, req *connect.Request[pb.GetGroupBalancesRequest]) (*connect.Response[pb.GetGroupBalancesResponse], error) {
	groupID := req.Msg.GetGroupId()
	if groupID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id required"))
	}

	_, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Error("GetGroupBalances failed - group not found", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}

	billSummaries, err := s.store.ListBillsByGroup(ctx, groupID)
	if err != nil {
		slog.Error("GetGroupBalances failed - could not list bills", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var bills []calculator.BillForBalance
	for _, summary := range billSummaries {
		bill, err := s.store.GetBill(ctx, summary.ID)
		if err != nil {
			slog.Error("GetGroupBalances failed - could not get bill", "bill_id", summary.ID, "error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

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
			Participants: participantDisplayNames(bill.Participants),
		})
	}

	settlementsList, err := s.store.ListSettlementsByGroup(ctx, groupID)
	if err != nil {
		slog.Error("GetGroupBalances failed - could not list settlements", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	calcSettlements := make([]calculator.SettlementForBalance, len(settlementsList))
	for i, settlement := range settlementsList {
		calcSettlements[i] = calculator.SettlementForBalance{
			FromUserID: settlement.FromUserID,
			ToUserID:   settlement.ToUserID,
			Amount:     settlement.Amount,
		}
	}

	memberBalances, debtEdges, err := calculator.CalculateGroupBalances(bills, calcSettlements)
	if err != nil {
		slog.Error("GetGroupBalances failed - calculation error", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

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

	return connect.NewResponse(&pb.GetGroupBalancesResponse{
		MemberBalances: pbBalances,
		DebtMatrix:     pbDebts,
	}), nil
}

// RecordSettlement records a payment between group members.
func (s *GroupService) RecordSettlement(ctx context.Context, req *connect.Request[pb.RecordSettlementRequest]) (*connect.Response[pb.RecordSettlementResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	groupID := req.Msg.GetGroupId()
	fromUserID := req.Msg.GetFromUserId()
	toUserID := req.Msg.GetToUserId()
	amount := req.Msg.GetAmount()
	note := req.Msg.GetNote()

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

	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Error("RecordSettlement failed - group not found", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}

	creatorDisplayName := s.resolveDisplayName(ctx, userID)
	if !isMemberByName(creatorDisplayName, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this group"))
	}

	// from/to are display names
	if !isMemberByName(fromUserID, group.Members) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("from_user is not a member of this group"))
	}
	if !isMemberByName(toUserID, group.Members) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("to_user is not a member of this group"))
	}

	settlement := &models.Settlement{
		GroupID:    groupID,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Amount:     amount,
		CreatedBy:  creatorDisplayName,
		Note:       note,
	}

	if err := s.store.CreateSettlement(ctx, settlement); err != nil {
		slog.Error("RecordSettlement failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

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
			FromName:   fromUserID,
			ToName:     toUserID,
		},
	}), nil
}

// ListSettlements lists all settlements for a group.
func (s *GroupService) ListSettlements(ctx context.Context, req *connect.Request[pb.ListSettlementsRequest]) (*connect.Response[pb.ListSettlementsResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	groupID := req.Msg.GetGroupId()
	if groupID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id required"))
	}

	group, err := s.store.GetGroup(ctx, groupID)
	if err != nil {
		slog.Error("ListSettlements failed - group not found", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}

	memberDisplayName := s.resolveDisplayName(ctx, userID)
	if !isMemberByName(memberDisplayName, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this group"))
	}

	settlements, err := s.store.ListSettlementsByGroup(ctx, groupID)
	if err != nil {
		slog.Error("ListSettlements failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

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
			FromName:   settlement.FromUserID,
			ToName:     settlement.ToUserID,
		}
	}

	return connect.NewResponse(&pb.ListSettlementsResponse{
		Settlements: pbSettlements,
	}), nil
}

// DeleteSettlement removes a settlement.
func (s *GroupService) DeleteSettlement(ctx context.Context, req *connect.Request[pb.DeleteSettlementRequest]) (*connect.Response[pb.DeleteSettlementResponse], error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	settlementID := req.Msg.GetSettlementId()
	if settlementID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("settlement_id required"))
	}

	settlement, err := s.store.GetSettlement(ctx, settlementID)
	if err != nil {
		slog.Error("DeleteSettlement failed - settlement not found", "settlement_id", settlementID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("settlement not found"))
	}

	group, err := s.store.GetGroup(ctx, settlement.GroupID)
	if err != nil {
		slog.Error("DeleteSettlement failed - group not found", "group_id", settlement.GroupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("group not found"))
	}

	deletorDisplayName := s.resolveDisplayName(ctx, userID)
	if !isMemberByName(deletorDisplayName, group.Members) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this group"))
	}

	if err := s.store.DeleteSettlement(ctx, settlementID); err != nil {
		slog.Error("DeleteSettlement failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.DeleteSettlementResponse{}), nil
}

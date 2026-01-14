package service

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/calculator"
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

// CreateGroup creates a new group.
func (s *GroupService) CreateGroup(ctx context.Context, req *connect.Request[pb.CreateGroupRequest]) (*connect.Response[pb.CreateGroupResponse], error) {
	slog.Info("CreateGroup request received",
		"name", req.Msg.Name,
		"members_count", len(req.Msg.Members),
	)

	// Create group model
	group := &models.Group{
		Name:    req.Msg.Name,
		Members: req.Msg.Members,
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
			Members:   group.Members,
			CreatedAt: group.CreatedAt,
		},
	}), nil
}

// GetGroup retrieves a group by ID.
func (s *GroupService) GetGroup(ctx context.Context, req *connect.Request[pb.GetGroupRequest]) (*connect.Response[pb.GetGroupResponse], error) {
	slog.Info("GetGroup request received", "group_id", req.Msg.GroupId)

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
			Members:   group.Members,
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
			Members:   group.Members,
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
		"members_count", len(req.Msg.Members),
	)

	// Create group model
	group := &models.Group{
		ID:      req.Msg.GroupId,
		Name:    req.Msg.Name,
		Members: req.Msg.Members,
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
			Members:   updatedGroup.Members,
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

	// Calculate balances
	memberBalances, debtEdges, err := calculator.CalculateGroupBalances(bills)
	if err != nil {
		slog.Error("GetGroupBalances failed - calculation error", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to proto messages
	pbBalances := make([]*pb.MemberBalance, len(memberBalances))
	for i, bal := range memberBalances {
		pbBalances[i] = &pb.MemberBalance{
			MemberName:  bal.MemberName,
			NetBalance:  bal.NetBalance,
			TotalPaid:   bal.TotalPaid,
			TotalOwed:   bal.TotalOwed,
		}
	}

	pbDebts := make([]*pb.DebtEdge, len(debtEdges))
	for i, debt := range debtEdges {
		pbDebts[i] = &pb.DebtEdge{
			From:   debt.From,
			To:     debt.To,
			Amount: debt.Amount,
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

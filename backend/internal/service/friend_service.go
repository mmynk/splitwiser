package service

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/middleware"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// FriendService implements the Connect FriendService.
type FriendService struct {
	protoconnect.UnimplementedFriendServiceHandler
	store storage.Store
}

// NewFriendService creates a new FriendService with the given storage backend.
func NewFriendService(store storage.Store) *FriendService {
	return &FriendService{store: store}
}

// SendFriendRequest sends a friend request to another registered user.
func (s *FriendService) SendFriendRequest(ctx context.Context, req *connect.Request[pb.SendFriendRequestRequest]) (*connect.Response[pb.SendFriendRequestResponse], error) {
	callerID := middleware.GetUserID(ctx)
	if callerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	addresseeID := req.Msg.AddresseeId
	if addresseeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("addressee_id required"))
	}
	if addresseeID == callerID {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cannot send friend request to yourself"))
	}

	// Verify addressee exists
	users, err := s.store.GetUsersByIDs(ctx, []string{addresseeID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lookup user: %w", err))
	}
	addressee, ok := users[addresseeID]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found"))
	}

	// Look up caller's display name
	callerUsers, err := s.store.GetUsersByIDs(ctx, []string{callerID})
	if err != nil || callerUsers[callerID] == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lookup caller: %w", err))
	}
	caller := callerUsers[callerID]

	friendship := &models.Friendship{
		RequesterID: callerID,
		AddresseeID: addresseeID,
		Status:      models.FriendshipPending,
	}

	if err := s.store.SendFriendRequest(ctx, friendship); err != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	}

	return connect.NewResponse(&pb.SendFriendRequestResponse{
		Request: &pb.FriendRequest{
			Id:                     friendship.ID,
			RequesterId:            callerID,
			RequesterDisplayName:   caller.DisplayName,
			AddresseeId:            addresseeID,
			AddresseeDisplayName:   addressee.DisplayName,
			Status:                 string(friendship.Status),
			CreatedAt:              friendship.CreatedAt,
		},
	}), nil
}

// RespondToFriendRequest accepts or declines a pending friend request.
func (s *FriendService) RespondToFriendRequest(ctx context.Context, req *connect.Request[pb.RespondToFriendRequestRequest]) (*connect.Response[pb.RespondToFriendRequestResponse], error) {
	callerID := middleware.GetUserID(ctx)
	if callerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if req.Msg.RequestId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request_id required"))
	}

	friendship, err := s.store.GetFriendship(ctx, req.Msg.RequestId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("friend request not found"))
	}

	if friendship.AddresseeID != callerID {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("only the addressee can respond to this request"))
	}

	if friendship.Status != models.FriendshipPending {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("friend request is no longer pending"))
	}

	newStatus := models.FriendshipDeclined
	if req.Msg.Accept {
		newStatus = models.FriendshipAccepted
	}

	if err := s.store.UpdateFriendshipStatus(ctx, friendship.ID, newStatus); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	friendship.Status = newStatus

	// Hydrate display names
	userMap, err := s.store.GetUsersByIDs(ctx, []string{friendship.RequesterID, friendship.AddresseeID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbReq := friendshipToProto(friendship, userMap)
	return connect.NewResponse(&pb.RespondToFriendRequestResponse{Request: pbReq}), nil
}

// ListFriends returns all accepted friends of the authenticated user.
func (s *FriendService) ListFriends(ctx context.Context, req *connect.Request[pb.ListFriendsRequest]) (*connect.Response[pb.ListFriendsResponse], error) {
	callerID := middleware.GetUserID(ctx)
	if callerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	friends, err := s.store.GetFriends(ctx, callerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbFriends := make([]*pb.Friend, len(friends))
	for i, u := range friends {
		pbFriends[i] = &pb.Friend{
			UserId:      u.ID,
			DisplayName: u.DisplayName,
			Email:       u.Email,
		}
	}

	return connect.NewResponse(&pb.ListFriendsResponse{Friends: pbFriends}), nil
}

// ListFriendRequests lists pending incoming or outgoing friend requests.
func (s *FriendService) ListFriendRequests(ctx context.Context, req *connect.Request[pb.ListFriendRequestsRequest]) (*connect.Response[pb.ListFriendRequestsResponse], error) {
	callerID := middleware.GetUserID(ctx)
	if callerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	friendships, err := s.store.ListFriendships(ctx, callerID, req.Msg.Incoming, models.FriendshipPending)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Collect all user IDs to fetch in one query
	idSet := make(map[string]struct{})
	for _, f := range friendships {
		idSet[f.RequesterID] = struct{}{}
		idSet[f.AddresseeID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	userMap, err := s.store.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbRequests := make([]*pb.FriendRequest, len(friendships))
	for i, f := range friendships {
		pbRequests[i] = friendshipToProto(f, userMap)
	}

	return connect.NewResponse(&pb.ListFriendRequestsResponse{Requests: pbRequests}), nil
}

// RemoveFriend removes an accepted friendship.
func (s *FriendService) RemoveFriend(ctx context.Context, req *connect.Request[pb.RemoveFriendRequest]) (*connect.Response[pb.RemoveFriendResponse], error) {
	callerID := middleware.GetUserID(ctx)
	if callerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if req.Msg.UserId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id required"))
	}

	friendship, err := s.store.GetFriendshipBetween(ctx, callerID, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no friendship found with that user"))
	}
	if friendship.Status != models.FriendshipAccepted {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no accepted friendship with that user"))
	}
	if err := s.store.DeleteFriendship(ctx, friendship.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&pb.RemoveFriendResponse{}), nil
}

// friendshipToProto converts a Friendship model to proto, hydrating display names from userMap.
func friendshipToProto(f *models.Friendship, userMap map[string]*models.User) *pb.FriendRequest {
	requesterName := f.RequesterID
	if u, ok := userMap[f.RequesterID]; ok {
		requesterName = u.DisplayName
	}
	addresseeName := f.AddresseeID
	if u, ok := userMap[f.AddresseeID]; ok {
		addresseeName = u.DisplayName
	}
	return &pb.FriendRequest{
		Id:                   f.ID,
		RequesterId:          f.RequesterID,
		RequesterDisplayName: requesterName,
		AddresseeId:          f.AddresseeID,
		AddresseeDisplayName: addresseeName,
		Status:               string(f.Status),
		CreatedAt:            f.CreatedAt,
	}
}

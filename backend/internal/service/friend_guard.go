package service

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage"
)

// validateFriendship checks that each (uid, name) pair is an accepted friend of callerID.
// Empty UIDs and callerID itself are skipped.
func validateFriendship(ctx context.Context, store storage.Store, callerID string, userIDs, displayNames []string) error {
	for i, uid := range userIDs {
		if uid == "" || uid == callerID {
			continue
		}
		ok, err := store.AreFriends(ctx, callerID, uid)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to verify friendship: %w", err))
		}
		if !ok {
			return connect.NewError(connect.CodePermissionDenied,
				fmt.Errorf("user %q is not your friend; only friends can be added", displayNames[i]))
		}
	}
	return nil
}

// validateRegisteredParticipants checks that every participant with a non-empty user_id
// (other than the caller themselves) is an accepted friend of the caller.
// Guests (empty user_id) are always allowed.
func validateRegisteredParticipants(ctx context.Context, store storage.Store, callerID string, participants []models.BillParticipant) error {
	ids := make([]string, len(participants))
	names := make([]string, len(participants))
	for i, p := range participants {
		ids[i], names[i] = p.UserID, p.DisplayName
	}
	return validateFriendship(ctx, store, callerID, ids, names)
}

// validateRegisteredMembers checks that every group member with a non-empty user_id
// (other than the caller themselves) is an accepted friend of the caller.
// Guests (empty user_id) are always allowed.
func validateRegisteredMembers(ctx context.Context, store storage.Store, callerID string, members []models.GroupMember) error {
	ids := make([]string, len(members))
	names := make([]string, len(members))
	for i, m := range members {
		ids[i], names[i] = m.UserID, m.DisplayName
	}
	return validateFriendship(ctx, store, callerID, ids, names)
}

import { apiPost } from './client';
import type {
  ListFriendRequestsRequest,
  ListFriendRequestsResponse,
  ListFriendsResponse,
  RemoveFriendRequest,
  RemoveFriendResponse,
  RespondToFriendRequestRequest,
  RespondToFriendRequestResponse,
  SearchFriendsRequest,
  SearchFriendsResponse,
  SendFriendRequestRequest,
  SendFriendRequestResponse,
} from './types';

const SERVICE = 'FriendService';

export function sendFriendRequest(addresseeId: string): Promise<SendFriendRequestResponse> {
  return apiPost<SendFriendRequestRequest, SendFriendRequestResponse>(
    SERVICE,
    'SendFriendRequest',
    { addresseeId },
  );
}

export function respondToFriendRequest(
  requestId: string,
  accept: boolean,
): Promise<RespondToFriendRequestResponse> {
  return apiPost<RespondToFriendRequestRequest, RespondToFriendRequestResponse>(
    SERVICE,
    'RespondToFriendRequest',
    { requestId, accept },
  );
}

export function listFriends(): Promise<ListFriendsResponse> {
  return apiPost(SERVICE, 'ListFriends', {});
}

export function listFriendRequests(incoming: boolean): Promise<ListFriendRequestsResponse> {
  return apiPost<ListFriendRequestsRequest, ListFriendRequestsResponse>(
    SERVICE,
    'ListFriendRequests',
    { incoming },
  );
}

export function removeFriend(userId: string): Promise<RemoveFriendResponse> {
  return apiPost<RemoveFriendRequest, RemoveFriendResponse>(SERVICE, 'RemoveFriend', { userId });
}

export function searchFriends(query: string): Promise<SearchFriendsResponse> {
  return apiPost<SearchFriendsRequest, SearchFriendsResponse>(SERVICE, 'SearchFriends', {
    query,
  });
}

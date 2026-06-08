// Proto JSON wire types (camelCase).
// Proto3 omits zero/default numeric values from JSON responses — callers should
// fallback with `value ?? 0`. Optional proto fields may simply be absent.

export type Empty = Record<string, never>;

// ── common.proto ──────────────────────────────────────────────────────────

export interface Item {
  description: string;
  amount: number;
  participantIds: string[];
}

export interface PersonItem {
  description: string;
  amount: number;
}

export interface PersonSplit {
  subtotal?: number;
  tax?: number;
  total?: number;
  items?: PersonItem[];
}

// ── bill.proto ────────────────────────────────────────────────────────────

export interface BillParticipant {
  displayName: string;
  userId?: string;
}

export interface BillSummary {
  billId: string;
  title: string;
  total?: number;
  payerId: string;
  createdAt: number;
  participantCount?: number;
  groupName?: string;
  groupId?: string;
}

export interface UserSearchResult {
  userId: string;
  displayName: string;
}

export interface CalculateSplitRequest {
  items: Item[];
  total: number;
  subtotal: number;
  participantIds: string[];
}

export interface CalculateSplitResponse {
  splits: Record<string, PersonSplit>;
  taxAmount?: number;
  subtotal?: number;
}

export interface CreateBillRequest {
  title: string;
  total: number;
  subtotal: number;
  items: Item[];
  participants: BillParticipant[];
  payerId?: string;
  groupId?: string;
}

export interface CreateBillResponse {
  billId: string;
  split: CalculateSplitResponse;
}

export interface GetBillRequest {
  billId: string;
}

export interface GetBillResponse {
  billId: string;
  title: string;
  total?: number;
  subtotal?: number;
  items: Item[];
  participants: BillParticipant[];
  payerId: string;
  groupId?: string;
  createdAt: number;
  split: CalculateSplitResponse;
  groupName?: string;
}

export interface UpdateBillRequest {
  billId: string;
  title: string;
  total: number;
  subtotal: number;
  items: Item[];
  participants: BillParticipant[];
  payerId?: string;
  groupId?: string;
}

export interface UpdateBillResponse {
  billId: string;
  split: CalculateSplitResponse;
}

export interface DeleteBillRequest {
  billId: string;
}

export type DeleteBillResponse = Empty;

export interface ListBillsByGroupRequest {
  groupId: string;
}

export interface ListBillsByGroupResponse {
  bills: BillSummary[];
}

export type ListMyBillsRequest = Empty;

export interface ListMyBillsResponse {
  bills: BillSummary[];
}

export interface SearchUsersRequest {
  query: string;
}

export interface SearchUsersResponse {
  users: UserSearchResult[];
}

// ── group.proto ───────────────────────────────────────────────────────────

export interface GroupMember {
  displayName: string;
  userId?: string;
}

export interface Group {
  id: string;
  name: string;
  members: GroupMember[];
  createdAt: number;
}

export interface MemberBalance {
  userId: string;
  displayName: string;
  netBalance?: number;
  totalPaid?: number;
  totalOwed?: number;
}

export interface DebtEdge {
  fromUserId: string;
  toUserId: string;
  amount?: number;
  fromName: string;
  toName: string;
}

export interface Settlement {
  id: string;
  groupId?: string;
  fromUserId: string;
  toUserId: string;
  amount?: number;
  createdAt: number;
  createdBy: string;
  note: string;
  fromName: string;
  toName: string;
}

export interface PersonGroupBalance {
  groupId: string;
  groupName: string;
  netAmount?: number;
}

export interface PersonBalance {
  displayName: string;
  netAmount?: number;
  groupBalances: PersonGroupBalance[];
  userId?: string;
}

export interface CreateGroupRequest {
  name: string;
  members: GroupMember[];
}

export interface CreateGroupResponse {
  group: Group;
}

export interface GetGroupRequest {
  groupId: string;
}

export interface GetGroupResponse {
  group: Group;
}

export type ListGroupsRequest = Empty;

export interface ListGroupsResponse {
  groups: Group[];
}

export interface UpdateGroupRequest {
  groupId: string;
  name: string;
  members: GroupMember[];
}

export interface UpdateGroupResponse {
  group: Group;
}

export interface DeleteGroupRequest {
  groupId: string;
}

export type DeleteGroupResponse = Empty;

export interface GetGroupBalancesRequest {
  groupId: string;
}

export interface GetGroupBalancesResponse {
  memberBalances: MemberBalance[];
  debtMatrix: DebtEdge[];
}

export interface RecordSettlementRequest {
  groupId: string;
  fromUserId: string;
  toUserId: string;
  amount: number;
  note?: string;
}

export interface RecordSettlementResponse {
  settlement: Settlement;
}

export interface ListSettlementsRequest {
  groupId: string;
}

export interface ListSettlementsResponse {
  settlements: Settlement[];
}

export interface DeleteSettlementRequest {
  settlementId: string;
}

export type DeleteSettlementResponse = Empty;

export type GetMyBalancesRequest = Empty;

export interface GetMyBalancesResponse {
  totalYouOwe?: number;
  totalOwedToYou?: number;
  personBalances: PersonBalance[];
}

export interface SettleUpWithPersonRequest {
  toUserId: string;
}

export interface SettleUpWithPersonResponse {
  settlements: Settlement[];
}

// ── friend.proto ──────────────────────────────────────────────────────────

export interface FriendRequest {
  id: string;
  requesterId: string;
  requesterDisplayName: string;
  addresseeId: string;
  addresseeDisplayName: string;
  status: 'pending' | 'accepted' | 'declined' | string;
  createdAt: number;
}

export interface Friend {
  userId: string;
  displayName: string;
  email: string;
}

export interface FriendSearchResult {
  userId: string;
  displayName: string;
}

export interface SendFriendRequestRequest {
  addresseeId: string;
}

export interface SendFriendRequestResponse {
  request: FriendRequest;
}

export interface RespondToFriendRequestRequest {
  requestId: string;
  accept: boolean;
}

export interface RespondToFriendRequestResponse {
  request: FriendRequest;
}

export type ListFriendsRequest = Empty;

export interface ListFriendsResponse {
  friends: Friend[];
}

export interface ListFriendRequestsRequest {
  incoming: boolean;
}

export interface ListFriendRequestsResponse {
  requests: FriendRequest[];
}

export interface RemoveFriendRequest {
  userId: string;
}

export type RemoveFriendResponse = Empty;

export interface SearchFriendsRequest {
  query: string;
}

export interface SearchFriendsResponse {
  users: FriendSearchResult[];
}

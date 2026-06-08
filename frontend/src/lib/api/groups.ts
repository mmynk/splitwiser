import { apiPost } from './client';
import type {
  CreateGroupRequest,
  CreateGroupResponse,
  DeleteGroupRequest,
  DeleteGroupResponse,
  DeleteSettlementRequest,
  DeleteSettlementResponse,
  GetGroupBalancesRequest,
  GetGroupBalancesResponse,
  GetGroupRequest,
  GetGroupResponse,
  GetMyBalancesResponse,
  ListGroupsResponse,
  ListSettlementsRequest,
  ListSettlementsResponse,
  RecordSettlementRequest,
  RecordSettlementResponse,
  SettleUpWithPersonRequest,
  SettleUpWithPersonResponse,
  UpdateGroupRequest,
  UpdateGroupResponse,
} from './types';

const SERVICE = 'GroupService';

export function createGroup(req: CreateGroupRequest): Promise<CreateGroupResponse> {
  return apiPost(SERVICE, 'CreateGroup', req);
}

export function getGroup(groupId: string): Promise<GetGroupResponse> {
  return apiPost<GetGroupRequest, GetGroupResponse>(SERVICE, 'GetGroup', { groupId });
}

export function listGroups(): Promise<ListGroupsResponse> {
  return apiPost(SERVICE, 'ListGroups', {});
}

export function updateGroup(req: UpdateGroupRequest): Promise<UpdateGroupResponse> {
  return apiPost(SERVICE, 'UpdateGroup', req);
}

export function deleteGroup(groupId: string): Promise<DeleteGroupResponse> {
  return apiPost<DeleteGroupRequest, DeleteGroupResponse>(SERVICE, 'DeleteGroup', { groupId });
}

export function getGroupBalances(groupId: string): Promise<GetGroupBalancesResponse> {
  return apiPost<GetGroupBalancesRequest, GetGroupBalancesResponse>(
    SERVICE,
    'GetGroupBalances',
    { groupId },
  );
}

export function recordSettlement(
  req: RecordSettlementRequest,
): Promise<RecordSettlementResponse> {
  return apiPost(SERVICE, 'RecordSettlement', req);
}

export function listSettlements(groupId: string): Promise<ListSettlementsResponse> {
  return apiPost<ListSettlementsRequest, ListSettlementsResponse>(SERVICE, 'ListSettlements', {
    groupId,
  });
}

export function deleteSettlement(settlementId: string): Promise<DeleteSettlementResponse> {
  return apiPost<DeleteSettlementRequest, DeleteSettlementResponse>(
    SERVICE,
    'DeleteSettlement',
    { settlementId },
  );
}

export function getMyBalances(): Promise<GetMyBalancesResponse> {
  return apiPost(SERVICE, 'GetMyBalances', {});
}

export function settleUpWithPerson(toUserId: string): Promise<SettleUpWithPersonResponse> {
  return apiPost<SettleUpWithPersonRequest, SettleUpWithPersonResponse>(
    SERVICE,
    'SettleUpWithPerson',
    { toUserId },
  );
}

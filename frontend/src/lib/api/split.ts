import { apiPost } from './client';
import type {
  CalculateSplitRequest,
  CalculateSplitResponse,
  CreateBillRequest,
  CreateBillResponse,
  DeleteBillRequest,
  DeleteBillResponse,
  GetBillRequest,
  GetBillResponse,
  ListBillsByGroupRequest,
  ListBillsByGroupResponse,
  ListMyBillsResponse,
  SearchUsersRequest,
  SearchUsersResponse,
  UpdateBillRequest,
  UpdateBillResponse,
} from './types';

const SERVICE = 'SplitService';

export function calculateSplit(req: CalculateSplitRequest): Promise<CalculateSplitResponse> {
  return apiPost(SERVICE, 'CalculateSplit', req);
}

export function createBill(req: CreateBillRequest): Promise<CreateBillResponse> {
  return apiPost(SERVICE, 'CreateBill', req);
}

export function getBill(billId: string): Promise<GetBillResponse> {
  return apiPost<GetBillRequest, GetBillResponse>(SERVICE, 'GetBill', { billId });
}

export function updateBill(req: UpdateBillRequest): Promise<UpdateBillResponse> {
  return apiPost(SERVICE, 'UpdateBill', req);
}

export function deleteBill(billId: string): Promise<DeleteBillResponse> {
  return apiPost<DeleteBillRequest, DeleteBillResponse>(SERVICE, 'DeleteBill', { billId });
}

export function listMyBills(): Promise<ListMyBillsResponse> {
  return apiPost(SERVICE, 'ListMyBills', {});
}

export function listBillsByGroup(groupId: string): Promise<ListBillsByGroupResponse> {
  return apiPost<ListBillsByGroupRequest, ListBillsByGroupResponse>(SERVICE, 'ListBillsByGroup', {
    groupId,
  });
}

export function searchUsers(query: string): Promise<SearchUsersResponse> {
  return apiPost<SearchUsersRequest, SearchUsersResponse>(SERVICE, 'SearchUsers', { query });
}

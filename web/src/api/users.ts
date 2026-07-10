import { api, type PaginationMeta } from "../lib/apiClient";

export interface UserResponse {
  id: string;
  email: string;
  username: string;
  firstName: string;
  lastName: string;
  phoneNumber?: string;
  profilePicture?: string;
  gender?: string;
  status: string;
  roleId: string;
  emailVerified: boolean;
  twoFAEnabled: boolean;
  lastLoginAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateUserRequest {
  email: string;
  username: string;
  password: string;
  firstName: string;
  lastName: string;
  phoneNumber?: string;
  roleId: string;
}

export interface UpdateUserRequest {
  firstName?: string;
  lastName?: string;
  phoneNumber?: string;
  profilePicture?: string;
  gender?: string;
}

export interface ListUsersQuery {
  [key: string]: string | number | boolean | undefined;
  page?: number;
  perPage?: number;
  search?: string;
  sortBy?: string;
  sortDir?: "asc" | "desc";
}

export interface ListResult<T> {
  items: T[];
  meta?: PaginationMeta;
}

export async function listUsers(query: ListUsersQuery = {}): Promise<ListResult<UserResponse>> {
  const res = await api.get<UserResponse[]>("/v1/users", { query });
  return { items: res.data, meta: res.meta };
}

export async function getUser(id: string): Promise<UserResponse> {
  const res = await api.get<UserResponse>(`/v1/users/${encodeURIComponent(id)}`);
  return res.data;
}

export async function createUser(payload: CreateUserRequest): Promise<UserResponse> {
  const res = await api.post<UserResponse>("/v1/users", payload);
  return res.data;
}

export async function updateUser(id: string, payload: UpdateUserRequest): Promise<UserResponse> {
  const res = await api.patch<UserResponse>(`/v1/users/${encodeURIComponent(id)}`, payload);
  return res.data;
}

export async function deleteUser(id: string): Promise<void> {
  await api.delete(`/v1/users/${encodeURIComponent(id)}`);
}

export async function blockUser(id: string, reason: string): Promise<void> {
  await api.post(`/v1/users/${encodeURIComponent(id)}/block`, { reason });
}

export async function unblockUser(id: string): Promise<void> {
  await api.post(`/v1/users/${encodeURIComponent(id)}/unblock`);
}

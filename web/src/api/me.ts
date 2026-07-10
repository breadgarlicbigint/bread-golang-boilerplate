import { api } from "../lib/apiClient";
import type { UserResponse } from "./users";

export interface UpdateProfileRequest {
  firstName?: string;
  lastName?: string;
  phoneNumber?: string;
  profilePicture?: string;
  gender?: string;
}

export interface ChangePasswordRequest {
  oldPassword: string;
  newPassword: string;
}

export async function getMe(): Promise<UserResponse> {
  const res = await api.get<UserResponse>("/v1/me");
  return res.data;
}

export async function updateMe(payload: UpdateProfileRequest): Promise<UserResponse> {
  const res = await api.patch<UserResponse>("/v1/me", payload);
  return res.data;
}

export async function changePassword(payload: ChangePasswordRequest): Promise<void> {
  await api.patch("/v1/me/password", payload);
}

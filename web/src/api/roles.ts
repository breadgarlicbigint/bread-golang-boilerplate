import { api } from "../lib/apiClient";

export interface RoleResponse {
  id: string;
  name: string;
  slug: string;
  description: string;
}

export async function listRoles(): Promise<RoleResponse[]> {
  const res = await api.get<RoleResponse[]>("/v1/roles");
  return res.data;
}

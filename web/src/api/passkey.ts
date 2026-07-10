import { api } from "../lib/apiClient";

// NOTE: modules/passkey/handler returns the go-webauthn library's native
// protocol.CredentialCreation / protocol.CredentialAssertion structs directly
// (not the custom shapes in modules/passkey/dto/passkey.dto.go, which the
// handler never actually uses). Both are shaped as `{ publicKey: {...} }`,
// matching the standard WebAuthn JSON convention that
// @simplewebauthn/browser expects.

export type Attachment = "platform" | "cross-platform";

export interface PasskeyResponse {
  id: string;
  friendlyName: string;
  attachment: string;
  isBiometric: boolean;
  backedUp: boolean;
  lastUsedAt?: string;
  createdAt: string;
}

export async function listPasskeys(): Promise<PasskeyResponse[]> {
  const res = await api.get<PasskeyResponse[]>("/v1/me/passkeys");
  return res.data;
}

export async function deletePasskey(id: string): Promise<void> {
  await api.delete(`/v1/me/passkeys/${encodeURIComponent(id)}`);
}

/** Returns the raw `{ publicKey: PublicKeyCredentialCreationOptionsJSON }` payload. */
export async function beginRegistration(attachment: Attachment): Promise<{ publicKey: unknown }> {
  const res = await api.post<{ publicKey: unknown }>("/v1/me/passkeys/register/begin", undefined, {
    query: { attachment },
  });
  return res.data;
}

export async function finishRegistration(
  attachment: Attachment,
  friendlyName: string,
  credential: unknown,
): Promise<PasskeyResponse> {
  const res = await api.post<PasskeyResponse>(
    "/v1/me/passkeys/register/finish",
    { sessionToken: crypto.randomUUID(), friendlyName, response: credential },
    { query: { attachment } },
  );
  return res.data;
}

/** Returns the raw `{ options: { publicKey: ... }, sessionToken }` payload. */
export async function beginDiscoverableLogin(): Promise<{ options: { publicKey: unknown }; sessionToken: string }> {
  const res = await api.post<{ options: { publicKey: unknown }; sessionToken: string }>(
    "/v1/auth/passkey/login/begin",
    undefined,
    { auth: false },
  );
  return res.data;
}

export async function finishDiscoverableLogin(sessionToken: string, credential: unknown): Promise<unknown> {
  const res = await api.post<unknown>(
    "/v1/auth/passkey/login/finish",
    { sessionToken, response: credential },
    { auth: false },
  );
  return res.data;
}

export async function beginIdentifiedLogin(email: string): Promise<unknown> {
  const res = await api.post<unknown>("/v1/auth/passkey/identified/begin", { email }, { auth: false });
  return res.data;
}

export async function finishIdentifiedLogin(email: string, credential: unknown): Promise<unknown> {
  const res = await api.post<unknown>(
    "/v1/auth/passkey/identified/finish",
    { email, response: credential },
    { auth: false },
  );
  return res.data;
}

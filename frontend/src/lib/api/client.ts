import { get } from 'svelte/store';
import { token, logout } from '$lib/stores/auth';

const BASE_PATH = '/splitwiser.v1.';

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

export async function apiPost<TReq = unknown, TRes = unknown>(
  service: string,
  method: string,
  body: TReq,
  options: { auth?: boolean } = {},
): Promise<TRes> {
  const useAuth = options.auth !== false;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (useAuth) {
    const t = get(token);
    if (t) headers['Authorization'] = `Bearer ${t}`;
  }

  const response = await fetch(`${BASE_PATH}${service}/${method}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body ?? {}),
  });

  if (!response.ok) {
    const errBody: unknown = await response.json().catch(() => ({}));
    const message =
      typeof errBody === 'object' && errBody && 'message' in errBody && typeof errBody.message === 'string'
        ? errBody.message
        : `Request failed (${response.status})`;

    // Only auto-logout on 401 from authenticated requests — a failed login attempt
    // also returns 401 and must not clobber the user's form input.
    if (response.status === 401 && useAuth) {
      logout();
      throw new ApiError('Session expired. Please login again.', 401);
    }
    throw new ApiError(message, response.status);
  }

  return (await response.json()) as TRes;
}

import { apiPost } from './client';
import type { AuthUser } from '$lib/stores/auth';

interface LoginRequest {
  email: string;
  password: string;
}

interface RegisterRequest {
  email: string;
  password: string;
  displayName: string;
}

interface AuthResponse {
  token: string;
  user: AuthUser;
}

export function loginApi(email: string, password: string): Promise<AuthResponse> {
  return apiPost<LoginRequest, AuthResponse>(
    'AuthService',
    'Login',
    { email, password },
    { auth: false },
  );
}

export function registerApi(
  email: string,
  password: string,
  displayName: string,
): Promise<AuthResponse> {
  return apiPost<RegisterRequest, AuthResponse>(
    'AuthService',
    'Register',
    { email, password, displayName },
    { auth: false },
  );
}

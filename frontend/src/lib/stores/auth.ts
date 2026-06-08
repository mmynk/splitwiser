import { writable, get } from 'svelte/store';
import { replace } from 'svelte-spa-router';

export interface AuthUser {
  id: string;
  email: string;
  displayName: string;
}

const TOKEN_KEY = 'auth_token';
const USER_KEY = 'user';

function readUser(): AuthUser | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    return null;
  }
}

export const token = writable<string | null>(localStorage.getItem(TOKEN_KEY));
export const currentUser = writable<AuthUser | null>(readUser());

export function isLoggedIn(): boolean {
  return !!get(token);
}

export function login(newToken: string, user: AuthUser): void {
  localStorage.setItem(TOKEN_KEY, newToken);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  token.set(newToken);
  currentUser.set(user);
}

export function logout(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  token.set(null);
  currentUser.set(null);
  if (location.hash.replace(/^#/, '') !== '/login') {
    replace('/login');
  }
}

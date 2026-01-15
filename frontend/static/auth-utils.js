// Auth utilities - shared authentication helpers

const API_BASE = 'http://localhost:8080';

// Get auth token from localStorage
export function getAuthToken() {
  return localStorage.getItem('auth_token');
}

// Get current user from localStorage
export function getCurrentUser() {
  const userJson = localStorage.getItem('user');
  return userJson ? JSON.parse(userJson) : null;
}

// Check if user is logged in
export function isLoggedIn() {
  return !!getAuthToken();
}

// Logout - clear localStorage and redirect to login
export function logout() {
  localStorage.removeItem('auth_token');
  localStorage.removeItem('user');
  window.location.href = '/login.html';
}

// Require authentication - redirect to login if not logged in
// Call this at the top of protected pages
export function requireAuth() {
  if (!isLoggedIn()) {
    window.location.href = '/login.html';
    return false;
  }
  return true;
}

// Make an authenticated API request
export async function authenticatedFetch(endpoint, options = {}) {
  const token = getAuthToken();

  if (!token) {
    throw new Error('Not authenticated');
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
    ...options.headers,
  };

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  // Handle 401 Unauthenticated - redirect to login
  if (response.status === 401) {
    logout();
    throw new Error('Session expired. Please login again.');
  }

  return response;
}

// Create request body for Connect RPC (already JSON, but normalized)
export function createConnectRequest(data) {
  return JSON.stringify(data);
}

// Parse Connect RPC response
export async function parseConnectResponse(response) {
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || `Request failed with status ${response.status}`);
  }
  return await response.json();
}

// Show user info in header
export function displayUserInfo(containerId = 'user-info') {
  const container = document.getElementById(containerId);
  if (!container) return;

  const user = getCurrentUser();
  if (!user) {
    container.innerHTML = '';
    return;
  }

  container.innerHTML = `
    <div style="display: flex; align-items: center; gap: 1rem;">
      <span>👤 ${user.display_name || user.email}</span>
      <button type="button" onclick="window.authUtils.logout()" class="secondary outline" style="margin: 0; padding: 0.5rem 1rem;">
        Logout
      </button>
    </div>
  `;
}

// Make auth utils available globally for inline onclick handlers
if (typeof window !== 'undefined') {
  window.authUtils = {
    logout,
    displayUserInfo,
  };
}

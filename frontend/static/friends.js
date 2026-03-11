// Splitwiser - Friends Management

import { requireAuth, authenticatedFetch, getCurrentUser } from './auth-utils.js';

requireAuth();

// State
let friends = [];           // Friend objects {userId, displayName, email}
let incomingRequests = [];  // FriendRequest objects {id, requesterUserId, requesterDisplayName, ...}
let outgoingRequests = [];  // FriendRequest objects (to detect pending in search results)

// DOM Elements
const incomingSection = document.getElementById('incoming-section');
const incomingList = document.getElementById('incoming-list');
const friendsList = document.getElementById('friends-list');
const searchInput = document.getElementById('friend-search');
const searchResults = document.getElementById('search-results');
const errorEl = document.getElementById('error');

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
  renderUserInfo();
  await loadAll();
  attachSearch();
});

function renderUserInfo() {
  const user = getCurrentUser();
  const el = document.getElementById('user-info');
  if (!el || !user) return;
  const name = user.display_name || user.displayName || user.email || '';
  el.textContent = name;
}

// Load friends + both request lists in parallel
async function loadAll() {
  hideError();
  try {
    const [friendsResp, incomingResp, outgoingResp] = await Promise.all([
      authenticatedFetch('/splitwiser.v1.FriendService/ListFriends', { method: 'POST', body: JSON.stringify({}) }),
      authenticatedFetch('/splitwiser.v1.FriendService/ListFriendRequests', { method: 'POST', body: JSON.stringify({ incoming: true }) }),
      authenticatedFetch('/splitwiser.v1.FriendService/ListFriendRequests', { method: 'POST', body: JSON.stringify({ incoming: false }) }),
    ]);

    if (!friendsResp.ok) throw new Error((await friendsResp.json().catch(() => ({}))).message || 'Failed to load friends');
    if (!incomingResp.ok) throw new Error((await incomingResp.json().catch(() => ({}))).message || 'Failed to load requests');
    if (!outgoingResp.ok) throw new Error((await outgoingResp.json().catch(() => ({}))).message || 'Failed to load requests');

    friends = (await friendsResp.json()).friends || [];
    incomingRequests = (await incomingResp.json()).requests || [];
    outgoingRequests = (await outgoingResp.json()).requests || [];

    renderIncoming();
    renderFriends();
  } catch (err) {
    showError(err.message);
    friendsList.innerHTML = '<p>Failed to load friends.</p>';
  }
}

// Incoming Requests
function renderIncoming() {
  if (incomingRequests.length === 0) {
    incomingSection.classList.add('hidden');
    return;
  }
  incomingSection.classList.remove('hidden');
  incomingList.innerHTML = incomingRequests.map(req => `
    <div class="result-card" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: var(--spacing-sm);">
      <div>
        <strong>${escapeHtml(req.requesterDisplayName || 'Unknown')}</strong>
        ${req.requesterEmail ? `<small style="color: var(--pico-muted-color); margin-left: 0.5rem;">${escapeHtml(req.requesterEmail)}</small>` : ''}
      </div>
      <div style="display: flex; gap: var(--spacing-sm);">
        <button type="button" class="outline" data-action="accept" data-request-id="${escapeHtml(req.id)}">Accept</button>
        <button type="button" class="remove-btn" data-action="decline" data-request-id="${escapeHtml(req.id)}">Decline</button>
      </div>
    </div>
  `).join('');

  incomingList.querySelectorAll('[data-action]').forEach(btn => {
    btn.addEventListener('click', () => respondToRequest(btn.dataset.requestId, btn.dataset.action === 'accept'));
  });
}

async function respondToRequest(requestId, accept) {
  try {
    const resp = await authenticatedFetch('/splitwiser.v1.FriendService/RespondToFriendRequest', {
      method: 'POST',
      body: JSON.stringify({ requestId, accept })
    });
    if (!resp.ok) throw new Error((await resp.json().catch(() => ({}))).message || 'Failed to respond');
    await loadAll();
  } catch (err) {
    showError(err.message);
  }
}

// My Friends
function renderFriends() {
  if (friends.length === 0) {
    friendsList.innerHTML = '<p>No friends yet. Search above to add people.</p>';
    return;
  }
  friendsList.innerHTML = friends.map(f => `
    <div class="result-card" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: var(--spacing-sm);">
      <div>
        <strong>${escapeHtml(f.displayName || 'Unknown')}</strong>
        ${f.email ? `<small style="color: var(--pico-muted-color); margin-left: 0.5rem;">${escapeHtml(f.email)}</small>` : ''}
      </div>
      <button type="button" class="remove-btn" data-action="unfriend" data-user-id="${escapeHtml(f.userId)}" data-display-name="${escapeHtml(f.displayName || '')}">Unfriend</button>
    </div>
  `).join('');

  friendsList.querySelectorAll('[data-action="unfriend"]').forEach(btn => {
    btn.addEventListener('click', () => unfriend(btn.dataset.userId, btn.dataset.displayName));
  });
}

async function unfriend(userId, displayName) {
  if (!confirm(`Remove ${displayName} from your friends?`)) return;
  try {
    const resp = await authenticatedFetch('/splitwiser.v1.FriendService/RemoveFriend', {
      method: 'POST',
      body: JSON.stringify({ userId })
    });
    if (!resp.ok) throw new Error((await resp.json().catch(() => ({}))).message || 'Failed to unfriend');
    await loadAll();
    // Re-render search results in case this user was displayed there
    if (searchInput.value.trim().length >= 2) triggerSearch(searchInput.value.trim());
  } catch (err) {
    showError(err.message);
  }
}

// Search
let _searchTimeout = null;

function attachSearch() {
  searchInput.addEventListener('input', (e) => {
    const query = e.target.value.trim();
    clearTimeout(_searchTimeout);
    if (query.length < 2) {
      searchResults.innerHTML = '';
      return;
    }
    _searchTimeout = setTimeout(() => triggerSearch(query), 300);
  });
}

async function triggerSearch(query) {
  try {
    const resp = await authenticatedFetch('/splitwiser.v1.SplitService/SearchUsers', {
      method: 'POST',
      body: JSON.stringify({ query, includeNonFriends: true })
    });
    if (!resp.ok) { searchResults.innerHTML = ''; return; }
    const data = await resp.json();
    renderSearchResults(data.users || []);
  } catch {
    searchResults.innerHTML = '';
  }
}

function classifyUser(user) {
  if (friends.some(f => f.userId === user.userId)) return 'friend';
  if (outgoingRequests.some(r => r.addresseeUserId === user.userId)) return 'pending-outgoing';
  return 'stranger';
}

function renderSearchResults(users) {
  if (users.length === 0) {
    searchResults.innerHTML = '<p style="color: var(--pico-muted-color);">No users found.</p>';
    return;
  }

  searchResults.innerHTML = users.map(u => {
    const status = classifyUser(u);
    let actionBtn;
    if (status === 'friend') {
      actionBtn = `<button type="button" disabled style="opacity: 0.6;">Friends ✓</button>`;
    } else if (status === 'pending-outgoing') {
      actionBtn = `<button type="button" disabled style="opacity: 0.6;">Pending...</button>`;
    } else {
      actionBtn = `<button type="button" class="outline" data-action="add-friend" data-user-id="${escapeHtml(u.userId)}">Add Friend</button>`;
    }
    return `
      <div class="result-card" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: var(--spacing-sm);">
        <div>
          <strong>${escapeHtml(u.displayName || '')}</strong>
          ${u.email ? `<small style="color: var(--pico-muted-color); margin-left: 0.5rem;">${escapeHtml(u.email)}</small>` : ''}
        </div>
        ${actionBtn}
      </div>`;
  }).join('');

  searchResults.querySelectorAll('[data-action="add-friend"]').forEach(btn => {
    btn.addEventListener('click', () => sendFriendRequest(btn.dataset.userId, btn));
  });
}

async function sendFriendRequest(addresseeId, btn) {
  btn.disabled = true;
  btn.textContent = 'Sending...';
  try {
    const resp = await authenticatedFetch('/splitwiser.v1.FriendService/SendFriendRequest', {
      method: 'POST',
      body: JSON.stringify({ addresseeId })
    });
    if (!resp.ok) throw new Error((await resp.json().catch(() => ({}))).message || 'Failed to send request');
    // Reload outgoing requests so classifyUser reflects the new pending state
    const outResp = await authenticatedFetch('/splitwiser.v1.FriendService/ListFriendRequests', {
      method: 'POST',
      body: JSON.stringify({ incoming: false })
    });
    outgoingRequests = outResp.ok ? ((await outResp.json()).requests || []) : outgoingRequests;
    btn.textContent = 'Pending...';
  } catch (err) {
    btn.disabled = false;
    btn.textContent = 'Add Friend';
    showError(err.message);
  }
}

function showError(message) {
  errorEl.textContent = message;
  errorEl.classList.remove('hidden');
}

function hideError() {
  errorEl.classList.add('hidden');
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = String(text || '');
  return div.innerHTML;
}

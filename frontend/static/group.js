// Splitwiser - Group Detail Logic

import { requireAuth, displayUserInfo, authenticatedFetch, getCurrentUser } from './auth-utils.js';

// Require authentication
requireAuth();

const API_BASE = 'http://localhost:8080';
let currentUser = null;

const groupId = new URLSearchParams(window.location.search).get('id');
const errorEl = document.getElementById('error');

let currentGroup = null;
let balances = null;

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
  if (!groupId) {
    showError('No group ID provided');
    return;
  }

  await loadGroup();
  await loadBalances();
  await loadBills();

  // Set up "New Bill" link to pre-select this group
  document.getElementById('new-bill-link').href = `/?group=${groupId}`;
  document.getElementById('create-first-bill').href = `/?group=${groupId}`;
});

// View toggle
document.getElementById('total-view-btn').addEventListener('click', showTotalView);
document.getElementById('detailed-view-btn').addEventListener('click', showDetailedView);

async function loadGroup() {
  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/GetGroup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ groupId })
    });

    if (!response.ok) throw new Error('Failed to load group');

    const data = await response.json();
    currentGroup = data;

    document.getElementById('group-name').textContent = data.name;
    document.getElementById('group-members').textContent =
      `Members: ${(data.members || []).join(', ')}`;
  } catch (err) {
    showError('Failed to load group: ' + err.message);
  }
}

async function loadBalances() {
  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/GetGroupBalances`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ groupId })
    });

    if (!response.ok) throw new Error('Failed to load balances');

    balances = await response.json();

    if (!balances.memberBalances || balances.memberBalances.length === 0) {
      document.getElementById('no-bills-message').classList.remove('hidden');
      document.getElementById('balances-content').classList.add('hidden');
      return;
    }

    renderBalances();
  } catch (err) {
    showError('Failed to load balances: ' + err.message);
  }
}

function renderBalances() {
  // Render total balances
  const grid = document.querySelector('#total-balances .balance-grid');
  grid.innerHTML = balances.memberBalances
    .map(bal => {
      const netClass = bal.netBalance > 0 ? 'positive' : bal.netBalance < 0 ? 'negative' : 'neutral';
      const sign = bal.netBalance > 0 ? '+' : '';

      return `
        <article class="balance-card ${netClass}">
          <h3>${escapeHtml(bal.memberName)}</h3>
          <div class="balance-amount">${sign}$${Math.abs(bal.netBalance).toFixed(2)}</div>
          <div class="balance-details">
            <small>Paid: $${bal.totalPaid.toFixed(2)}<br>
            Owes: $${bal.totalOwed.toFixed(2)}</small>
          </div>
        </article>
      `;
    })
    .join('');

  // Render debt matrix
  const tbody = document.getElementById('debt-table-body');
  if (!balances.debtMatrix || balances.debtMatrix.length === 0) {
    tbody.innerHTML = '';
    document.getElementById('no-debts-message').classList.remove('hidden');
  } else {
    document.getElementById('no-debts-message').classList.add('hidden');
    tbody.innerHTML = balances.debtMatrix
      .map(debt => `
        <tr>
          <td>${escapeHtml(debt.from)}</td>
          <td>${escapeHtml(debt.to)}</td>
          <td>$${debt.amount.toFixed(2)}</td>
        </tr>
      `)
      .join('');
  }
}

async function loadBills() {
  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/ListBillsByGroup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ groupId })
    });

    if (!response.ok) throw new Error('Failed to load bills');

    const data = await response.json();
    const bills = data.bills || [];

    const tbody = document.getElementById('bills-table-body');
    tbody.innerHTML = bills
      .map(bill => {
        const date = new Date(bill.createdAt * 1000).toLocaleDateString();
        return `
          <tr>
            <td><a href="/bill.html?id=${bill.billId}">${escapeHtml(bill.title)}</a></td>
            <td>$${bill.total.toFixed(2)}</td>
            <td>${bill.payerId ? escapeHtml(bill.payerId) : '<em>Not recorded</em>'}</td>
            <td>${date}</td>
            <td>
              <button type="button" class="secondary outline" style="font-size: 0.85em; padding: 0.25em 0.75em;" onclick="deleteBill('${bill.billId}', '${escapeHtml(bill.title)}')">Delete</button>
            </td>
          </tr>
        `;
      })
      .join('');
  } catch (err) {
    showError('Failed to load bills: ' + err.message);
  }
}

async function deleteBill(billId, billTitle) {
  if (!confirm(`Delete "${billTitle}"? This cannot be undone.`)) {
    return;
  }

  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/DeleteBill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ billId })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to delete bill');
    }

    // Reload the page to refresh bills and balances
    await loadBalances();
    await loadBills();
  } catch (err) {
    showError('Failed to delete bill: ' + err.message);
  }
}

function showTotalView() {
  document.getElementById('total-balances').classList.remove('hidden');
  document.getElementById('debt-matrix').classList.add('hidden');
  document.getElementById('total-view-btn').classList.remove('outline');
  document.getElementById('total-view-btn').classList.add('contrast');
  document.getElementById('detailed-view-btn').classList.remove('contrast');
  document.getElementById('detailed-view-btn').classList.add('outline');
}

function showDetailedView() {
  document.getElementById('total-balances').classList.add('hidden');
  document.getElementById('debt-matrix').classList.remove('hidden');
  document.getElementById('detailed-view-btn').classList.remove('outline');
  document.getElementById('detailed-view-btn').classList.add('contrast');
  document.getElementById('total-view-btn').classList.remove('contrast');
  document.getElementById('total-view-btn').classList.add('outline');
}

function showError(msg) {
  errorEl.textContent = msg;
  errorEl.classList.remove('hidden');
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

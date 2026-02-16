// Splitwiser - Group Detail Logic

import { requireAuth, displayUserInfo, authenticatedFetch, getCurrentUser } from './auth-utils.js';

// Require authentication
requireAuth();

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
  await loadSettlements();
  await loadBills();

  // Set up "New Bill" link to pre-select this group
  document.getElementById('new-bill-link').href = `/?group=${groupId}`;
  document.getElementById('create-first-bill').href = `/?group=${groupId}`;

  // Set up settlement button and form
  document.getElementById('record-settlement-btn').addEventListener('click', openSettlementDialog);
  document.getElementById('settlement-form').addEventListener('submit', handleSettlementSubmit);
});

// View toggle
document.getElementById('total-view-btn').addEventListener('click', showTotalView);
document.getElementById('detailed-view-btn').addEventListener('click', showDetailedView);

async function loadGroup() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/GetGroup', {
      method: 'POST',
      body: JSON.stringify({ groupId })
    });

    if (!response.ok) throw new Error('Failed to load group');

    const data = await response.json();
    currentGroup = data.group || data;

    document.getElementById('group-name').textContent = currentGroup.name;
    document.getElementById('group-members').textContent =
      `Members: ${(currentGroup.memberIds || []).join(', ')}`;
  } catch (err) {
    showError('Failed to load group: ' + err.message);
  }
}

async function loadBalances() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/GetGroupBalances', {
      method: 'POST',
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
      const netBalance = bal.netBalance || 0;
      const totalPaid = bal.totalPaid || 0;
      const totalOwed = bal.totalOwed || 0;
      const netClass = netBalance > 0 ? 'positive' : netBalance < 0 ? 'negative' : 'neutral';
      const sign = netBalance > 0 ? '+' : '';

      return `
        <article class="balance-card ${netClass}">
          <h3>${escapeHtml(bal.displayName || bal.userId || 'Unknown')}</h3>
          <div class="balance-amount">${sign}$${Math.abs(netBalance).toFixed(2)}</div>
          <div class="balance-details">
            <small>Paid: $${totalPaid.toFixed(2)}<br>
            Owes: $${totalOwed.toFixed(2)}</small>
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
          <td>${escapeHtml(debt.fromName || debt.fromUserId || '')}</td>
          <td>${escapeHtml(debt.toName || debt.toUserId || '')}</td>
          <td>$${(debt.amount || 0).toFixed(2)}</td>
        </tr>
      `)
      .join('');
  }
}

async function loadBills() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.SplitService/ListBillsByGroup', {
      method: 'POST',
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
    const response = await authenticatedFetch('/splitwiser.v1.SplitService/DeleteBill', {
      method: 'POST',
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

// Settlement functions
async function loadSettlements() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/ListSettlements', {
      method: 'POST',
      body: JSON.stringify({ groupId })
    });

    if (!response.ok) throw new Error('Failed to load settlements');

    const data = await response.json();
    const settlements = data.settlements || [];

    const table = document.getElementById('settlements-table');
    const noSettlementsMsg = document.getElementById('no-settlements-message');

    if (settlements.length === 0) {
      table.classList.add('hidden');
      noSettlementsMsg.classList.remove('hidden');
      return;
    }

    table.classList.remove('hidden');
    noSettlementsMsg.classList.add('hidden');

    const tbody = document.getElementById('settlements-table-body');
    tbody.innerHTML = settlements
      .map(settlement => {
        const date = new Date(settlement.createdAt * 1000).toLocaleDateString();
        return `
          <tr>
            <td>${escapeHtml(settlement.fromName || settlement.fromUserId)}</td>
            <td>${escapeHtml(settlement.toName || settlement.toUserId)}</td>
            <td>$${settlement.amount.toFixed(2)}</td>
            <td>${settlement.note ? escapeHtml(settlement.note) : '<em>-</em>'}</td>
            <td>${date}</td>
            <td>
              <button type="button" class="secondary outline" style="font-size: 0.85em; padding: 0.25em 0.75em;" onclick="deleteSettlement('${settlement.id}')">Delete</button>
            </td>
          </tr>
        `;
      })
      .join('');
  } catch (err) {
    showError('Failed to load settlements: ' + err.message);
  }
}

function openSettlementDialog() {
  if (!currentGroup || !currentGroup.memberIds) {
    showError('Group data not loaded');
    return;
  }

  const fromSelect = document.getElementById('from-user');
  const toSelect = document.getElementById('to-user');

  // Clear and populate dropdowns with group members
  fromSelect.innerHTML = '<option value="">Select payer...</option>';
  toSelect.innerHTML = '<option value="">Select recipient...</option>';

  for (const memberId of currentGroup.memberIds) {
    fromSelect.innerHTML += `<option value="${escapeHtml(memberId)}">${escapeHtml(memberId)}</option>`;
    toSelect.innerHTML += `<option value="${escapeHtml(memberId)}">${escapeHtml(memberId)}</option>`;
  }

  // Clear form
  document.getElementById('settlement-amount').value = '';
  document.getElementById('settlement-note').value = '';

  document.getElementById('settlement-dialog').showModal();
}

function closeSettlementDialog() {
  document.getElementById('settlement-dialog').close();
}

async function handleSettlementSubmit(event) {
  event.preventDefault();

  const fromUserId = document.getElementById('from-user').value;
  const toUserId = document.getElementById('to-user').value;
  const amount = parseFloat(document.getElementById('settlement-amount').value);
  const note = document.getElementById('settlement-note').value;

  if (!fromUserId || !toUserId) {
    showError('Please select both payer and recipient');
    return;
  }

  if (fromUserId === toUserId) {
    showError('Payer and recipient must be different');
    return;
  }

  if (isNaN(amount) || amount <= 0) {
    showError('Please enter a valid amount');
    return;
  }

  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/RecordSettlement', {
      method: 'POST',
      body: JSON.stringify({
        groupId,
        fromUserId,
        toUserId,
        amount,
        note
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to record settlement');
    }

    closeSettlementDialog();

    // Refresh balances and settlements
    await loadBalances();
    await loadSettlements();
  } catch (err) {
    showError('Failed to record settlement: ' + err.message);
  }
}

async function deleteSettlement(settlementId) {
  if (!confirm('Delete this settlement? This cannot be undone.')) {
    return;
  }

  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/DeleteSettlement', {
      method: 'POST',
      body: JSON.stringify({ settlementId })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to delete settlement');
    }

    // Refresh balances and settlements
    await loadBalances();
    await loadSettlements();
  } catch (err) {
    showError('Failed to delete settlement: ' + err.message);
  }
}

// Make functions available globally for onclick handlers
window.deleteBill = deleteBill;
window.deleteSettlement = deleteSettlement;
window.closeSettlementDialog = closeSettlementDialog;

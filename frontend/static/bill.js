// Splitwiser - Bill View Logic

import { requireAuth, displayUserInfo, authenticatedFetch, getCurrentUser } from './auth-utils.js';

// Require authentication
requireAuth();

const API_BASE = 'http://localhost:8080';
let currentUser = null;

// State
let currentBill = null;
let editForm = null;
let isEditMode = false;

// DOM Elements
const loadingEl = document.getElementById('loading');
const errorStateEl = document.getElementById('error-state');
const errorMessageEl = document.getElementById('error-message');
const billContentEl = document.getElementById('bill-content');
const billTitleEl = document.getElementById('bill-title');
const billDateEl = document.getElementById('bill-date');
const groupDisplayEl = document.getElementById('group-display');
const groupLinkEl = document.getElementById('group-link');
const payerDisplayEl = document.getElementById('payer-display');
const payerNameEl = document.getElementById('payer-name');
const shareUrlEl = document.getElementById('share-url');
const copyBtn = document.getElementById('copy-btn');
const summarySubtotalEl = document.getElementById('summary-subtotal');
const summaryTaxEl = document.getElementById('summary-tax');
const summaryTotalEl = document.getElementById('summary-total');
const itemsSectionEl = document.getElementById('items-section');
const itemsListEl = document.getElementById('items-list');
const splitsGridEl = document.getElementById('splits-grid');
const participantsListEl = document.getElementById('participants-list');

// Edit mode elements
const editBtn = document.getElementById('edit-btn');
const deleteBtn = document.getElementById('delete-btn');
const editFormEl = document.getElementById('edit-form');
const viewModeEl = document.getElementById('view-mode');
const saveEditBtn = document.getElementById('save-edit-btn');
const cancelEditBtn = document.getElementById('cancel-edit-btn');
const editErrorEl = document.getElementById('edit-error');
const addParticipantBtn = document.getElementById('edit-add-participant');
const addItemBtn = document.getElementById('edit-add-item');
const editTitleInput = document.getElementById('edit-title');
const editPayerSelect = document.getElementById('edit-payer-select');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  const billId = getBillId();
  if (!billId) {
    showError('No bill ID provided.');
    return;
  }

  loadBill(billId);

  // Set share URL
  shareUrlEl.value = window.location.href;

  // Initialize edit form handler
  editForm = new BillForm();
  editForm.init({
    participantsList: document.getElementById('edit-participants-list'),
    itemsList: document.getElementById('edit-items-list'),
    totalInput: document.getElementById('edit-total'),
    subtotalInput: document.getElementById('edit-subtotal'),
    taxAmountEl: document.getElementById('edit-tax-amount')
  });
});

// Copy button
copyBtn.addEventListener('click', async () => {
  try {
    await navigator.clipboard.writeText(shareUrlEl.value);
    copyBtn.textContent = 'Copied!';
    setTimeout(() => {
      copyBtn.textContent = 'Copy';
    }, 2000);
  } catch (err) {
    // Fallback for older browsers
    shareUrlEl.select();
    document.execCommand('copy');
    copyBtn.textContent = 'Copied!';
    setTimeout(() => {
      copyBtn.textContent = 'Copy';
    }, 2000);
  }
});

// Edit mode handlers
editBtn.addEventListener('click', () => {
  enterEditMode();
});

deleteBtn.addEventListener('click', async () => {
  await deleteBill();
});

cancelEditBtn.addEventListener('click', () => {
  exitEditMode();
});

addParticipantBtn.addEventListener('click', () => {
  editForm.addParticipant('');
});

addItemBtn.addEventListener('click', () => {
  editForm.addItem();
});

editFormEl.addEventListener('submit', async (e) => {
  e.preventDefault();
  await saveChanges();
});

// Get bill ID from URL
function getBillId() {
  const params = new URLSearchParams(window.location.search);
  return params.get('id');
}

// Load bill from API
async function loadBill(billId) {
  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/GetBill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ billId })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to load bill');
    }

    const data = await response.json();
    currentBill = data;
    displayBill(data);
  } catch (err) {
    showError(err.message);
  }
}

// Display bill data
function displayBill(bill) {
  loadingEl.classList.add('hidden');
  billContentEl.classList.remove('hidden');

  // Title and date
  billTitleEl.textContent = bill.title || 'Bill';
  if (bill.createdAt) {
    const date = new Date(bill.createdAt * 1000);
    billDateEl.textContent = date.toLocaleDateString('en-US', {
      month: 'long',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit'
    });
  }

  // Group
  if (bill.groupId) {
    groupDisplayEl.classList.remove('hidden');
    groupLinkEl.href = `/group.html?id=${bill.groupId}`;
    groupLinkEl.textContent = bill.groupName || 'View Group';
  } else {
    groupDisplayEl.classList.add('hidden');
  }

  // Payer
  if (bill.payerId) {
    payerDisplayEl.classList.remove('hidden');
    payerNameEl.textContent = bill.payerId;
  } else {
    payerDisplayEl.classList.add('hidden');
  }

  // Summary
  const total = bill.total || 0;
  const subtotal = bill.subtotal || total;
  const tax = total - subtotal;

  summarySubtotalEl.textContent = `$${subtotal.toFixed(2)}`;
  summaryTaxEl.textContent = `$${tax.toFixed(2)}`;
  summaryTotalEl.textContent = `$${total.toFixed(2)}`;

  // Items
  const items = bill.items || [];
  if (items.length > 0) {
    itemsSectionEl.classList.remove('hidden');
    itemsListEl.innerHTML = items.map(item => `
      <div class="item">
        <div class="item-info">
          <div class="description">${escapeHtml(item.description || 'Item')}</div>
          <div class="assigned">${(item.participants || []).map(escapeHtml).join(', ')}</div>
        </div>
        <div class="item-amount">$${(item.amount || 0).toFixed(2)}</div>
      </div>
    `).join('');
  } else {
    itemsSectionEl.classList.add('hidden');
  }

  // Splits
  const splits = bill.split?.splits || {};
  const participants = bill.participants || [];

  splitsGridEl.innerHTML = participants.map(name => {
    const rawSplit = splits[name] || {};
    const split = {
      subtotal: rawSplit.subtotal || 0,
      tax: rawSplit.tax || 0,
      total: rawSplit.total || 0,
      items: rawSplit.items || []
    };
    const personItems = split.items;
    return `
      <div class="result-card">
        <h3>${escapeHtml(name)}</h3>
        <div class="total">$${split.total.toFixed(2)}</div>
        ${personItems.length > 0 ? `
          <div class="items-breakdown">
            ${personItems.map(item => `
              <div class="item-line">
                <span>${escapeHtml(item.description)}</span>
                <span>$${item.amount.toFixed(2)}</span>
              </div>
            `).join('')}
          </div>
        ` : ''}
        <div class="breakdown">
          Subtotal: $${split.subtotal.toFixed(2)}<br>
          Tax: $${split.tax.toFixed(2)}
        </div>
      </div>
    `;
  }).join('');

  // Participants list
  participantsListEl.textContent = participants.join(', ');
}

// Enter edit mode
function enterEditMode() {
  if (!currentBill) return;

  isEditMode = true;
  editFormEl.classList.remove('hidden');
  viewModeEl.classList.add('hidden');
  editBtn.classList.add('hidden');

  // Load current bill data into form
  editForm.loadBill(currentBill);

  // Load title
  editTitleInput.value = currentBill.title || '';

  // Populate payer dropdown
  const participants = currentBill.participants || [];
  editPayerSelect.innerHTML = '<option value="">Not recorded</option>' +
    participants.map(p =>
      `<option value="${escapeHtml(p)}" ${p === currentBill.payerId ? 'selected' : ''}>
        ${escapeHtml(p)}
      </option>`
    ).join('');
}

// Exit edit mode
function exitEditMode() {
  isEditMode = false;
  editFormEl.classList.add('hidden');
  viewModeEl.classList.remove('hidden');
  editBtn.classList.remove('hidden');
  hideEditError();
}

// Save changes
async function saveChanges() {
  hideEditError();

  const validation = editForm.validate();
  if (!validation.valid) {
    showEditError(validation.error);
    return;
  }

  const data = editForm.getData();
  const billId = getBillId();

  const request = {
    billId,
    title: editTitleInput.value.trim() || '',
    payerId: editPayerSelect.value || undefined,
    ...data
  };

  try {
    saveEditBtn.setAttribute('aria-busy', 'true');
    saveEditBtn.disabled = true;

    const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/UpdateBill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to save changes');
    }

    // Reload bill to get updated data
    await loadBill(billId);
    exitEditMode();
  } catch (err) {
    showEditError(err.message);
  } finally {
    saveEditBtn.removeAttribute('aria-busy');
    saveEditBtn.disabled = false;
  }
}

// Show error state
function showError(message) {
  loadingEl.classList.add('hidden');
  errorStateEl.classList.remove('hidden');
  errorMessageEl.textContent = message;
}

// Edit error handling
function showEditError(message) {
  editErrorEl.textContent = message;
  editErrorEl.classList.remove('hidden');
}

function hideEditError() {
  editErrorEl.classList.add('hidden');
}

// Delete bill
async function deleteBill() {
  if (!confirm('Are you sure you want to delete this bill? This cannot be undone.')) {
    return;
  }

  const billId = getBillId();
  deleteBtn.setAttribute('aria-busy', 'true');
  deleteBtn.disabled = true;

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

    // Redirect to home or group page if bill belonged to a group
    if (currentBill?.groupId) {
      window.location.href = `/group.html?id=${currentBill.groupId}`;
    } else {
      window.location.href = '/';
    }
  } catch (err) {
    alert(`Failed to delete bill: ${err.message}`);
    deleteBtn.removeAttribute('aria-busy');
    deleteBtn.disabled = false;
  }
}

// Utility
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

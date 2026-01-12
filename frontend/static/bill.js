// Splitwiser - Bill View Logic

const API_BASE = 'http://localhost:8080';

// DOM Elements
const loadingEl = document.getElementById('loading');
const errorStateEl = document.getElementById('error-state');
const errorMessageEl = document.getElementById('error-message');
const billContentEl = document.getElementById('bill-content');
const billTitleEl = document.getElementById('bill-title');
const billDateEl = document.getElementById('bill-date');
const shareUrlEl = document.getElementById('share-url');
const copyBtn = document.getElementById('copy-btn');
const summarySubtotalEl = document.getElementById('summary-subtotal');
const summaryTaxEl = document.getElementById('summary-tax');
const summaryTotalEl = document.getElementById('summary-total');
const itemsSectionEl = document.getElementById('items-section');
const itemsListEl = document.getElementById('items-list');
const splitsGridEl = document.getElementById('splits-grid');
const participantsListEl = document.getElementById('participants-list');

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
    const date = new Date(bill.createdAt);
    billDateEl.textContent = date.toLocaleDateString('en-US', {
      month: 'long',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit'
    });
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
          <div class="assigned">${(item.assignedTo || []).map(escapeHtml).join(', ')}</div>
        </div>
        <div class="item-amount">$${(item.amount || 0).toFixed(2)}</div>
      </div>
    `).join('');
  }

  // Splits
  const splits = bill.split?.splits || {};
  const participants = bill.participants || [];

  splitsGridEl.innerHTML = participants.map(name => {
    const split = splits[name] || { subtotal: 0, tax: 0, total: 0 };
    return `
      <div class="result-card">
        <h3>${escapeHtml(name)}</h3>
        <div class="total">$${split.total.toFixed(2)}</div>
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

// Show error state
function showError(message) {
  loadingEl.classList.add('hidden');
  errorStateEl.classList.remove('hidden');
  errorMessageEl.textContent = message;
}

// Utility
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

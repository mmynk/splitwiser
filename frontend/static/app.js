// Splitwiser - Main Application Logic

import { requireAuth, displayUserInfo, authenticatedFetch, getCurrentUser } from './auth-utils.js';

// Require authentication
requireAuth();

// State
let participants = []; // {id, displayName, userId}
let items = [];
let groups = [];
let selectedGroupId = null;
let selectedPayerId = '';
let currentUser = null;
let _searchTimeout = null;

// DOM Elements
const balanceSection = document.getElementById('balance-section');
const totalYouOweEl = document.getElementById('total-you-owe');
const totalOwedToYouEl = document.getElementById('total-owed-to-you');
const personBalancesList = document.getElementById('person-balances-list');
const myBillsSection = document.getElementById('my-bills-section');
const myBillsList = document.getElementById('my-bills-list');
const newBillBtn = document.getElementById('new-bill-btn');
const form = document.getElementById('bill-form');
const billTitleInput = document.getElementById('bill-title');
const totalInput = document.getElementById('total');
const subtotalInput = document.getElementById('subtotal');
const taxAmountEl = document.getElementById('tax-amount');
const participantsList = document.getElementById('participants-list');
const itemsList = document.getElementById('items-list');
const addParticipantBtn = document.getElementById('add-participant');
const addItemBtn = document.getElementById('add-item');
const calculateBtn = document.getElementById('calculate-btn');
const saveBtn = document.getElementById('save-btn');
const resultsSection = document.getElementById('results');
const resultsContent = document.getElementById('results-content');
const errorEl = document.getElementById('error');
const groupSelector = document.getElementById('group-selector');
const groupSelect = document.getElementById('group-select');
const payerSection = document.getElementById('payer-section');
const payerSelect = document.getElementById('payer-select');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  currentUser = getCurrentUser();
  displayUserInfo();
  addParticipant('');
  addParticipant('');
  updateTaxDisplay();
  loadGroups();
  loadMyBalances();
  loadMyBills();
});

newBillBtn.addEventListener('click', () => {
  myBillsSection.classList.add('hidden');
  form.classList.remove('hidden');
  form.scrollIntoView({ behavior: 'smooth' });
});

// Event Listeners
addParticipantBtn.addEventListener('click', () => addParticipant(''));
addItemBtn.addEventListener('click', () => addItem());
totalInput.addEventListener('input', updateTaxDisplay);
subtotalInput.addEventListener('input', updateTaxDisplay);
groupSelect.addEventListener('change', handleGroupSelect);
payerSelect.addEventListener('change', () => {
  selectedPayerId = payerSelect.value;
});

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  await calculateSplit();
});

saveBtn.addEventListener('click', async () => {
  await saveBill();
});

// Tax Display
function updateTaxDisplay() {
  const total = parseFloat(totalInput.value) || 0;
  const subtotal = parseFloat(subtotalInput.value) || 0;
  taxAmountEl.textContent = `$${(total - subtotal).toFixed(2)}`;
}

// Groups
async function loadGroups() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/ListGroups', {
      method: 'POST',
      body: JSON.stringify({})
    });

    if (!response.ok) {
      console.error('Failed to load groups');
      return;
    }

    const data = await response.json();
    groups = data.groups || [];

    if (groups.length > 0) {
      groupSelector.classList.remove('hidden');
      renderGroupOptions();

      const urlGroupId = new URLSearchParams(window.location.search).get('group');
      if (urlGroupId && groups.find(g => g.id === urlGroupId)) {
        groupSelect.value = urlGroupId;
        handleGroupSelect();
      }
    }
  } catch (err) {
    console.error('Failed to load groups:', err);
  }
}

async function loadMyBills() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.SplitService/ListMyBills', {
      method: 'POST',
      body: JSON.stringify({})
    });

    if (!response.ok) {
      myBillsList.innerHTML = '<em>Failed to load bills.</em>';
      return;
    }

    const data = await response.json();
    const bills = data.bills || [];

    if (bills.length === 0) {
      myBillsList.innerHTML = '<p>No bills yet. <a href="#" id="create-first-bill">Create your first bill</a>.</p>';
      document.getElementById('create-first-bill')?.addEventListener('click', (e) => {
        e.preventDefault();
        myBillsSection.classList.add('hidden');
        form.classList.remove('hidden');
        form.scrollIntoView({ behavior: 'smooth' });
      });
      return;
    }

    myBillsList.innerHTML = `
      <table>
        <thead>
          <tr>
            <th>Bill</th>
            <th>Group</th>
            <th>Total</th>
            <th>Participants</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          ${bills.map(bill => {
            const date = bill.createdAt ? new Date(bill.createdAt * 1000).toLocaleDateString() : '';
            return `
              <tr>
                <td><a href="/bill.html?id=${escapeHtml(bill.billId)}">${escapeHtml(bill.title || 'Untitled')}</a></td>
                <td>${bill.groupName ? escapeHtml(bill.groupName) : '<em>—</em>'}</td>
                <td>$${(bill.total || 0).toFixed(2)}</td>
                <td>${bill.participantCount || 0}</td>
                <td>${date}</td>
              </tr>
            `;
          }).join('')}
        </tbody>
      </table>
    `;
  } catch (err) {
    console.error('Failed to load my bills:', err);
    myBillsList.innerHTML = '<em>Failed to load bills.</em>';
  }
}

async function loadMyBalances() {
  try {
    const response = await authenticatedFetch('/splitwiser.v1.GroupService/GetMyBalances', {
      method: 'POST',
      body: JSON.stringify({})
    });

    if (!response.ok) {
      console.error('Failed to load balances');
      return;
    }

    const data = await response.json();
    renderBalanceSummary(data);
  } catch (err) {
    console.error('Failed to load balances:', err);
  }
}

function renderBalanceSummary(data) {
  const youOwe = data.totalYouOwe || 0;
  const owedToYou = data.totalOwedToYou || 0;
  const personBalances = data.personBalances || [];

  if (youOwe === 0 && owedToYou === 0 && personBalances.length === 0) {
    balanceSection.classList.add('hidden');
    return;
  }

  balanceSection.classList.remove('hidden');
  totalYouOweEl.textContent = `$${youOwe.toFixed(2)}`;
  totalOwedToYouEl.textContent = `$${owedToYou.toFixed(2)}`;

  if (personBalances.length === 0) {
    personBalancesList.innerHTML = '';
    return;
  }

  // Sort: people who owe you first (positive), then people you owe (negative)
  personBalances.sort((a, b) => (b.netAmount || 0) - (a.netAmount || 0));

  personBalancesList.innerHTML = personBalances.map(person => {
    const net = person.netAmount || 0;
    const absAmount = Math.abs(net).toFixed(2);
    const isPositive = net > 0;
    const direction = isPositive ? 'owes you' : 'you owe';
    const colorClass = net === 0 ? 'neutral' : (isPositive ? 'positive' : 'negative');
    const groupBalances = person.groupBalances || [];
    const hasMultipleGroups = groupBalances.length > 1;

    return `
      <div class="person-balance-row ${colorClass}" ${hasMultipleGroups ? 'data-expandable' : ''}>
        <div class="person-balance-main">
          <span class="person-name">${escapeHtml(person.displayName)}</span>
          <span class="person-direction">${direction}</span>
          <span class="person-amount">$${absAmount}</span>
          ${hasMultipleGroups ? '<span class="expand-icon">&#9654;</span>' : ''}
        </div>
        ${groupBalances.length > 0 ? `
          <div class="person-group-breakdown hidden">
            ${groupBalances.map(gb => {
              const gbNet = gb.netAmount || 0;
              const gbAbs = Math.abs(gbNet).toFixed(2);
              const gbDir = gbNet > 0 ? 'owes you' : 'you owe';
              const gbColor = gbNet === 0 ? 'neutral' : (gbNet > 0 ? 'positive' : 'negative');
              return `
                <a href="/group.html?id=${escapeHtml(gb.groupId)}" class="group-balance-link ${gbColor}">
                  <span>${escapeHtml(gb.groupName)}</span>
                  <span>${gbDir} $${gbAbs}</span>
                </a>
              `;
            }).join('')}
          </div>
        ` : ''}
      </div>
    `;
  }).join('');

  // Expand/collapse for multi-group persons
  personBalancesList.querySelectorAll('[data-expandable]').forEach(row => {
    row.querySelector('.person-balance-main').addEventListener('click', () => {
      const breakdown = row.querySelector('.person-group-breakdown');
      const icon = row.querySelector('.expand-icon');
      breakdown.classList.toggle('hidden');
      icon.classList.toggle('expanded');
    });
  });
}

function renderGroupOptions() {
  groupSelect.innerHTML = '<option value="">Select a group...</option>' +
    groups.map(g => `<option value="${g.id}">${escapeHtml(g.name)} (${(g.members || []).length} members)</option>`).join('');
}

function handleGroupSelect() {
  const groupId = groupSelect.value;
  if (!groupId) {
    selectedGroupId = null;
    return;
  }

  const group = groups.find(g => g.id === groupId);
  if (!group) return;

  selectedGroupId = groupId;

  // group.members are now [{displayName, userId}] objects from the API
  participants = [];
  (group.members || []).forEach(m => {
    const id = Date.now() + Math.random();
    participants.push({
      id,
      displayName: m.displayName || m,
      userId: m.userId || null
    });
  });

  items.forEach(item => {
    item.participants = participants.filter(p => p.displayName.trim()).map(p => p.displayName);
  });

  renderParticipants();
  renderItems();
}

// Participants
function addParticipant(displayName = '', userId = null) {
  const id = Date.now() + Math.random();
  participants.push({ id, displayName, userId });
  renderParticipants();

  setTimeout(() => {
    const inputs = participantsList.querySelectorAll('input');
    const lastInput = inputs[inputs.length - 1];
    if (lastInput && !displayName) lastInput.focus();
  }, 0);
}

function removeParticipant(id) {
  if (participants.length <= 1) return;

  const participant = participants.find(p => p.id === id);
  participants = participants.filter(p => p.id !== id);

  if (participant) {
    items.forEach(item => {
      item.participants = item.participants.filter(name => name !== participant.displayName);
    });
  }

  renderParticipants();
  renderItems();
}

function updateParticipantName(id, oldName, newName) {
  const participant = participants.find(p => p.id === id);
  if (participant) {
    participant.displayName = newName;

    items.forEach(item => {
      const idx = item.participants.indexOf(oldName);
      if (idx !== -1) item.participants[idx] = newName;
    });

    renderItems();
  }
}

function linkParticipant(id, user) {
  const participant = participants.find(p => p.id === id);
  if (!participant) return;

  const oldName = participant.displayName;
  participant.displayName = user.displayName;
  participant.userId = user.userId;

  items.forEach(item => {
    const idx = item.participants.indexOf(oldName);
    if (idx !== -1) item.participants[idx] = user.displayName;
  });

  renderParticipants();
  renderItems();
}

// Debounced search for registered users (min 2 chars)
function searchUsers(query, callback) {
  clearTimeout(_searchTimeout);
  if (!query || query.length < 2) { callback([]); return; }
  _searchTimeout = setTimeout(async () => {
    try {
      const resp = await authenticatedFetch('/splitwiser.v1.SplitService/SearchUsers', {
        method: 'POST',
        body: JSON.stringify({ query })
      });
      callback(resp.ok ? ((await resp.json()).users || []) : []);
    } catch { callback([]); }
  }, 300);
}

function renderParticipants() {
  participantsList.innerHTML = participants.map((p, index) => `
    <div class="participant-row">
      <div class="participant-input-wrapper">
        <input
          type="text"
          value="${escapeHtml(p.displayName)}"
          placeholder="Person ${index + 1}"
          data-id="${p.id}"
          data-old-name="${escapeHtml(p.displayName)}"
        >
        ${p.userId ? `<span class="linked-badge" title="Registered user linked">✓</span>` : ''}
        <div class="search-dropdown hidden"></div>
      </div>
      <button type="button" class="remove-btn" data-remove-participant="${p.id}" ${participants.length <= 1 ? 'disabled' : ''}>
        Remove
      </button>
    </div>
  `).join('');

  participantsList.querySelectorAll('input').forEach(input => {
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        addParticipant('');
      }
    });

    input.addEventListener('blur', (e) => {
      const id = parseFloat(e.target.dataset.id);
      const oldName = e.target.dataset.oldName;
      const newName = e.target.value.trim();
      if (oldName !== newName) {
        // If user manually changed a linked participant's name, unlink them
        const p = participants.find(p => p.id === id);
        if (p?.userId) {
          p.userId = null;
          renderParticipants();
        } else {
          updateParticipantName(id, oldName, newName);
        }
        e.target.dataset.oldName = newName;
      }
      updatePayerDropdown();

      // Hide the search dropdown after a 200ms delay
      // (to allow mousedown on results to fire first)
      setTimeout(() => {
        e.target.closest('.participant-input-wrapper')
          ?.querySelector('.search-dropdown')
          ?.classList.add('hidden');
      }, 200);
    });

    input.addEventListener('input', (e) => {
      const id = parseFloat(e.target.dataset.id);
      const oldName = e.target.dataset.oldName;
      const newName = e.target.value.trim();
      updateParticipantName(id, oldName, newName);

      const dropdown = e.target.closest('.participant-input-wrapper')?.querySelector('.search-dropdown');
      if (!dropdown) return;

      searchUsers(newName, (users) => {
        if (users.length === 0) { dropdown.classList.add('hidden'); return; }
        dropdown.innerHTML = users.map(u =>
          `<div class="search-result"
                data-user-id="${escapeHtml(u.userId)}"
                data-display-name="${escapeHtml(u.displayName)}">
            <strong>${escapeHtml(u.displayName)}</strong>
            <small>${escapeHtml(u.email)}</small>
          </div>`
        ).join('');
        dropdown.classList.remove('hidden');
        dropdown.querySelectorAll('.search-result').forEach(item => {
          item.addEventListener('mousedown', (ev) => {
            ev.preventDefault();
            linkParticipant(id, { displayName: item.dataset.displayName, userId: item.dataset.userId });
          });
        });
      });
    });
  });

  participantsList.querySelectorAll('[data-remove-participant]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const id = parseFloat(e.target.dataset.removeParticipant);
      removeParticipant(id);
    });
  });

  updatePayerDropdown();
}

// Update payer dropdown when participants change
function updatePayerDropdown() {
  const validParticipants = participants.filter(p => p.displayName.trim()).map(p => p.displayName);

  if (validParticipants.length === 0) {
    payerSection.classList.add('hidden');
    return;
  }

  payerSection.classList.remove('hidden');

  payerSelect.innerHTML = '<option value="">Select payer...</option>' +
    validParticipants.map(p =>
      `<option value="${escapeHtml(p)}" ${p === selectedPayerId ? 'selected' : ''}>
        ${escapeHtml(p)}
      </option>`
    ).join('');

  if (!selectedPayerId && validParticipants.length > 0) {
    selectedPayerId = validParticipants[0];
    payerSelect.value = selectedPayerId;
  }
}

// Items
function addItem() {
  const id = Date.now() + Math.random();
  const validParticipants = participants.filter(p => p.displayName.trim()).map(p => p.displayName);
  items.push({
    id,
    description: '',
    amount: 0,
    participants: [...validParticipants]
  });
  renderItems();

  setTimeout(() => {
    const itemRows = itemsList.querySelectorAll('.item-row');
    const lastRow = itemRows[itemRows.length - 1];
    if (lastRow) {
      const descInput = lastRow.querySelector('input[type="text"]');
      if (descInput) descInput.focus();
    }
  }, 0);
}

function removeItem(id) {
  items = items.filter(i => i.id !== id);
  renderItems();
}

function updateItem(id, field, value) {
  const item = items.find(i => i.id === id);
  if (item) item[field] = value;
}

function toggleItemAssignment(itemId, participantName) {
  const item = items.find(i => i.id === itemId);
  if (item) {
    const idx = item.participants.indexOf(participantName);
    if (idx === -1) {
      item.participants.push(participantName);
    } else {
      item.participants.splice(idx, 1);
    }
  }
}

function renderItems() {
  const validParticipants = participants.filter(p => p.displayName.trim());

  itemsList.innerHTML = items.map(item => `
    <div class="item-row" data-item-id="${item.id}">
      <div class="item-header">
        <input
          type="text"
          value="${escapeHtml(item.description)}"
          placeholder="Item description"
          data-field="description"
        >
        <input
          type="number"
          value="${item.amount || ''}"
          placeholder="0.00"
          step="0.01"
          min="0"
          data-field="amount"
        >
        <button type="button" class="remove-btn" data-remove-item="${item.id}">Remove</button>
      </div>
      <div class="item-assignments">
        ${validParticipants.length === 0 ? '<span style="color: var(--pico-muted-color)">Add participants first</span>' : ''}
        ${validParticipants.map(p => `
          <label>
            <input
              type="checkbox"
              ${item.participants.includes(p.displayName) ? 'checked' : ''}
              data-participant="${escapeHtml(p.displayName)}"
            >
            ${escapeHtml(p.displayName)}
          </label>
        `).join('')}
      </div>
    </div>
  `).join('');

  itemsList.querySelectorAll('.item-row').forEach(row => {
    const itemId = parseFloat(row.dataset.itemId);

    row.querySelectorAll('input[data-field]').forEach(input => {
      input.addEventListener('input', (e) => {
        const field = e.target.dataset.field;
        let value = e.target.value;
        if (field === 'amount') value = parseFloat(value) || 0;
        updateItem(itemId, field, value);
      });

      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          const allInputs = Array.from(itemsList.querySelectorAll('input[data-field]'));
          const currentIdx = allInputs.indexOf(e.target);
          if (currentIdx === allInputs.length - 1) {
            addItem();
          } else if (currentIdx !== -1) {
            allInputs[currentIdx + 1].focus();
          }
        }
      });
    });

    row.querySelectorAll('input[data-participant]').forEach(checkbox => {
      checkbox.addEventListener('change', (e) => {
        toggleItemAssignment(itemId, e.target.dataset.participant);
      });
    });
  });

  itemsList.querySelectorAll('[data-remove-item]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const id = parseFloat(e.target.dataset.removeItem);
      removeItem(id);
    });
  });
}

// API Calls
async function calculateSplit() {
  hideError();
  resultsSection.classList.add('hidden');

  const total = parseFloat(totalInput.value) || 0;
  const subtotal = parseFloat(subtotalInput.value) || 0;
  const validParticipants = participants.filter(p => p.displayName.trim());

  if (validParticipants.length === 0) {
    showError('Please add at least one participant with a name.');
    return;
  }

  if (total <= 0) {
    showError('Please enter a valid total amount.');
    return;
  }

  const requestItems = items.map(i => ({
    description: i.description || 'Item',
    amount: i.amount,
    participantIds: i.participants
  }));

  // CalculateSplit uses display names (math only, no auth)
  const request = {
    items: requestItems,
    total,
    subtotal: subtotal || total,
    participantIds: validParticipants.map(p => p.displayName)
  };

  try {
    calculateBtn.setAttribute('aria-busy', 'true');
    calculateBtn.disabled = true;

    const response = await authenticatedFetch('/splitwiser.v1.SplitService/CalculateSplit', {
      method: 'POST',
      body: JSON.stringify(request)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to calculate split');
    }

    const data = await response.json();
    displayResults(data, validParticipants.map(p => p.displayName));
  } catch (err) {
    showError(err.message);
  } finally {
    calculateBtn.removeAttribute('aria-busy');
    calculateBtn.disabled = false;
  }
}

async function saveBill() {
  hideError();

  const total = parseFloat(totalInput.value) || 0;
  const subtotal = parseFloat(subtotalInput.value) || 0;
  const validParticipants = participants.filter(p => p.displayName.trim());

  if (validParticipants.length === 0) {
    showError('Please add at least one participant with a name.');
    return;
  }

  if (total <= 0) {
    showError('Please enter a valid total amount.');
    return;
  }

  const requestItems = items.map(i => ({
    description: i.description || 'Item',
    amount: i.amount,
    participantIds: i.participants
  }));

  const request = {
    title: billTitleInput.value.trim() || '',
    items: requestItems,
    total,
    subtotal: subtotal || total,
    // Send {displayName, userId?} — userId omitted for guests
    participants: validParticipants.map(p => ({
      displayName: p.displayName,
      ...(p.userId ? { userId: p.userId } : {})
    })),
    payerId: selectedPayerId || undefined
  };

  if (selectedGroupId) {
    request.groupId = selectedGroupId;
  }

  try {
    saveBtn.setAttribute('aria-busy', 'true');
    saveBtn.disabled = true;

    const response = await authenticatedFetch('/splitwiser.v1.SplitService/CreateBill', {
      method: 'POST',
      body: JSON.stringify(request)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to save bill');
    }

    const data = await response.json();
    window.location.href = `/bill.html?id=${data.billId}`;
  } catch (err) {
    showError(err.message);
  } finally {
    saveBtn.removeAttribute('aria-busy');
    saveBtn.disabled = false;
  }
}

// Display Results
function displayResults(data, participantNames) {
  const splits = data.splits || {};

  resultsContent.innerHTML = `
    <div class="results-grid">
      ${participantNames.map(name => {
        const rawSplit = splits[name] || {};
        const split = {
          subtotal: rawSplit.subtotal || 0,
          tax: rawSplit.tax || 0,
          total: rawSplit.total || 0,
          items: rawSplit.items || []
        };
        const items = split.items;
        return `
          <div class="result-card">
            <h3>${escapeHtml(name)}</h3>
            <div class="total">$${split.total.toFixed(2)}</div>
            ${items.length > 0 ? `
              <div class="items-breakdown">
                ${items.map(item => `
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
      }).join('')}
    </div>
  `;

  resultsSection.classList.remove('hidden');
  resultsSection.scrollIntoView({ behavior: 'smooth' });
}

// Error Handling
function showError(message) {
  errorEl.textContent = message;
  errorEl.classList.remove('hidden');
}

function hideError() {
  errorEl.classList.add('hidden');
}

// Utility
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

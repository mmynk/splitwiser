// Splitwiser - Main Application Logic

const API_BASE = 'http://localhost:8080';

// State
let participants = [];
let items = [];
let groups = [];
let selectedGroupId = null;

// DOM Elements
const form = document.getElementById('bill-form');
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

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  addParticipant('');
  addParticipant('');
  updateTaxDisplay();
  loadGroups();
});

// Event Listeners
addParticipantBtn.addEventListener('click', () => addParticipant(''));
addItemBtn.addEventListener('click', () => addItem());
totalInput.addEventListener('input', updateTaxDisplay);
subtotalInput.addEventListener('input', updateTaxDisplay);
groupSelect.addEventListener('change', handleGroupSelect);

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
  const tax = total - subtotal;
  taxAmountEl.textContent = `$${tax.toFixed(2)}`;
}

// Groups
async function loadGroups() {
  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/ListGroups`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
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
    }
  } catch (err) {
    console.error('Failed to load groups:', err);
  }
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

  // Replace participants with group members
  participants = [];
  (group.members || []).forEach(name => {
    const id = Date.now() + Math.random();
    participants.push({ id, name });
  });

  // Clear items (their participant assignments are now invalid)
  items.forEach(item => {
    item.participants = [...(group.members || [])];
  });

  renderParticipants();
  renderItems();
}

// Participants
function addParticipant(name) {
  const id = Date.now() + Math.random();
  participants.push({ id, name });
  renderParticipants();

  // Focus the new input
  setTimeout(() => {
    const inputs = participantsList.querySelectorAll('input');
    const lastInput = inputs[inputs.length - 1];
    if (lastInput && !name) lastInput.focus();
  }, 0);
}

function removeParticipant(id) {
  if (participants.length <= 1) return;

  const participant = participants.find(p => p.id === id);
  participants = participants.filter(p => p.id !== id);

  // Remove from all item assignments
  if (participant) {
    items.forEach(item => {
      item.participants = item.participants.filter(name => name !== participant.name);
    });
  }

  renderParticipants();
  renderItems();
}

function updateParticipantName(id, oldName, newName) {
  const participant = participants.find(p => p.id === id);
  if (participant) {
    participant.name = newName;

    // Update item assignments
    items.forEach(item => {
      const idx = item.participants.indexOf(oldName);
      if (idx !== -1) {
        item.participants[idx] = newName;
      }
    });

    renderItems();
  }
}

function renderParticipants() {
  participantsList.innerHTML = participants.map((p, index) => `
    <div class="participant-row">
      <input
        type="text"
        value="${escapeHtml(p.name)}"
        placeholder="Person ${index + 1}"
        data-id="${p.id}"
        data-old-name="${escapeHtml(p.name)}"
      >
      <button type="button" class="remove-btn" data-remove-participant="${p.id}" ${participants.length <= 1 ? 'disabled' : ''}>
        Remove
      </button>
    </div>
  `).join('');

  // Add event listeners
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
        updateParticipantName(id, oldName, newName);
        e.target.dataset.oldName = newName;
      }
    });

    input.addEventListener('input', (e) => {
      const id = parseFloat(e.target.dataset.id);
      const oldName = e.target.dataset.oldName;
      const newName = e.target.value.trim();
      updateParticipantName(id, oldName, newName);
    });
  });

  participantsList.querySelectorAll('[data-remove-participant]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const id = parseFloat(e.target.dataset.removeParticipant);
      removeParticipant(id);
    });
  });
}

// Items
function addItem() {
  const id = Date.now() + Math.random();
  const validParticipants = participants.filter(p => p.name.trim()).map(p => p.name);
  items.push({
    id,
    description: '',
    amount: 0,
    participants: [...validParticipants]
  });
  renderItems();

  // Focus the new description input
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
  if (item) {
    item[field] = value;
  }
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
  const validParticipants = participants.filter(p => p.name.trim());

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
              ${item.participants.includes(p.name) ? 'checked' : ''}
              data-participant="${escapeHtml(p.name)}"
            >
            ${escapeHtml(p.name)}
          </label>
        `).join('')}
      </div>
    </div>
  `).join('');

  // Add event listeners
  itemsList.querySelectorAll('.item-row').forEach(row => {
    const itemId = parseFloat(row.dataset.itemId);

    row.querySelectorAll('input[data-field]').forEach(input => {
      input.addEventListener('input', (e) => {
        const field = e.target.dataset.field;
        let value = e.target.value;
        if (field === 'amount') {
          value = parseFloat(value) || 0;
        }
        updateItem(itemId, field, value);
      });

      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          // Move to next input or add new item
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
        const participantName = e.target.dataset.participant;
        toggleItemAssignment(itemId, participantName);
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
  const validParticipants = participants.filter(p => p.name.trim()).map(p => p.name);

  if (validParticipants.length === 0) {
    showError('Please add at least one participant with a name.');
    return;
  }

  if (total <= 0) {
    showError('Please enter a valid total amount.');
    return;
  }

  // Send items as-is, let backend handle validation
  const requestItems = items.map(i => ({
    description: i.description || 'Item',
    amount: i.amount,
    participants: i.participants
  }));

  const request = {
    items: requestItems,
    total,
    subtotal: subtotal || total,
    participants: validParticipants
  };

  try {
    calculateBtn.setAttribute('aria-busy', 'true');
    calculateBtn.disabled = true;

    const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/CalculateSplit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to calculate split');
    }

    const data = await response.json();
    displayResults(data, validParticipants);
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
  const validParticipants = participants.filter(p => p.name.trim()).map(p => p.name);

  if (validParticipants.length === 0) {
    showError('Please add at least one participant with a name.');
    return;
  }

  if (total <= 0) {
    showError('Please enter a valid total amount.');
    return;
  }

  // Send items as-is, let backend handle validation
  const requestItems = items.map(i => ({
    description: i.description || 'Item',
    amount: i.amount,
    participants: i.participants
  }));

  const request = {
    title: 'Bill',
    items: requestItems,
    total,
    subtotal: subtotal || total,
    participants: validParticipants
  };

  // Include group_id if a group was selected
  if (selectedGroupId) {
    request.groupId = selectedGroupId;
  }

  try {
    saveBtn.setAttribute('aria-busy', 'true');
    saveBtn.disabled = true;

    const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/CreateBill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
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

// Splitwiser - Groups Management

const API_BASE = 'http://localhost:8080';

// State
let members = [];
let editMembers = [];
let groups = [];

// DOM Elements
const createForm = document.getElementById('create-form');
const groupNameInput = document.getElementById('group-name');
const membersList = document.getElementById('members-list');
const addMemberBtn = document.getElementById('add-member');
const createBtn = document.getElementById('create-btn');
const groupsSection = document.getElementById('groups-section');
const groupsList = document.getElementById('groups-list');
const editSection = document.getElementById('edit-section');
const editForm = document.getElementById('edit-form');
const editGroupIdInput = document.getElementById('edit-group-id');
const editGroupNameInput = document.getElementById('edit-group-name');
const editMembersList = document.getElementById('edit-members-list');
const editAddMemberBtn = document.getElementById('edit-add-member');
const cancelEditBtn = document.getElementById('cancel-edit');
const errorEl = document.getElementById('error');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  addMember('');
  addMember('');
  loadGroups();
});

// Event Listeners
addMemberBtn.addEventListener('click', () => addMember(''));
createForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  await createGroup();
});

editAddMemberBtn.addEventListener('click', () => addEditMember(''));
cancelEditBtn.addEventListener('click', cancelEdit);
editForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  await saveGroup();
});

// Members Management (Create Form)
function addMember(name) {
  const id = Date.now() + Math.random();
  members.push({ id, name });
  renderMembers();

  setTimeout(() => {
    const inputs = membersList.querySelectorAll('input');
    const lastInput = inputs[inputs.length - 1];
    if (lastInput && !name) lastInput.focus();
  }, 0);
}

function removeMember(id) {
  if (members.length <= 1) return;
  members = members.filter(m => m.id !== id);
  renderMembers();
}

function updateMemberName(id, name) {
  const member = members.find(m => m.id === id);
  if (member) {
    member.name = name;
  }
}

function renderMembers() {
  membersList.innerHTML = members.map((m, index) => `
    <div class="participant-row">
      <input
        type="text"
        value="${escapeHtml(m.name)}"
        placeholder="Member ${index + 1}"
        data-id="${m.id}"
      >
      <button type="button" class="remove-btn" data-remove="${m.id}" ${members.length <= 1 ? 'disabled' : ''}>
        Remove
      </button>
    </div>
  `).join('');

  membersList.querySelectorAll('input').forEach(input => {
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        addMember('');
      }
    });

    input.addEventListener('input', (e) => {
      const id = parseFloat(e.target.dataset.id);
      updateMemberName(id, e.target.value.trim());
    });
  });

  membersList.querySelectorAll('[data-remove]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const id = parseFloat(e.target.dataset.remove);
      removeMember(id);
    });
  });
}

// Edit Members Management
function addEditMember(name) {
  const id = Date.now() + Math.random();
  editMembers.push({ id, name });
  renderEditMembers();

  setTimeout(() => {
    const inputs = editMembersList.querySelectorAll('input');
    const lastInput = inputs[inputs.length - 1];
    if (lastInput && !name) lastInput.focus();
  }, 0);
}

function removeEditMember(id) {
  if (editMembers.length <= 1) return;
  editMembers = editMembers.filter(m => m.id !== id);
  renderEditMembers();
}

function updateEditMemberName(id, name) {
  const member = editMembers.find(m => m.id === id);
  if (member) {
    member.name = name;
  }
}

function renderEditMembers() {
  editMembersList.innerHTML = editMembers.map((m, index) => `
    <div class="participant-row">
      <input
        type="text"
        value="${escapeHtml(m.name)}"
        placeholder="Member ${index + 1}"
        data-id="${m.id}"
      >
      <button type="button" class="remove-btn" data-remove="${m.id}" ${editMembers.length <= 1 ? 'disabled' : ''}>
        Remove
      </button>
    </div>
  `).join('');

  editMembersList.querySelectorAll('input').forEach(input => {
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        addEditMember('');
      }
    });

    input.addEventListener('input', (e) => {
      const id = parseFloat(e.target.dataset.id);
      updateEditMemberName(id, e.target.value.trim());
    });
  });

  editMembersList.querySelectorAll('[data-remove]').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const id = parseFloat(e.target.dataset.remove);
      removeEditMember(id);
    });
  });
}

// API Calls
async function loadGroups() {
  hideError();

  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/ListGroups`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to load groups');
    }

    const data = await response.json();
    groups = data.groups || [];
    renderGroups();
  } catch (err) {
    showError(err.message);
    groupsList.innerHTML = '<p>Failed to load groups.</p>';
  }
}

async function createGroup() {
  hideError();

  const name = groupNameInput.value.trim();
  const validMembers = members.filter(m => m.name.trim()).map(m => m.name.trim());

  if (!name) {
    showError('Please enter a group name.');
    return;
  }

  if (validMembers.length === 0) {
    showError('Please add at least one member.');
    return;
  }

  try {
    createBtn.setAttribute('aria-busy', 'true');
    createBtn.disabled = true;

    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/CreateGroup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name,
        members: validMembers
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to create group');
    }

    // Reset form
    groupNameInput.value = '';
    members = [];
    addMember('');
    addMember('');

    // Reload groups
    await loadGroups();
  } catch (err) {
    showError(err.message);
  } finally {
    createBtn.removeAttribute('aria-busy');
    createBtn.disabled = false;
  }
}

function startEdit(group) {
  editGroupIdInput.value = group.id;
  editGroupNameInput.value = group.name;
  editMembers = (group.members || []).map((name, i) => ({ id: Date.now() + i, name }));
  if (editMembers.length === 0) {
    editMembers = [{ id: Date.now(), name: '' }];
  }
  renderEditMembers();
  editSection.classList.remove('hidden');
  editGroupNameInput.focus();
}

function cancelEdit() {
  editSection.classList.add('hidden');
  editGroupIdInput.value = '';
  editGroupNameInput.value = '';
  editMembers = [];
}

async function saveGroup() {
  hideError();

  const groupId = editGroupIdInput.value;
  const name = editGroupNameInput.value.trim();
  const validMembers = editMembers.filter(m => m.name.trim()).map(m => m.name.trim());

  if (!name) {
    showError('Please enter a group name.');
    return;
  }

  if (validMembers.length === 0) {
    showError('Please add at least one member.');
    return;
  }

  try {
    const saveBtn = document.getElementById('save-btn');
    saveBtn.setAttribute('aria-busy', 'true');
    saveBtn.disabled = true;

    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/UpdateGroup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        groupId,
        name,
        members: validMembers
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to update group');
    }

    cancelEdit();
    await loadGroups();
  } catch (err) {
    showError(err.message);
  } finally {
    const saveBtn = document.getElementById('save-btn');
    saveBtn.removeAttribute('aria-busy');
    saveBtn.disabled = false;
  }
}

async function deleteGroup(groupId, groupName) {
  if (!confirm(`Delete "${groupName}"? This cannot be undone.`)) {
    return;
  }

  hideError();

  try {
    const response = await fetch(`${API_BASE}/splitwiser.v1.GroupService/DeleteGroup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ groupId })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Failed to delete group');
    }

    await loadGroups();
  } catch (err) {
    showError(err.message);
  }
}

// Render Groups
function renderGroups() {
  if (groups.length === 0) {
    groupsList.innerHTML = '<p>No groups yet. Create one above!</p>';
    return;
  }

  groupsList.innerHTML = groups.map(group => `
    <div class="result-card" id="group-${group.id}">
      <h3>${escapeHtml(group.name)}</h3>
      <div class="breakdown">
        <strong>Members:</strong> ${(group.members || []).map(m => escapeHtml(m)).join(', ') || 'No members'}
      </div>
      <div id="bills-${group.id}" class="group-bills" style="margin-top: var(--spacing-md);">
        <div style="display: flex; align-items: center; gap: var(--spacing-sm);">
          <strong>Bills:</strong>
          <button type="button" class="secondary outline" style="font-size: 0.9em; padding: 0.25em 0.5em;" onclick="toggleBills('${group.id}')">Show Bills</button>
        </div>
        <div id="bills-list-${group.id}" style="display: none; margin-top: var(--spacing-sm);"></div>
      </div>
      <div style="margin-top: var(--spacing-md); display: flex; gap: var(--spacing-sm); flex-wrap: wrap;">
        <a href="/group.html?id=${group.id}" role="button" class="contrast" style="flex: 2; text-align: center; margin-bottom: 0;">View Group</a>
        <button type="button" class="secondary outline" onclick="startEdit(${JSON.stringify(group).replace(/"/g, '&quot;')})" style="flex: 1;">Edit</button>
        <button type="button" class="remove-btn" onclick="deleteGroup('${group.id}', '${escapeHtml(group.name)}')" style="flex: 1;">Delete</button>
      </div>
    </div>
  `).join('');
}

// Toggle bills display for a group
async function toggleBills(groupId) {
  const billsList = document.getElementById(`bills-list-${groupId}`);
  const button = event.target;

  if (billsList.style.display === 'none') {
    // Load and show bills
    button.setAttribute('aria-busy', 'true');
    button.textContent = 'Loading...';

    try {
      const bills = await loadBillsForGroup(groupId);
      renderBillsList(groupId, bills);
      billsList.style.display = 'block';
      button.textContent = 'Hide Bills';
    } catch (err) {
      showError(err.message);
      button.textContent = 'Show Bills';
    } finally {
      button.removeAttribute('aria-busy');
    }
  } else {
    // Hide bills
    billsList.style.display = 'none';
    button.textContent = 'Show Bills';
  }
}

// Load bills for a specific group
async function loadBillsForGroup(groupId) {
  const response = await fetch(`${API_BASE}/splitwiser.v1.SplitService/ListBillsByGroup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ groupId })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Failed to load bills');
  }

  const data = await response.json();
  return data.bills || [];
}

// Render bills list for a group
function renderBillsList(groupId, bills) {
  const billsList = document.getElementById(`bills-list-${groupId}`);

  if (bills.length === 0) {
    billsList.innerHTML = '<p style="color: var(--muted-color); font-style: italic;">No bills yet for this group.</p>';
    return;
  }

  billsList.innerHTML = `
    <table role="grid">
      <thead>
        <tr>
          <th>Title</th>
          <th>Total</th>
          <th>Participants</th>
          <th>Date</th>
        </tr>
      </thead>
      <tbody>
        ${bills.map(bill => `
          <tr>
            <td><a href="/bill.html?id=${bill.billId}">${escapeHtml(bill.title)}</a></td>
            <td>$${bill.total.toFixed(2)}</td>
            <td>${bill.participantCount}</td>
            <td>${formatDate(bill.createdAt)}</td>
          </tr>
        `).join('')}
      </tbody>
    </table>
  `;
}

// Format Unix timestamp to readable date
function formatDate(timestamp) {
  const date = new Date(timestamp * 1000);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  });
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

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
    <div class="result-card">
      <h3>${escapeHtml(group.name)}</h3>
      <div class="breakdown">
        ${(group.members || []).map(m => escapeHtml(m)).join(', ') || 'No members'}
      </div>
      <div style="margin-top: var(--spacing-md); display: flex; gap: var(--spacing-sm);">
        <button type="button" class="secondary outline" onclick="startEdit(${JSON.stringify(group).replace(/"/g, '&quot;')})">Edit</button>
        <button type="button" class="remove-btn" onclick="deleteGroup('${group.id}', '${escapeHtml(group.name)}')">Delete</button>
      </div>
    </div>
  `).join('');
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

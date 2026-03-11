// Splitwiser - Shared Form Components
// Reusable participant and item management for bill creation/editing

class BillForm {
  constructor(options = {}) {
    this.participants = []; // {id, displayName, userId}
    this.items = [];
    this.onUpdate = options.onUpdate || (() => {});
    this._searchTimeout = null;

    // DOM containers (set via init)
    this.participantsList = null;
    this.itemsList = null;
    this.totalInput = null;
    this.subtotalInput = null;
    this.taxAmountEl = null;
  }

  init({ participantsList, itemsList, totalInput, subtotalInput, taxAmountEl }) {
    this.participantsList = participantsList;
    this.itemsList = itemsList;
    this.totalInput = totalInput;
    this.subtotalInput = subtotalInput;
    this.taxAmountEl = taxAmountEl;

    if (this.totalInput && this.subtotalInput) {
      this.totalInput.addEventListener('input', () => this.updateTaxDisplay());
      this.subtotalInput.addEventListener('input', () => this.updateTaxDisplay());
    }
  }

  // Load existing bill data
  loadBill(bill) {
    if (this.totalInput) this.totalInput.value = bill.total || '';
    if (this.subtotalInput) this.subtotalInput.value = bill.subtotal || '';
    this.updateTaxDisplay();

    // Participants come as [{displayName, userId}] objects from the API
    this.participants = (bill.participants || []).map((p, i) => ({
      id: Date.now() + i + Math.random(),
      displayName: typeof p === 'string' ? p : (p.displayName || ''),
      userId: (typeof p === 'object' ? p.userId : null) || null
    }));

    this.items = (bill.items || []).map((item, i) => ({
      id: Date.now() + i + Math.random(),
      description: item.description || '',
      amount: item.amount || 0,
      participants: [...(item.participants || [])]
    }));

    this.renderParticipants();
    this.renderItems();
  }

  // Get current form data
  getData() {
    const total = parseFloat(this.totalInput?.value) || 0;
    const subtotal = parseFloat(this.subtotalInput?.value) || 0;
    const validParticipants = this.participants.filter(p => p.displayName.trim());

    // Send {displayName, userId?} — userId omitted for guests
    const participants = validParticipants.map(p => ({
      displayName: p.displayName,
      ...(p.userId ? { userId: p.userId } : {})
    }));

    const requestItems = this.items.map(i => ({
      description: i.description || 'Item',
      amount: i.amount,
      participantIds: i.participants
    }));

    return { total, subtotal: subtotal || total, participants, items: requestItems };
  }

  // Tax Display
  updateTaxDisplay() {
    if (!this.taxAmountEl) return;
    const total = parseFloat(this.totalInput?.value) || 0;
    const subtotal = parseFloat(this.subtotalInput?.value) || 0;
    this.taxAmountEl.textContent = `$${(total - subtotal).toFixed(2)}`;
  }

  // Participants
  addParticipant(displayName = '', userId = null) {
    const id = Date.now() + Math.random();
    this.participants.push({ id, displayName, userId });
    this.renderParticipants();

    setTimeout(() => {
      const inputs = this.participantsList?.querySelectorAll('input');
      const lastInput = inputs?.[inputs.length - 1];
      if (lastInput && !displayName) lastInput.focus();
    }, 0);
  }

  removeParticipant(id) {
    if (this.participants.length <= 1) return;

    const participant = this.participants.find(p => p.id === id);
    this.participants = this.participants.filter(p => p.id !== id);

    if (participant) {
      this.items.forEach(item => {
        item.participants = item.participants.filter(name => name !== participant.displayName);
      });
    }

    this.renderParticipants();
    this.renderItems();
  }

  updateParticipantName(id, oldName, newName) {
    const participant = this.participants.find(p => p.id === id);
    if (participant) {
      participant.displayName = newName;

      this.items.forEach(item => {
        const idx = item.participants.indexOf(oldName);
        if (idx !== -1) item.participants[idx] = newName;
      });

      this.renderItems();
    }
  }

  // Link a participant to a registered user account
  linkParticipant(id, user) {
    const participant = this.participants.find(p => p.id === id);
    if (!participant) return;

    const oldName = participant.displayName;
    participant.displayName = user.displayName;
    participant.userId = user.userId;

    this.items.forEach(item => {
      const idx = item.participants.indexOf(oldName);
      if (idx !== -1) item.participants[idx] = user.displayName;
    });

    this.renderParticipants();
    this.renderItems();
  }

  // Debounced search for registered users (min 2 chars)
  _searchUsers(query, callback) {
    clearTimeout(this._searchTimeout);
    if (!query || query.length < 2) { callback([]); return; }
    this._searchTimeout = setTimeout(async () => {
      try {
        const token = localStorage.getItem('auth_token');
        const resp = await fetch('/splitwiser.v1.FriendService/SearchFriends', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
          body: JSON.stringify({ query })
        });
        callback(resp.ok ? ((await resp.json()).users || []) : []);
      } catch { callback([]); }
    }, 300);
  }

  renderParticipants() {
    if (!this.participantsList) return;

    this.participantsList.innerHTML = this.participants.map((p, index) => `
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
        <button type="button" class="remove-btn" data-remove-participant="${p.id}" ${this.participants.length <= 1 ? 'disabled' : ''}>
          Remove
        </button>
      </div>
    `).join('');

    this.participantsList.querySelectorAll('input').forEach(input => {
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          this.addParticipant('');
        }
      });

      input.addEventListener('input', (e) => {
        const id = parseFloat(e.target.dataset.id);
        const oldName = e.target.dataset.oldName;
        const newName = e.target.value.trim();
        this.updateParticipantName(id, oldName, newName);

        // Search for matching registered users and populate dropdown
        const dropdown = e.target.closest('.participant-input-wrapper')?.querySelector('.search-dropdown');
        if (!dropdown) return;

        this._searchUsers(newName, (users) => {
          if (users.length === 0) { dropdown.classList.add('hidden'); return; }
          dropdown.innerHTML = users.map(u =>
            `<div class="search-result"
                  data-user-id="${escapeHtml(u.userId)}"
                  data-display-name="${escapeHtml(u.displayName)}">
              <strong>${escapeHtml(u.displayName)}</strong>
            </div>`
          ).join('');
          dropdown.classList.remove('hidden');
          dropdown.querySelectorAll('.search-result').forEach(item => {
            // mousedown fires before blur, preventing dropdown hiding before selection
            item.addEventListener('mousedown', (ev) => {
              ev.preventDefault();
              this.linkParticipant(id, {
                displayName: item.dataset.displayName,
                userId: item.dataset.userId
              });
            });
          });
        });
      });

      input.addEventListener('blur', (e) => {
        const id = parseFloat(e.target.dataset.id);
        const oldName = e.target.dataset.oldName;
        const newName = e.target.value.trim();
        if (oldName !== newName) {
          // If the name was manually changed away from a linked user, unlink them
          const p = this.participants.find(p => p.id === id);
          if (p?.userId) {
            p.userId = null;
            this.renderParticipants();
          } else {
            this.updateParticipantName(id, oldName, newName);
          }
          e.target.dataset.oldName = newName;
        }
        // Hide dropdown after a delay to allow mousedown on results to fire first
        setTimeout(() => {
          e.target.closest?.('.participant-input-wrapper')?.querySelector('.search-dropdown')?.classList.add('hidden');
        }, 200);
      });
    });

    this.participantsList.querySelectorAll('[data-remove-participant]').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const id = parseFloat(e.target.dataset.removeParticipant);
        this.removeParticipant(id);
      });
    });
  }

  // Items
  addItem() {
    const id = Date.now() + Math.random();
    const validParticipants = this.participants.filter(p => p.displayName.trim()).map(p => p.displayName);
    this.items.push({
      id,
      description: '',
      amount: 0,
      participants: [...validParticipants]
    });
    this.renderItems();

    setTimeout(() => {
      const itemRows = this.itemsList?.querySelectorAll('.item-row');
      const lastRow = itemRows?.[itemRows.length - 1];
      if (lastRow) {
        const descInput = lastRow.querySelector('input[type="text"]');
        if (descInput) descInput.focus();
      }
    }, 0);
  }

  removeItem(id) {
    this.items = this.items.filter(i => i.id !== id);
    this.renderItems();
  }

  updateItem(id, field, value) {
    const item = this.items.find(i => i.id === id);
    if (item) item[field] = value;
  }

  toggleItemAssignment(itemId, participantName) {
    const item = this.items.find(i => i.id === itemId);
    if (item) {
      const idx = item.participants.indexOf(participantName);
      if (idx === -1) {
        item.participants.push(participantName);
      } else {
        item.participants.splice(idx, 1);
      }
    }
  }

  renderItems() {
    if (!this.itemsList) return;

    const validParticipants = this.participants.filter(p => p.displayName.trim());

    this.itemsList.innerHTML = this.items.map(item => `
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

    this.itemsList.querySelectorAll('.item-row').forEach(row => {
      const itemId = parseFloat(row.dataset.itemId);

      row.querySelectorAll('input[data-field]').forEach(input => {
        input.addEventListener('input', (e) => {
          const field = e.target.dataset.field;
          let value = e.target.value;
          if (field === 'amount') value = parseFloat(value) || 0;
          this.updateItem(itemId, field, value);
        });

        input.addEventListener('keydown', (e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            const allInputs = Array.from(this.itemsList.querySelectorAll('input[data-field]'));
            const currentIdx = allInputs.indexOf(e.target);
            if (currentIdx === allInputs.length - 1) {
              this.addItem();
            } else if (currentIdx !== -1) {
              allInputs[currentIdx + 1].focus();
            }
          }
        });
      });

      row.querySelectorAll('input[data-participant]').forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
          this.toggleItemAssignment(itemId, e.target.dataset.participant);
        });
      });
    });

    this.itemsList.querySelectorAll('[data-remove-item]').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const id = parseFloat(e.target.dataset.removeItem);
        this.removeItem(id);
      });
    });
  }

  // Validation
  validate() {
    const data = this.getData();

    if (data.participants.length === 0) {
      return { valid: false, error: 'Please add at least one participant with a name.' };
    }

    if (data.total <= 0) {
      return { valid: false, error: 'Please enter a valid total amount.' };
    }

    return { valid: true };
  }
}

// Utility
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { BillForm, escapeHtml };
}

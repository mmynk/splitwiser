// Splitwiser - Shared Form Components
// Reusable participant and item management for bill creation/editing

class BillForm {
  constructor(options = {}) {
    this.participants = [];
    this.items = [];
    this.onUpdate = options.onUpdate || (() => {});

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

    // Add input listeners for tax calculation
    if (this.totalInput && this.subtotalInput) {
      this.totalInput.addEventListener('input', () => this.updateTaxDisplay());
      this.subtotalInput.addEventListener('input', () => this.updateTaxDisplay());
    }
  }

  // Load existing bill data
  loadBill(bill) {
    // Set amounts
    if (this.totalInput) this.totalInput.value = bill.total || '';
    if (this.subtotalInput) this.subtotalInput.value = bill.subtotal || '';
    this.updateTaxDisplay();

    // Load participants
    this.participants = (bill.participants || []).map((name, i) => ({
      id: Date.now() + i + Math.random(),
      name
    }));

    // Load items
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
    const validParticipants = this.participants.filter(p => p.name.trim()).map(p => p.name);
    const requestItems = this.items.map(i => ({
      description: i.description || 'Item',
      amount: i.amount,
      participants: i.participants
    }));

    return {
      total,
      subtotal: subtotal || total,
      participants: validParticipants,
      items: requestItems
    };
  }

  // Tax Display
  updateTaxDisplay() {
    if (!this.taxAmountEl) return;
    const total = parseFloat(this.totalInput?.value) || 0;
    const subtotal = parseFloat(this.subtotalInput?.value) || 0;
    const tax = total - subtotal;
    this.taxAmountEl.textContent = `$${tax.toFixed(2)}`;
  }

  // Participants
  addParticipant(name = '') {
    const id = Date.now() + Math.random();
    this.participants.push({ id, name });
    this.renderParticipants();

    // Focus the new input
    setTimeout(() => {
      const inputs = this.participantsList?.querySelectorAll('input');
      const lastInput = inputs?.[inputs.length - 1];
      if (lastInput && !name) lastInput.focus();
    }, 0);
  }

  removeParticipant(id) {
    if (this.participants.length <= 1) return;

    const participant = this.participants.find(p => p.id === id);
    this.participants = this.participants.filter(p => p.id !== id);

    // Remove from all item assignments
    if (participant) {
      this.items.forEach(item => {
        item.participants = item.participants.filter(name => name !== participant.name);
      });
    }

    this.renderParticipants();
    this.renderItems();
  }

  updateParticipantName(id, oldName, newName) {
    const participant = this.participants.find(p => p.id === id);
    if (participant) {
      participant.name = newName;

      // Update item assignments
      this.items.forEach(item => {
        const idx = item.participants.indexOf(oldName);
        if (idx !== -1) {
          item.participants[idx] = newName;
        }
      });

      this.renderItems();
    }
  }

  renderParticipants() {
    if (!this.participantsList) return;

    this.participantsList.innerHTML = this.participants.map((p, index) => `
      <div class="participant-row">
        <input
          type="text"
          value="${escapeHtml(p.name)}"
          placeholder="Person ${index + 1}"
          data-id="${p.id}"
          data-old-name="${escapeHtml(p.name)}"
        >
        <button type="button" class="remove-btn" data-remove-participant="${p.id}" ${this.participants.length <= 1 ? 'disabled' : ''}>
          Remove
        </button>
      </div>
    `).join('');

    // Add event listeners
    this.participantsList.querySelectorAll('input').forEach(input => {
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          this.addParticipant('');
        }
      });

      input.addEventListener('blur', (e) => {
        const id = parseFloat(e.target.dataset.id);
        const oldName = e.target.dataset.oldName;
        const newName = e.target.value.trim();
        if (oldName !== newName) {
          this.updateParticipantName(id, oldName, newName);
          e.target.dataset.oldName = newName;
        }
      });

      input.addEventListener('input', (e) => {
        const id = parseFloat(e.target.dataset.id);
        const oldName = e.target.dataset.oldName;
        const newName = e.target.value.trim();
        this.updateParticipantName(id, oldName, newName);
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
    const validParticipants = this.participants.filter(p => p.name.trim()).map(p => p.name);
    this.items.push({
      id,
      description: '',
      amount: 0,
      participants: [...validParticipants]
    });
    this.renderItems();

    // Focus the new description input
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
    if (item) {
      item[field] = value;
    }
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

    const validParticipants = this.participants.filter(p => p.name.trim());

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
    this.itemsList.querySelectorAll('.item-row').forEach(row => {
      const itemId = parseFloat(row.dataset.itemId);

      row.querySelectorAll('input[data-field]').forEach(input => {
        input.addEventListener('input', (e) => {
          const field = e.target.dataset.field;
          let value = e.target.value;
          if (field === 'amount') {
            value = parseFloat(value) || 0;
          }
          this.updateItem(itemId, field, value);
        });

        input.addEventListener('keydown', (e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            // Move to next input or add new item
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
          const participantName = e.target.dataset.participant;
          this.toggleItemAssignment(itemId, participantName);
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

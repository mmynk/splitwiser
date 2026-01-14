# Splitwiser Roadmap

## Priority 1: Groups - ✅ COMPLETE
Reusable participant groups for common splitting scenarios.

**Use cases:**
- Roommates splitting rent/utilities monthly
- Work lunch crew
- Travel groups
- Regular friend groups

**Completed Features:**
- ✅ Create/edit/delete groups with named participants
- ✅ Select group when creating a bill to auto-populate participants
- ✅ Group management UI (groups.html)
- ✅ Track who paid each bill (single payer, extensible to multiple)
- ✅ Bill naming with smart auto-generation (items + participants hybrid)
- ✅ Backend: GetGroupBalances API with balance calculation algorithm
- ✅ Backend: All payer validation and storage
- ✅ Frontend: Title and payer inputs in bill creation form
- ✅ Group detail page with balance views (group.html/group.js)
- ✅ Bill view/edit page updates for title/payer display
- ✅ Balance card styling with color-coded net balances
- ✅ Clickable group names in groups list (replaced separate "View Group" button)
- ✅ Toggle between total balances and detailed debt matrix views
- ✅ Show group link on bill page when bill belongs to a group
- ✅ Delete bill functionality from bill detail page and group bills list
- ✅ Proto refactoring: split into common.proto, bill.proto, group.proto
- ✅ Debt simplification algorithm: minimizes number of transactions
- ✅ Fixed payer_id in ListBillsByGroup response

**Technical Implementation:**
- Single payer per bill (extensible to multiple payers via bill_payments table)
- Server-side balance calculation (accurate rounding, single source of truth)
- Two balance views: Total balances (default) + Simplified debt matrix
- Debt simplification: greedy algorithm minimizes number of transactions
- Title auto-generation: "Items - Participants" format with graceful fallbacks
- Modular proto organization for better maintainability

## Priority 2: Settlements
Calculate optimal payments to settle balances.

**Features:**
- Track who paid vs who owes across bills
- Debt simplification algorithm (minimize number of transactions)
- "Settle up" suggestions (e.g., "Alice pays Bob $15")
- Mark debts as settled

## Future Ideas

### Receipt OCR
- Upload receipt image
- Auto-extract items and amounts
- Review/edit before saving

### Other Ideas
- Bill templates (save common bill structures)
- Notifications/reminders
- Export to CSV/PDF
- Multiple currencies
- Recurring bills

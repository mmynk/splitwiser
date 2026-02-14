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

## Priority 2: Settlements - ✅ COMPLETE
Record payments between group members to clear debts.

**Completed Features:**
- ✅ Settlement model and database table with cascade delete on group
- ✅ RecordSettlement API with validation (positive amount, different users, both members)
- ✅ ListSettlements API returns all settlements for a group
- ✅ DeleteSettlement API with group membership check
- ✅ GetGroupBalances now includes settlements in balance calculations
- ✅ Frontend: Settlements section on group page with table view
- ✅ Frontend: "Record Settlement" dialog with from/to dropdowns, amount, note
- ✅ Frontend: Delete settlement with confirmation
- ✅ Balances automatically update when settlements are recorded/deleted

**Technical Implementation:**
- Settlement fields: ID, GroupID, FromUserID, ToUserID, Amount, CreatedAt, CreatedBy, Note
- Balance calculation: settlements adjust TotalPaid (payer) and TotalOwed (receiver)
- Debt simplification algorithm automatically computes minimal transactions from adjusted balances
- No confirmation flow (MVP simplicity - trust the recorder)
- Any group member can record/delete settlements (no ownership restrictions)

**How it works:**
1. Alice pays $100 bill split with Bob → Bob owes Alice $50
2. Bob records settlement to Alice for $30 → Bob now owes Alice $20
3. Balances and debt matrix update automatically

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

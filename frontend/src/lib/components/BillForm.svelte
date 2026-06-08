<script lang="ts" module>
  export interface BillParticipantState {
    id: string;
    displayName: string;
    userId?: string;
  }

  export interface BillItemState {
    id: string;
    description: string;
    // Stored as a string so partial inputs like "0." or "" render verbatim — parseFloat coercion
    // mid-typing would wipe the input.
    amountRaw: string;
    participantNames: string[];
  }

  export interface BillFormSerialized {
    total: number;
    subtotal: number;
    participants: { displayName: string; userId?: string }[];
    items: { description: string; amount: number; participantIds: string[] }[];
    payerId: string;
    groupId: string;
  }

  export interface BillFormInitial {
    total?: number;
    subtotal?: number;
    participants?: { displayName: string; userId?: string }[];
    items?: { description: string; amount: number; participantIds?: string[] }[];
    payerId?: string;
    groupId?: string;
  }

  let nextLocalId = 1;
  function nextId(): string {
    return `r${nextLocalId++}`;
  }
</script>

<script lang="ts">
  import { tick, untrack } from 'svelte';
  import { Plus, Trash2, BadgeCheck } from 'lucide-svelte';
  import type { Group } from '$lib/api/types';
  import type { AuthUser } from '$lib/stores/auth';
  import UserSearch, { type UserPick } from './UserSearch.svelte';
  import { validateImportData, type ImportedBill } from '$lib/util/importValidator';

  interface Props {
    groups?: Group[];
    currentUser?: AuthUser | null;
    initial?: BillFormInitial;
    showGroupSelector?: boolean;
  }

  let {
    groups = [],
    currentUser = null,
    initial,
    showGroupSelector = true,
  }: Props = $props();

  function makeParticipant(displayName: string, userId?: string): BillParticipantState {
    return { id: nextId(), displayName, userId };
  }

  function makeItem(description = '', amountRaw = '', participantNames: string[] = []): BillItemState {
    return { id: nextId(), description, amountRaw, participantNames };
  }

  function amountToRaw(n: number | undefined): string {
    return n == null || n === 0 ? '' : String(n);
  }

  function parseNumber(raw: string): number {
    const n = parseFloat(raw);
    return isNaN(n) ? 0 : n;
  }

  function buildInitialParticipants(
    data: BillFormInitial | undefined,
    user: AuthUser | null,
  ): BillParticipantState[] {
    if (data) {
      return (data.participants ?? []).map((p) => makeParticipant(p.displayName, p.userId));
    }
    const out: BillParticipantState[] = [];
    if (user?.displayName) out.push(makeParticipant(user.displayName, user.id));
    out.push(makeParticipant(''));
    return out;
  }

  function buildInitialItems(data: BillFormInitial | undefined): BillItemState[] {
    if (!data) return [];
    return (data.items ?? []).map((i) =>
      makeItem(i.description, amountToRaw(i.amount), [...(i.participantIds ?? [])]),
    );
  }

  // `initial` and `currentUser` are read once at construction. The form does not
  // re-sync if the parent swaps them later — use reset() / loadImport() instead.
  const _initial: BillFormInitial | undefined = untrack(() => initial);
  const _initialUser: AuthUser | null = untrack(() => currentUser);

  let participants: BillParticipantState[] = $state(buildInitialParticipants(_initial, _initialUser));
  let items: BillItemState[] = $state(buildInitialItems(_initial));
  let totalRaw = $state(_initial?.total != null ? String(_initial.total) : '');
  let subtotalRaw = $state(_initial?.subtotal != null ? String(_initial.subtotal) : '');
  let payerName = $state(_initial?.payerId ?? '');
  let groupId = $state(_initial?.groupId ?? '');

  let validParticipants = $derived(participants.filter((p) => p.displayName.trim()));
  let linkedUserIds = $derived(
    validParticipants.filter((p): p is BillParticipantState & { userId: string } => !!p.userId).map((p) => p.userId),
  );
  let payerOptions = $derived(validParticipants.map((p) => p.displayName));

  $effect(() => {
    if (payerOptions.length === 0) {
      if (payerName !== '') payerName = '';
    } else if (!payerOptions.includes(payerName)) {
      payerName = payerOptions[0];
    }
  });

  let total = $derived(parseNumber(totalRaw));
  let subtotal = $derived(parseNumber(subtotalRaw));
  let taxAmount = $derived(Math.max(0, total - subtotal));
  let showSubtotal = $derived(items.length > 0);

  async function addParticipantRow(): Promise<void> {
    participants = [...participants, makeParticipant('')];
    await tick();
  }

  function removeParticipantRow(id: string): void {
    if (participants.length <= 1) return;
    const target = participants.find((p) => p.id === id);
    participants = participants.filter((p) => p.id !== id);
    if (target?.displayName) {
      const oldName = target.displayName;
      items = items.map((item) => ({
        ...item,
        participantNames: item.participantNames.filter((n) => n !== oldName),
      }));
    }
  }

  function updateParticipantName(id: string, newName: string): void {
    const p = participants.find((x) => x.id === id);
    if (!p) return;
    const oldName = p.displayName;
    if (oldName === newName) return;
    // Manually editing a linked user unlinks them.
    if (p.userId) p.userId = undefined;
    p.displayName = newName;
    participants = [...participants];
    if (oldName && items.some((it) => it.participantNames.includes(oldName))) {
      items = items.map((item) => ({
        ...item,
        participantNames: item.participantNames.map((n) => (n === oldName ? newName : n)),
      }));
    }
    if (payerName === oldName) payerName = newName;
  }

  function linkParticipant(id: string, user: UserPick): void {
    const p = participants.find((x) => x.id === id);
    if (!p) return;
    const oldName = p.displayName;
    p.displayName = user.displayName;
    p.userId = user.userId;
    participants = [...participants];
    items = items.map((item) => ({
      ...item,
      participantNames: item.participantNames.map((n) => (n === oldName ? user.displayName : n)),
    }));
    if (payerName === oldName) payerName = user.displayName;
  }

  function handleGroupChange(): void {
    if (!groupId) return;
    const g = groups.find((x) => x.id === groupId);
    if (!g) return;
    participants = (g.members ?? []).map((m) => makeParticipant(m.displayName, m.userId));
    const names = participants.map((p) => p.displayName);
    items = items.map((item) => ({ ...item, participantNames: [...names] }));
  }

  async function addItemRow(): Promise<void> {
    const allNames = validParticipants.map((p) => p.displayName);
    items = [...items, makeItem('', '', allNames)];
    await tick();
    const last = document.querySelector<HTMLInputElement>(
      '.bf-item-row:last-of-type input[data-field="description"]',
    );
    last?.focus();
  }

  function removeItemRow(id: string): void {
    items = items.filter((i) => i.id !== id);
  }

  function toggleAssignment(itemId: string, participantName: string): void {
    const it = items.find((i) => i.id === itemId);
    if (!it) return;
    const idx = it.participantNames.indexOf(participantName);
    if (idx === -1) it.participantNames = [...it.participantNames, participantName];
    else it.participantNames = it.participantNames.filter((_, i) => i !== idx);
  }

  function handleItemKeydown(e: KeyboardEvent): void {
    if (e.key !== 'Enter') return;
    e.preventDefault();
    const root = (e.currentTarget as HTMLElement).closest('.bf-items') as HTMLElement | null;
    if (!root) return;
    const all = Array.from(root.querySelectorAll<HTMLInputElement>('input[data-field]'));
    const idx = all.indexOf(e.target as HTMLInputElement);
    if (idx === -1) return;
    if (idx === all.length - 1) addItemRow();
    else all[idx + 1].focus();
  }

  export function getData(): BillFormSerialized {
    const cleaned = participants
      .map((p) => ({ ...p, displayName: p.displayName.trim() }))
      .filter((p) => p.displayName);

    const serializedParticipants = cleaned.map((p) =>
      p.userId ? { displayName: p.displayName, userId: p.userId } : { displayName: p.displayName },
    );

    const serializedItems = items.map((i) => ({
      description: i.description.trim() || 'Item',
      amount: parseNumber(i.amountRaw),
      participantIds: [...i.participantNames],
    }));

    return {
      total,
      subtotal: subtotal > 0 ? subtotal : total,
      participants: serializedParticipants,
      items: serializedItems,
      payerId: cleaned.some((p) => p.displayName === payerName) ? payerName : '',
      groupId,
    };
  }

  export function loadImport(data: ImportedBill): void {
    totalRaw = data.total.toFixed(2);
    subtotalRaw = data.subtotal.toFixed(2);
    participants = data.participants.map((name) => makeParticipant(name));
    const names = participants.map((p) => p.displayName);
    items = data.items.map((it) => makeItem(it.description, amountToRaw(it.amount), [...names]));
    // payerOptions derives from participants; use the freshly computed names instead.
    if (!names.includes(payerName)) payerName = names[0] ?? '';
  }

  export function importJson(raw: unknown): void {
    const normalized = validateImportData(raw);
    loadImport(normalized);
  }

  export function setGroup(id: string): void {
    if (!groups.find((x) => x.id === id)) return;
    groupId = id;
    handleGroupChange();
  }

  export function reset(): void {
    participants = buildInitialParticipants(undefined, currentUser);
    items = [];
    totalRaw = '';
    subtotalRaw = '';
    payerName = '';
    groupId = '';
  }
</script>

<div class="flex flex-col gap-6">
  <!-- Total -->
  <div>
    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Total (with tax)</span>
      <input
        type="number"
        step="0.01"
        min="0"
        placeholder="0.00"
        bind:value={totalRaw}
        required
        class="rounded-md border border-border px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
      />
    </label>
  </div>

  <!-- Participants -->
  <section class="flex flex-col gap-2">
    <div class="flex items-baseline justify-between">
      <h3 class="text-base font-semibold text-text">Participants</h3>
      <span class="text-xs text-text-muted">Press Enter to add another</span>
    </div>

    {#if showGroupSelector && groups.length > 0}
      <label class="flex flex-col gap-1 text-sm">
        <span class="text-text-muted">Load from group</span>
        <select
          bind:value={groupId}
          onchange={handleGroupChange}
          class="rounded-md border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
        >
          <option value="">Select a group…</option>
          {#each groups as g (g.id)}
            <option value={g.id}>{g.name} ({(g.members ?? []).length} members)</option>
          {/each}
        </select>
      </label>
    {/if}

    <div class="flex flex-col gap-2">
      {#each participants as p, i (p.id)}
        <div class="flex items-start gap-2">
          <div class="relative flex-1">
            <UserSearch
              value={p.displayName}
              placeholder={`Person ${i + 1}`}
              excludeIds={linkedUserIds.filter((id) => id !== p.userId)}
              autofocus={!p.displayName && i === participants.length - 1}
              inputClass="w-full rounded-md border border-border px-3 py-2 pr-8 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
              onInput={(v) => updateParticipantName(p.id, v)}
              onSelect={(u) => linkParticipant(p.id, u)}
              onEnter={() => addParticipantRow()}
            />
            {#if p.userId}
              <span
                class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-success"
                title="Registered user linked"
              >
                <BadgeCheck size={16} />
              </span>
            {/if}
          </div>
          <button
            type="button"
            aria-label="Remove participant"
            disabled={participants.length <= 1}
            onclick={() => removeParticipantRow(p.id)}
            class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border text-text-muted hover:bg-surface-sunken disabled:cursor-not-allowed disabled:opacity-40"
          >
            <Trash2 size={16} />
          </button>
        </div>
      {/each}
    </div>

    <button
      type="button"
      onclick={() => addParticipantRow()}
      class="mt-1 inline-flex items-center gap-1 self-start rounded-md border border-dashed border-border px-3 py-1.5 text-sm font-medium text-text-muted hover:bg-surface-sunken"
    >
      <Plus size={14} /> Add participant
    </button>
  </section>

  <!-- Payer -->
  {#if payerOptions.length > 0}
    <section class="flex flex-col gap-1">
      <label class="flex flex-col gap-1 text-sm">
        <span class="font-medium text-text">Who paid this bill?</span>
        <select
          bind:value={payerName}
          class="rounded-md border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
        >
          {#each payerOptions as name}
            <option value={name}>{name}</option>
          {/each}
        </select>
      </label>
    </section>
  {/if}

  <!-- Items -->
  <section class="flex flex-col gap-2">
    <div class="flex items-baseline justify-between">
      <h3 class="text-base font-semibold text-text">
        Items <span class="text-xs font-normal text-text-muted">(optional)</span>
      </h3>
      <span class="text-xs text-text-muted">Leave empty for equal split</span>
    </div>

    {#if showSubtotal}
      <label class="flex flex-col gap-1 text-sm">
        <span class="text-text-muted">Subtotal (before tax)</span>
        <input
          type="number"
          step="0.01"
          min="0"
          placeholder="0.00"
          bind:value={subtotalRaw}
          class="rounded-md border border-border px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
        />
      </label>
      <p class="text-sm text-text-muted">
        Tax &amp; fees: <strong class="tabular-nums">${taxAmount.toFixed(2)}</strong>
      </p>
    {/if}

    <div class="bf-items flex flex-col gap-3">
      {#each items as item (item.id)}
        <div class="bf-item-row rounded-md border border-border bg-surface-elevated p-3">
          <div class="flex flex-wrap items-center gap-2">
            <input
              type="text"
              bind:value={item.description}
              data-field="description"
              placeholder="Item description"
              onkeydown={handleItemKeydown}
              class="flex-1 min-w-[10rem] rounded-md border border-border px-3 py-1.5 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
            />
            <input
              type="number"
              step="0.01"
              min="0"
              bind:value={item.amountRaw}
              data-field="amount"
              placeholder="0.00"
              onkeydown={handleItemKeydown}
              class="w-28 rounded-md border border-border px-3 py-1.5 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
            />
            <button
              type="button"
              aria-label="Remove item"
              onclick={() => removeItemRow(item.id)}
              class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border text-text-muted hover:bg-surface-sunken"
            >
              <Trash2 size={16} />
            </button>
          </div>

          <div class="mt-2 flex flex-wrap gap-x-3 gap-y-1">
            {#if validParticipants.length === 0}
              <span class="text-xs text-text-muted">Add participants first</span>
            {:else}
              {#each validParticipants as p (p.id)}
                <label class="inline-flex items-center gap-1 text-sm">
                  <input
                    type="checkbox"
                    checked={item.participantNames.includes(p.displayName)}
                    onchange={() => toggleAssignment(item.id, p.displayName)}
                    class="rounded border-border text-primary focus:ring-primary"
                  />
                  <span>{p.displayName}</span>
                </label>
              {/each}
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <button
      type="button"
      onclick={() => addItemRow()}
      class="mt-1 inline-flex items-center gap-1 self-start rounded-md border border-dashed border-border px-3 py-1.5 text-sm font-medium text-text-muted hover:bg-surface-sunken"
    >
      <Plus size={14} /> Add item
    </button>
  </section>
</div>

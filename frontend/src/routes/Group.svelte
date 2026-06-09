<script lang="ts">
  import { fade, slide } from 'svelte/transition';
  import { link, querystring } from 'svelte-spa-router';
  import {
    ArrowLeft,
    Plus,
    Trash2,
    Receipt,
    BadgeCheck,
    HandCoins,
    Users,
  } from 'lucide-svelte';
  import {
    deleteSettlement,
    getGroup,
    getGroupBalances,
    listSettlements,
    recordSettlement,
  } from '$lib/api/groups';
  import { deleteBill, listBillsByGroup } from '$lib/api/split';
  import type {
    BillSummary,
    GetGroupBalancesResponse,
    Group,
    GroupMember,
    Settlement,
  } from '$lib/api/types';
  import { toasts } from '$lib/stores/toast';
  import { apiMessage } from '$lib/api/client';
  import { formatDate, formatMoney } from '$lib/util/format';
  import Modal from '$lib/components/Modal.svelte';

  interface Props {
    params?: { id?: string };
  }

  let { params }: Props = $props();
  let groupId = $derived(params?.id ?? '');

  let group = $state<Group | null>(null);
  let balances = $state<GetGroupBalancesResponse | null>(null);
  let bills = $state<BillSummary[]>([]);
  let settlements = $state<Settlement[]>([]);

  let groupLoading = $state(true);
  let balancesLoading = $state(true);
  let billsLoading = $state(true);
  let settlementsLoading = $state(true);

  type BalanceView = 'total' | 'detailed';
  let balanceView = $state<BalanceView>('total');

  let settlementOpen = $state(false);
  let settleFrom = $state('');
  let settleTo = $state('');
  let settleAmount = $state('');
  let settleNote = $state('');
  let settleError = $state('');
  let settleSaving = $state(false);

  // The settlement deep-link (?settleFrom=…) must fire at most once per groupId.
  // Reading $querystring inside an effect would otherwise re-open the modal on
  // every URL change.
  const deepLinkAppliedFor = new Set<string>();

  let lastLoadedId = '';
  $effect(() => {
    const id = groupId;
    if (!id || id === lastLoadedId) return;
    lastLoadedId = id;
    void initForGroup(id);
  });

  async function initForGroup(id: string): Promise<void> {
    await Promise.all([loadGroup(id), loadBalances(id), loadBills(id), loadSettlements(id)]);
    if (deepLinkAppliedFor.has(id)) return;
    deepLinkAppliedFor.add(id);
    applyQueryDeepLink();
  }

  function applyQueryDeepLink(): void {
    const qs = $querystring ?? '';
    if (!qs) return;
    const search = new URLSearchParams(qs);
    const from = search.get('settleFrom');
    const to = search.get('settleTo');
    const amount = search.get('amount');
    if (from || to || amount) {
      openSettlement({ from, to, amount });
    }
  }

  /** Map a value that might be a userId or displayName to a matching member's displayName. */
  function resolveMemberName(value: string | null | undefined): string {
    if (!value) return '';
    const members = group?.members ?? [];
    const hit = members.find((m) => m.userId === value || m.displayName === value);
    return hit?.displayName ?? value;
  }

  async function loadGroup(id: string): Promise<void> {
    groupLoading = true;
    try {
      const r = await getGroup(id);
      group = r.group ?? null;
    } catch (e) {
      group = null;
      toasts.error(apiMessage(e, 'Failed to load group.'));
    } finally {
      groupLoading = false;
    }
  }

  async function loadBalances(id: string): Promise<void> {
    balancesLoading = true;
    try {
      balances = await getGroupBalances(id);
    } catch (e) {
      balances = null;
      toasts.error(apiMessage(e, 'Failed to load balances.'));
    } finally {
      balancesLoading = false;
    }
  }

  async function loadBills(id: string): Promise<void> {
    billsLoading = true;
    try {
      const r = await listBillsByGroup(id);
      bills = r.bills ?? [];
    } catch (e) {
      bills = [];
      toasts.error(apiMessage(e, 'Failed to load bills.'));
    } finally {
      billsLoading = false;
    }
  }

  async function loadSettlements(id: string): Promise<void> {
    settlementsLoading = true;
    try {
      const r = await listSettlements(id);
      settlements = r.settlements ?? [];
    } catch (e) {
      settlements = [];
      toasts.error(apiMessage(e, 'Failed to load settlements.'));
    } finally {
      settlementsLoading = false;
    }
  }

  function openSettlement(prefill?: {
    from?: string | null;
    to?: string | null;
    amount?: string | null;
  }): void {
    if (!group || (group.members ?? []).length === 0) {
      toasts.error('Group has no members yet.');
      return;
    }
    settleError = '';
    settleFrom = resolveMemberName(prefill?.from);
    settleTo = resolveMemberName(prefill?.to);
    settleAmount = prefill?.amount ?? '';
    settleNote = '';
    settlementOpen = true;
  }

  function closeSettlement(): void {
    settlementOpen = false;
  }

  async function submitSettlement(e: SubmitEvent): Promise<void> {
    e.preventDefault();
    settleError = '';
    if (!settleFrom || !settleTo) {
      settleError = 'Please select both payer and recipient.';
      return;
    }
    if (settleFrom === settleTo) {
      settleError = 'Payer and recipient must be different.';
      return;
    }
    const amount = parseFloat(settleAmount);
    if (isNaN(amount) || amount <= 0) {
      settleError = 'Please enter a valid amount.';
      return;
    }
    settleSaving = true;
    try {
      await recordSettlement({
        groupId,
        fromUserId: settleFrom,
        toUserId: settleTo,
        amount,
        note: settleNote || undefined,
      });
      toasts.success('Settlement recorded.');
      closeSettlement();
      await Promise.all([loadBalances(groupId), loadSettlements(groupId)]);
    } catch (err) {
      settleError = apiMessage(err, 'Failed to record settlement.');
    } finally {
      settleSaving = false;
    }
  }

  async function handleDeleteBill(bill: BillSummary): Promise<void> {
    if (!confirm(`Delete "${bill.title}"? This cannot be undone.`)) return;
    try {
      await deleteBill(bill.billId);
      toasts.success('Bill deleted.');
      await Promise.all([loadBalances(groupId), loadBills(groupId)]);
    } catch (err) {
      toasts.error(apiMessage(err, 'Failed to delete bill.'));
    }
  }

  async function handleDeleteSettlement(s: Settlement): Promise<void> {
    if (!confirm('Delete this settlement? This cannot be undone.')) return;
    try {
      await deleteSettlement(s.id);
      toasts.success('Settlement deleted.');
      await Promise.all([loadBalances(groupId), loadSettlements(groupId)]);
    } catch (err) {
      toasts.error(apiMessage(err, 'Failed to delete settlement.'));
    }
  }

  type BalanceSign = 'pos' | 'neg' | 'zero';
  const EPSILON = 0.005;
  function signOf(net: number): BalanceSign {
    if (net > EPSILON) return 'pos';
    if (net < -EPSILON) return 'neg';
    return 'zero';
  }
  const NET_CLASS: Record<BalanceSign, string> = {
    pos: 'text-success',
    neg: 'text-rose-700',
    zero: 'text-text-muted',
  };
  const NET_LABEL: Record<BalanceSign, string> = {
    pos: 'is owed',
    neg: 'owes',
    zero: 'settled',
  };

  function signedAmount(net: number): string {
    const sign = signOf(net);
    const prefix = sign === 'pos' ? '+' : sign === 'neg' ? '−' : '';
    return `${prefix}${formatMoney(Math.abs(net))}`;
  }

  function memberKey(m: GroupMember, fallback: string | number): string {
    return m.userId ?? `name:${m.displayName}:${fallback}`;
  }

  let groupMembers = $derived(group?.members ?? []);
  let hasMembers = $derived(groupMembers.length > 0);
  let memberBalances = $derived(balances?.memberBalances ?? []);
  let debtMatrix = $derived(balances?.debtMatrix ?? []);
</script>

<main class="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-6">
  <nav class="flex items-center justify-between gap-2 text-sm">
    <a
      use:link
      href="/groups"
      class="inline-flex items-center gap-1 text-text-muted hover:text-text"
    >
      <ArrowLeft size={14} /> Back to Groups
    </a>
    <a
      use:link
      href={`/?group=${groupId}`}
      class="inline-flex items-center gap-1 rounded-md border border-border bg-surface-elevated px-3 py-1.5 text-sm font-medium text-text hover:bg-surface-sunken"
    >
      <Plus size={14} /> New Bill
    </a>
  </nav>

  <header class="flex flex-col gap-1">
    {#if groupLoading}
      <div class="h-8 w-48 animate-pulse rounded bg-surface-elevated ring-1 ring-border"></div>
      <div class="mt-2 h-4 w-72 animate-pulse rounded bg-surface-elevated ring-1 ring-border"></div>
    {:else if group}
      <h1 class="display-wonk text-3xl font-semibold text-text">{group.name}</h1>
      <div class="flex flex-wrap items-center gap-1.5 text-sm text-text-muted">
        <Users size={14} class="text-text-subtle" />
        {#if hasMembers}
          <span>{groupMembers.map((m) => m.displayName).filter(Boolean).join(', ')}</span>
        {:else}
          <span class="italic text-text-subtle">No members</span>
        {/if}
      </div>
    {:else}
      <p class="text-text-muted">Group not found.</p>
    {/if}
  </header>

  <!-- Balances -->
  <section class="flex flex-col gap-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="text-lg font-semibold text-text">Balances</h2>
      <div
        role="tablist"
        class="inline-flex overflow-hidden rounded-md border border-border bg-surface-elevated"
      >
        {#each [{ key: 'total', label: 'Total Balances' }, { key: 'detailed', label: 'Detailed Debts' }] as tab, i (tab.key)}
          {@const active = balanceView === tab.key}
          <button
            type="button"
            role="tab"
            aria-selected={active}
            onclick={() => (balanceView = tab.key as BalanceView)}
            class={[
              'px-3 py-1.5 text-xs font-medium transition-colors',
              i > 0 && 'border-l border-border',
              active ? 'bg-primary text-white' : 'text-text-muted hover:bg-surface-sunken',
            ]}
          >
            {tab.label}
          </button>
        {/each}
      </div>
    </div>

    {#if balancesLoading}
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {#each Array(3) as _, i (i)}
          <div class="h-24 animate-pulse rounded-lg bg-surface-elevated ring-1 ring-border"></div>
        {/each}
      </div>
    {:else if memberBalances.length === 0}
      <div
        class="flex flex-col items-center gap-2 rounded-lg bg-surface-elevated px-6 py-8 text-center ring-1 ring-border"
      >
        <Receipt size={26} class="text-text-subtle" />
        <p class="text-text-muted">
          No bills yet. <a use:link href={`/?group=${groupId}`} class="text-primary hover:underline">Create the first bill</a>.
        </p>
      </div>
    {:else if balanceView === 'total'}
      <ul class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {#each memberBalances as bal (bal.userId)}
          {@const net = bal.netBalance ?? 0}
          {@const paid = bal.totalPaid ?? 0}
          {@const owed = bal.totalOwed ?? 0}
          {@const sign = signOf(net)}
          <li class="rounded-lg bg-surface-elevated p-4 ring-1 ring-border">
            <div class="flex items-baseline justify-between gap-2">
              <h3 class="font-medium text-text">{bal.displayName || bal.userId}</h3>
              <span class="text-xs text-text-muted">{NET_LABEL[sign]}</span>
            </div>
            <div class="mt-1 text-2xl font-semibold tabular-nums {NET_CLASS[sign]}">
              {signedAmount(net)}
            </div>
            <div class="mt-2 grid grid-cols-2 gap-2 border-t border-border pt-2 text-xs text-text-muted">
              <span>Paid <span class="tabular-nums text-text">{formatMoney(paid)}</span></span>
              <span>Owes <span class="tabular-nums text-text">{formatMoney(owed)}</span></span>
            </div>
          </li>
        {/each}
      </ul>
    {:else if debtMatrix.length === 0}
      <p class="rounded-md bg-surface-elevated px-4 py-3 text-sm text-text-muted ring-1 ring-border">
        All settled up — nothing to pay off in this group.
      </p>
    {:else}
      <div class="overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
        <div
          class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-xs font-medium uppercase tracking-wide text-text-muted sm:grid sm:grid-cols-[1fr_1fr_auto_auto] sm:gap-4"
        >
          <span>Debtor</span>
          <span>Owes To</span>
          <span class="text-right">Amount</span>
          <span class="text-right">Action</span>
        </div>
        <ul class="divide-y divide-border">
          {#each debtMatrix as debt (debt.fromUserId + '->' + debt.toUserId)}
            {@const amount = debt.amount ?? 0}
            <li
              class="grid grid-cols-1 gap-1 px-4 py-3 sm:grid-cols-[1fr_1fr_auto_auto] sm:items-center sm:gap-4"
            >
              <span class="font-medium text-text">{debt.fromName || debt.fromUserId}</span>
              <span class="text-text">{debt.toName || debt.toUserId}</span>
              <span class="text-sm tabular-nums text-text sm:text-right">
                {formatMoney(amount)}
              </span>
              <span class="sm:text-right">
                <button
                  type="button"
                  onclick={() =>
                    openSettlement({
                      from: debt.fromName || debt.fromUserId,
                      to: debt.toName || debt.toUserId,
                      amount: amount.toFixed(2),
                    })}
                  class="inline-flex items-center gap-1 rounded-md border border-success/30 bg-success-soft px-2.5 py-1 text-xs font-medium text-success hover:bg-success-soft"
                >
                  <HandCoins size={12} /> Settle
                </button>
              </span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </section>

  <!-- Settlements -->
  <section class="flex flex-col gap-3">
    <div class="flex items-center justify-between gap-2">
      <h2 class="text-lg font-semibold text-text">Settlements</h2>
      <button
        type="button"
        onclick={() => openSettlement()}
        disabled={!hasMembers}
        class="inline-flex items-center gap-1 rounded-md border border-border bg-surface-elevated px-3 py-1.5 text-sm font-medium text-text hover:bg-surface-sunken disabled:opacity-60"
      >
        <Plus size={14} /> Record Settlement
      </button>
    </div>

    {#if settlementsLoading}
      <div class="h-16 animate-pulse rounded-lg bg-surface-elevated ring-1 ring-border"></div>
    {:else if settlements.length === 0}
      <p
        class="rounded-md bg-surface-elevated px-4 py-3 text-sm italic text-text-muted ring-1 ring-border"
      >
        No settlements recorded yet.
      </p>
    {:else}
      <div class="overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
        <div
          class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-xs font-medium uppercase tracking-wide text-text-muted sm:grid sm:grid-cols-[1fr_1fr_auto_2fr_auto_auto] sm:gap-4"
        >
          <span>From</span>
          <span>To</span>
          <span class="text-right">Amount</span>
          <span>Note</span>
          <span class="text-right">Date</span>
          <span></span>
        </div>
        <ul class="divide-y divide-border">
          {#each settlements as s (s.id)}
            {@const amount = s.amount ?? 0}
            <li
              class="grid grid-cols-1 gap-1 px-4 py-3 text-sm sm:grid-cols-[1fr_1fr_auto_2fr_auto_auto] sm:items-center sm:gap-4"
            >
              <span class="font-medium text-text">{s.fromName || s.fromUserId}</span>
              <span class="text-text">{s.toName || s.toUserId}</span>
              <span class="tabular-nums text-text sm:text-right">{formatMoney(amount)}</span>
              <span class="text-text-muted">
                {#if s.note}{s.note}{:else}<em class="text-text-subtle">—</em>{/if}
              </span>
              <span class="text-xs text-text-muted sm:text-right">{formatDate(s.createdAt)}</span>
              <span class="sm:text-right">
                <button
                  type="button"
                  onclick={() => handleDeleteSettlement(s)}
                  aria-label="Delete settlement"
                  title="Delete"
                  class="rounded-md p-1.5 text-text-muted hover:bg-danger-soft hover:text-danger"
                >
                  <Trash2 size={14} />
                </button>
              </span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </section>

  <!-- Bills -->
  <section class="flex flex-col gap-3">
    <h2 class="text-lg font-semibold text-text">Bills</h2>

    {#if billsLoading}
      <div class="h-16 animate-pulse rounded-lg bg-surface-elevated ring-1 ring-border"></div>
    {:else if bills.length === 0}
      <p
        class="rounded-md bg-surface-elevated px-4 py-3 text-sm italic text-text-muted ring-1 ring-border"
      >
        No bills yet in this group.
      </p>
    {:else}
      <div class="overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
        <div
          class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-xs font-medium uppercase tracking-wide text-text-muted sm:grid sm:grid-cols-[1fr_auto_1fr_auto_auto] sm:gap-4"
        >
          <span>Bill</span>
          <span class="text-right">Total</span>
          <span>Paid By</span>
          <span class="text-right">Date</span>
          <span></span>
        </div>
        <ul class="divide-y divide-border">
          {#each bills as bill (bill.billId)}
            <li
              class="grid grid-cols-1 gap-1 px-4 py-3 text-sm sm:grid-cols-[1fr_auto_1fr_auto_auto] sm:items-center sm:gap-4"
            >
              <a
                use:link
                href={`/bill/${bill.billId}`}
                class="font-medium text-text hover:text-primary"
              >
                {bill.title || 'Untitled'}
              </a>
              <span class="tabular-nums text-text sm:text-right">{formatMoney(bill.total)}</span>
              <span class="text-text-muted">
                {#if bill.payerId}{bill.payerId}{:else}<em class="text-text-subtle">Not recorded</em>{/if}
              </span>
              <span class="text-xs text-text-muted sm:text-right">{formatDate(bill.createdAt)}</span>
              <span class="sm:text-right">
                <button
                  type="button"
                  onclick={() => handleDeleteBill(bill)}
                  aria-label="Delete bill"
                  title="Delete"
                  class="rounded-md p-1.5 text-text-muted hover:bg-danger-soft hover:text-danger"
                >
                  <Trash2 size={14} />
                </button>
              </span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </section>
</main>

<Modal open={settlementOpen} title="Record Settlement" onClose={closeSettlement} maxWidth="max-w-md">
  <form id="settlement-form" class="flex flex-col gap-4" onsubmit={submitSettlement}>
    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Who paid?</span>
      <select
        bind:value={settleFrom}
        required
        class="rounded-md border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
      >
        <option value="">Select payer…</option>
        {#each groupMembers as m, i (memberKey(m, i))}
          <option value={m.displayName}>{m.displayName}</option>
        {/each}
      </select>
    </label>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Who received?</span>
      <select
        bind:value={settleTo}
        required
        class="rounded-md border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
      >
        <option value="">Select recipient…</option>
        {#each groupMembers as m, i (memberKey(m, i))}
          <option value={m.displayName}>{m.displayName}</option>
        {/each}
      </select>
    </label>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Amount</span>
      <input
        type="number"
        step="0.01"
        min="0.01"
        bind:value={settleAmount}
        placeholder="0.00"
        required
        class="rounded-md border border-border bg-surface-elevated px-3 py-2 tabular-nums outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
      />
    </label>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Note <span class="text-text-subtle">(optional)</span></span>
      <input
        type="text"
        bind:value={settleNote}
        placeholder="e.g. Venmo payment"
        class="rounded-md border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
      />
    </label>

    {#if settleError}
      <div
        role="alert"
        class="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700"
        transition:fade={{ duration: 100 }}
      >
        {settleError}
      </div>
    {/if}
  </form>

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <button
        type="button"
        onclick={closeSettlement}
        class="rounded-md border border-border bg-surface-elevated px-3 py-1.5 text-sm text-text hover:bg-surface-sunken"
      >
        Cancel
      </button>
      <button
        type="submit"
        form="settlement-form"
        disabled={settleSaving}
        class="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-hover disabled:opacity-60"
      >
        {settleSaving ? 'Recording…' : 'Record Settlement'}
      </button>
    </div>
  {/snippet}
</Modal>

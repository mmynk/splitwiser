<script lang="ts">
  import { slide } from 'svelte/transition';
  import { flip } from 'svelte/animate';
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
  import { confirmAction } from '$lib/stores/confirm';
  import { apiMessage } from '$lib/api/client';
  import { formatDate, formatMoney } from '$lib/util/format';
  import { dur, durFast, ease } from '$lib/motion';
  import Modal from '$lib/components/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import IconButton from '$lib/components/ui/IconButton.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Amount from '$lib/components/ui/Amount.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import EmptyState from '$lib/components/ui/EmptyState.svelte';
  import Alert from '$lib/components/ui/Alert.svelte';

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
    const ok = await confirmAction({
      title: `Delete "${bill.title || 'this bill'}"?`,
      body: "Can't be undone.",
      confirmLabel: 'Delete',
      tone: 'danger',
    });
    if (!ok) return;
    try {
      await deleteBill(bill.billId);
      toasts.success('Bill deleted.');
      await Promise.all([loadBalances(groupId), loadBills(groupId)]);
    } catch (err) {
      toasts.error(apiMessage(err, 'Could not delete the bill.'));
    }
  }

  async function handleDeleteSettlement(s: Settlement): Promise<void> {
    const ok = await confirmAction({
      title: 'Delete this settlement?',
      body: "Balances revert to before this payment. Can't be undone.",
      confirmLabel: 'Delete',
      tone: 'danger',
    });
    if (!ok) return;
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
    neg: 'text-danger',
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

<main class="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6">
  <nav class="flex items-center justify-between gap-2 text-sm">
    <a
      use:link
      href="/groups"
      class="inline-flex items-center gap-1 text-text-muted transition-colors hover:text-text"
    >
      <ArrowLeft size={14} strokeWidth={1.75} /> Back to groups
    </a>
    <a use:link href={`/?group=${groupId}`}>
      <Button variant="secondary" size="sm">
        <Plus size={14} strokeWidth={1.75} /> New bill
      </Button>
    </a>
  </nav>

  <header class="flex flex-col gap-1">
    {#if groupLoading}
      <Skeleton height="h-8" width="w-48" />
      <Skeleton height="h-4" width="w-72" />
    {:else if group}
      <h1 class="display-wonk text-3xl font-semibold text-text">{group.name}</h1>
      <div class="flex flex-wrap items-center gap-1.5 text-[0.875rem] text-text-muted">
        <Users size={14} strokeWidth={1.75} class="text-text-subtle" />
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
      <h2 class="font-serif text-lg font-semibold text-text">Balances</h2>
      <div
        role="tablist"
        class="inline-flex overflow-hidden rounded-input border border-border bg-surface-elevated"
      >
        {#each [{ key: 'total', label: 'Total' }, { key: 'detailed', label: 'Detailed' }] as tab, i (tab.key)}
          {@const active = balanceView === tab.key}
          <button
            type="button"
            role="tab"
            aria-selected={active}
            onclick={() => (balanceView = tab.key as BalanceView)}
            class={[
              'px-3 py-1.5 text-[0.75rem] font-medium transition-colors',
              i > 0 && 'border-l border-border',
              active ? 'bg-primary text-primary-foreground' : 'text-text-muted hover:bg-surface-sunken hover:text-text',
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
          <Skeleton height="h-24" rounded="card" />
        {/each}
      </div>
    {:else if memberBalances.length === 0}
      <EmptyState
        icon={Receipt}
        title="Nothing to settle. Yet."
        hint="Once a bill lands here, balances appear."
      >
        <a use:link href={`/?group=${groupId}`}>
          <Button variant="primary" size="sm">
            <Plus size={14} strokeWidth={1.75} /> Add the first bill
          </Button>
        </a>
      </EmptyState>
    {:else if balanceView === 'total'}
      <ul class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {#each memberBalances as bal (bal.userId)}
          {@const net = bal.netBalance ?? 0}
          {@const paid = bal.totalPaid ?? 0}
          {@const owed = bal.totalOwed ?? 0}
          {@const sign = signOf(net)}
          <li animate:flip={{ duration: durFast }}>
            <Card padding="sm">
              <div class="flex items-baseline justify-between gap-2">
                <h3 class="font-medium text-text">{bal.displayName || bal.userId}</h3>
                <span class="text-[0.75rem] text-text-muted">{NET_LABEL[sign]}</span>
              </div>
              <div class="mt-1 {NET_CLASS[sign]}">
                <Amount value={Math.abs(net)} signed={false} size="xl" animate />
              </div>
              <div class="mt-2 grid grid-cols-2 gap-2 border-t border-border pt-2 text-[0.75rem] text-text-muted">
                <span>Paid <span class="tabular-nums text-text">{formatMoney(paid)}</span></span>
                <span>Owes <span class="tabular-nums text-text">{formatMoney(owed)}</span></span>
              </div>
            </Card>
          </li>
        {/each}
      </ul>
    {:else if debtMatrix.length === 0}
      <EmptyState title="All square. Nice." hint="No outstanding debts in this group." />
    {:else}
      <div class="overflow-hidden rounded-card border border-border bg-surface-elevated">
        <div
          class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-wider text-text-muted sm:grid sm:grid-cols-[1fr_1fr_auto_auto] sm:gap-4"
        >
          <span>Debtor</span>
          <span>Owes to</span>
          <span class="text-right">Amount</span>
          <span class="text-right">Action</span>
        </div>
        <ul class="divide-y divide-border">
          {#each debtMatrix as debt (debt.fromUserId + '->' + debt.toUserId)}
            {@const amount = debt.amount ?? 0}
            <li
              animate:flip={{ duration: durFast }}
              class="grid grid-cols-1 gap-1 px-4 py-3 sm:grid-cols-[1fr_1fr_auto_auto] sm:items-center sm:gap-4"
            >
              <span class="font-medium text-text">{debt.fromName || debt.fromUserId}</span>
              <span class="text-text">{debt.toName || debt.toUserId}</span>
              <span class="text-[0.875rem] tabular-nums text-text sm:text-right">
                {formatMoney(amount)}
              </span>
              <span class="sm:text-right">
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() =>
                    openSettlement({
                      from: debt.fromName || debt.fromUserId,
                      to: debt.toName || debt.toUserId,
                      amount: amount.toFixed(2),
                    })}
                >
                  <HandCoins size={12} strokeWidth={1.75} /> Settle
                </Button>
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
      <h2 class="font-serif text-lg font-semibold text-text">Settlements</h2>
      <Button variant="secondary" size="sm" onclick={() => openSettlement()} disabled={!hasMembers}>
        <Plus size={14} strokeWidth={1.75} /> Record settlement
      </Button>
    </div>

    {#if settlementsLoading}
      <Skeleton height="h-16" rounded="card" />
    {:else if settlements.length === 0}
      <EmptyState title="No settlements logged." hint="When someone pays someone back, it shows up here." />
    {:else}
      <div class="overflow-hidden rounded-card border border-border bg-surface-elevated">
        <div
          class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-wider text-text-muted sm:grid sm:grid-cols-[1fr_1fr_auto_2fr_auto_auto] sm:gap-4"
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
              class="grid grid-cols-1 gap-1 px-4 py-3 text-[0.875rem] sm:grid-cols-[1fr_1fr_auto_2fr_auto_auto] sm:items-center sm:gap-4"
            >
              <span class="font-medium text-text">{s.fromName || s.fromUserId}</span>
              <span class="text-text">{s.toName || s.toUserId}</span>
              <span class="tabular-nums text-text sm:text-right">{formatMoney(amount)}</span>
              <span class="text-text-muted">
                {#if s.note}{s.note}{:else}<em class="text-text-subtle">—</em>{/if}
              </span>
              <span class="text-[0.75rem] text-text-muted sm:text-right">{formatDate(s.createdAt)}</span>
              <span class="sm:text-right">
                <IconButton
                  ariaLabel="Delete settlement"
                  title="Delete"
                  size="sm"
                  variant="danger"
                  onclick={() => handleDeleteSettlement(s)}
                >
                  <Trash2 size={14} strokeWidth={1.75} />
                </IconButton>
              </span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </section>

  <!-- Bills -->
  <section class="flex flex-col gap-3">
    <h2 class="font-serif text-lg font-semibold text-text">Bills</h2>

    {#if billsLoading}
      <Skeleton height="h-16" rounded="card" />
    {:else if bills.length === 0}
      <EmptyState
        icon={Receipt}
        title="No bills here yet."
        hint="The first one's always the awkward one — just add it."
      >
        <a use:link href={`/?group=${groupId}`}>
          <Button variant="primary" size="sm">
            <Plus size={14} strokeWidth={1.75} /> Add the first bill
          </Button>
        </a>
      </EmptyState>
    {:else}
      <div class="overflow-hidden rounded-card border border-border bg-surface-elevated">
        <div
          class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-wider text-text-muted sm:grid sm:grid-cols-[1fr_auto_1fr_auto_auto] sm:gap-4"
        >
          <span>Bill</span>
          <span class="text-right">Total</span>
          <span>Paid by</span>
          <span class="text-right">Date</span>
          <span></span>
        </div>
        <ul class="divide-y divide-border">
          {#each bills as bill (bill.billId)}
            <li
              class="grid grid-cols-1 gap-1 px-4 py-3 text-[0.875rem] sm:grid-cols-[1fr_auto_1fr_auto_auto] sm:items-center sm:gap-4"
            >
              <a
                use:link
                href={`/bill/${bill.billId}`}
                class="font-medium text-text transition-colors hover:text-primary"
              >
                {bill.title || 'Untitled'}
              </a>
              <span class="tabular-nums text-text sm:text-right">{formatMoney(bill.total)}</span>
              <span class="text-text-muted">
                {#if bill.payerId}{bill.payerId}{:else}<em class="text-text-subtle">Not recorded</em>{/if}
              </span>
              <span class="text-[0.75rem] text-text-muted sm:text-right">{formatDate(bill.createdAt)}</span>
              <span class="sm:text-right">
                <IconButton
                  ariaLabel="Delete bill"
                  title="Delete"
                  size="sm"
                  variant="danger"
                  onclick={() => handleDeleteBill(bill)}
                >
                  <Trash2 size={14} strokeWidth={1.75} />
                </IconButton>
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
        class="rounded-input border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
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
        class="rounded-input border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
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
        class="rounded-input border border-border bg-surface-elevated px-3 py-2 tabular-nums outline-none focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
      />
    </label>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Note <span class="text-text-subtle">(optional)</span></span>
      <input
        type="text"
        bind:value={settleNote}
        placeholder="e.g. Venmo payment"
        class="rounded-input border border-border bg-surface-elevated px-3 py-2 outline-none focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
      />
    </label>

    {#if settleError}
      <Alert>{settleError}</Alert>
    {/if}
  </form>

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button variant="ghost" size="sm" onclick={closeSettlement}>Cancel</Button>
      <Button type="submit" form="settlement-form" size="sm" loading={settleSaving}>
        {settleSaving ? 'Recording…' : 'Record settlement'}
      </Button>
    </div>
  {/snippet}
</Modal>

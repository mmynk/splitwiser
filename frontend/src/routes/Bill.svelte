<script lang="ts">
  import { onMount } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { slide } from 'svelte/transition';
  import { Copy, Check, Pencil, Trash2, ArrowLeft } from 'lucide-svelte';
  import { getBill, updateBill, deleteBill } from '$lib/api/split';
  import { ApiError } from '$lib/api/client';
  import { toasts } from '$lib/stores/toast';
  import { confirmAction } from '$lib/stores/confirm';
  import { currentUser } from '$lib/stores/auth';
  import { formatMoney, formatDateTime } from '$lib/util/format';
  import { dur, durFast, ease } from '$lib/motion';
  import type { GetBillResponse } from '$lib/api/types';
  import BillForm from '$lib/components/BillForm.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Amount from '$lib/components/ui/Amount.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Alert from '$lib/components/ui/Alert.svelte';

  interface Props {
    params?: { id?: string };
  }

  let { params }: Props = $props();
  let billId = $derived(params?.id ?? '');

  let bill = $state<GetBillResponse | null>(null);
  let loading = $state(true);
  let loadError = $state('');
  let editMode = $state(false);
  let saving = $state(false);
  let deleting = $state(false);
  let editTitle = $state('');
  let editError = $state('');
  let copied = $state(false);
  let copyTimer: ReturnType<typeof setTimeout> | null = null;

  let billForm: BillForm | null = $state(null);

  onMount(() => {
    if (billId) loadBill();
    return () => {
      if (copyTimer) clearTimeout(copyTimer);
    };
  });

  async function loadBill(): Promise<void> {
    loading = true;
    loadError = '';
    try {
      bill = await getBill(billId);
      editTitle = bill.title ?? '';
    } catch (e) {
      loadError = e instanceof ApiError ? e.message : 'Failed to load bill.';
      bill = null;
    } finally {
      loading = false;
    }
  }

  function buildSummaryText(b: GetBillResponse): string {
    const title = b.title || 'Bill';
    const total = b.total ?? 0;
    const subtotal = b.subtotal ?? total;
    const tax = total - subtotal;
    const splits = b.split?.splits ?? {};
    const participants = b.participants ?? [];

    const lines: string[] = [];
    lines.push(`${title} — ${formatMoney(total)}`);
    if (b.payerId) lines.push(`Paid by ${b.payerId}`);
    lines.push('');
    lines.push(`Subtotal: ${formatMoney(subtotal)}`);
    lines.push(`Tax & fees: ${formatMoney(tax)}`);
    lines.push('');
    lines.push('Splits:');
    for (const p of participants) {
      const name = p.displayName;
      const s = splits[name] ?? {};
      lines.push(`  ${name}: ${formatMoney(s.total ?? 0)}`);
    }
    return lines.join('\n');
  }

  async function copySummary(): Promise<void> {
    if (!bill) return;
    const text = buildSummaryText(bill);
    try {
      await navigator.clipboard.writeText(text);
      copied = true;
      if (copyTimer) clearTimeout(copyTimer);
      copyTimer = setTimeout(() => {
        copied = false;
        copyTimer = null;
      }, 2000);
    } catch {
      toasts.error('Could not copy to clipboard.');
    }
  }

  function enterEdit(): void {
    if (!bill) return;
    editTitle = bill.title ?? '';
    editError = '';
    editMode = true;
  }

  function cancelEdit(): void {
    editMode = false;
    editError = '';
  }

  async function saveEdit(): Promise<void> {
    editError = '';
    if (!billForm || !bill) return;
    const data = billForm.getData();
    if (data.participants.length === 0) {
      editError = 'Please add at least one participant with a name.';
      return;
    }
    if (data.total <= 0) {
      editError = 'Please enter a valid total amount.';
      return;
    }
    saving = true;
    try {
      await updateBill({
        billId,
        title: editTitle.trim(),
        total: data.total,
        subtotal: data.subtotal,
        items: data.items,
        participants: data.participants,
        payerId: data.payerId || undefined,
        groupId: bill.groupId || undefined,
      });
      toasts.success('Bill updated.');
      editMode = false;
      await loadBill();
    } catch (e) {
      editError = e instanceof ApiError ? e.message : 'Failed to save changes.';
    } finally {
      saving = false;
    }
  }

  async function confirmDelete(): Promise<void> {
    if (!bill) return;
    const ok = await confirmAction({
      title: 'Delete this bill?',
      body: "Can't be undone.",
      confirmLabel: 'Delete',
      tone: 'danger',
    });
    if (!ok) return;
    deleting = true;
    try {
      await deleteBill(billId);
      toasts.success('Bill deleted.');
      if (bill.groupId) push(`/group/${bill.groupId}`);
      else push('/');
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : 'Could not delete the bill.';
      toasts.error(msg);
      deleting = false;
    }
  }

  function buildInitial(b: GetBillResponse) {
    return {
      total: b.total ?? 0,
      subtotal: b.subtotal ?? b.total ?? 0,
      participants: (b.participants ?? []).map((p) => ({
        displayName: p.displayName,
        userId: p.userId,
      })),
      items: (b.items ?? []).map((it) => ({
        description: it.description,
        amount: it.amount ?? 0,
        participantIds: [...(it.participantIds ?? [])],
      })),
      payerId: b.payerId ?? '',
      groupId: b.groupId ?? '',
    };
  }
</script>

<main class="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-6 sm:px-6">
  {#if loading}
    <div class="flex flex-col gap-3">
      <Skeleton height="h-8" width="w-1/3" />
      <Skeleton height="h-24" rounded="card" />
      <Skeleton height="h-48" rounded="card" />
    </div>
  {:else if loadError || !bill}
    <section class="flex flex-col items-start gap-3 rounded-card border border-border bg-surface-elevated p-6">
      <h1 class="font-serif text-2xl font-semibold text-text">Bill not found</h1>
      <p class="text-text-muted">{loadError || "This bill doesn't exist or has been deleted."}</p>
      <a href="#/">
        <Button variant="primary" size="sm">
          <ArrowLeft size={14} strokeWidth={1.75} /> Back to bills
        </Button>
      </a>
    </section>
  {:else}
    <!-- Header -->
    <header class="flex flex-wrap items-start justify-between gap-3">
      <div class="flex flex-col gap-1">
        <h1 class="font-serif text-3xl font-semibold text-text">{bill.title || 'Bill'}</h1>
        <p class="text-sm text-text-muted">{formatDateTime(bill.createdAt)}</p>
        {#if bill.groupId}
          <p class="text-sm">
            <span class="text-text-muted">Group:</span>
            <a class="text-primary hover:underline" href={`#/group/${bill.groupId}`}>
              {bill.groupName || 'View group'}
            </a>
          </p>
        {/if}
        {#if bill.payerId}
          <p class="text-sm">
            <span class="text-text-muted">Paid by:</span>
            <span class="font-medium text-text">{bill.payerId}</span>
          </p>
        {/if}
      </div>

      {#if !editMode}
        <div class="flex flex-wrap gap-2">
          <Button variant="secondary" size="sm" onclick={copySummary} ariaLabel={copied ? 'Copied' : 'Copy summary'}>
            {#if copied}
              <Check size={14} strokeWidth={1.75} /> <span class="hidden sm:inline">Copied</span>
            {:else}
              <Copy size={14} strokeWidth={1.75} /> <span class="hidden sm:inline">Copy summary</span>
            {/if}
          </Button>
          <Button variant="secondary" size="sm" onclick={enterEdit} ariaLabel="Edit">
            <Pencil size={14} strokeWidth={1.75} /> <span class="hidden sm:inline">Edit</span>
          </Button>
          <Button variant="danger" size="sm" onclick={confirmDelete} loading={deleting} ariaLabel="Delete">
            <Trash2 size={14} strokeWidth={1.75} /> <span class="hidden sm:inline">{deleting ? 'Deleting…' : 'Delete'}</span>
          </Button>
        </div>
      {/if}
    </header>

    {#if editMode}
      <section
        class="rounded-card border border-border bg-surface-elevated p-5"
        transition:slide={{ duration: dur, easing: ease }}
      >
        <div class="flex items-center justify-between">
          <h2 class="font-serif text-xl font-semibold text-text">Edit bill</h2>
          <Button variant="ghost" size="sm" onclick={cancelEdit}>Cancel</Button>
        </div>

        <div class="mt-4 flex flex-col gap-6">
          <label class="flex flex-col gap-1 text-sm">
            <span class="font-medium text-text">
              Bill name <span class="text-text-subtle">(optional)</span>
            </span>
            <input
              type="text"
              bind:value={editTitle}
              placeholder="e.g. Team lunch, grocery run"
              class="rounded-input border border-border bg-surface-elevated px-3 py-2 outline-none transition-colors focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
            />
          </label>

          <BillForm
            bind:this={billForm}
            currentUser={$currentUser}
            initial={buildInitial(bill)}
            showGroupSelector={false}
          />

          {#if editError}
            <Alert>{editError}</Alert>
          {/if}

          <div class="flex flex-wrap gap-2">
            <Button onclick={saveEdit} loading={saving}>
              {saving ? 'Saving…' : 'Save changes'}
            </Button>
            <Button variant="secondary" onclick={cancelEdit} disabled={saving}>Cancel</Button>
          </div>
        </div>
      </section>
    {:else}
      {@const subtotalVal = bill.subtotal ?? bill.total ?? 0}
      {@const taxVal = (bill.total ?? 0) - (bill.subtotal ?? bill.total ?? 0)}
      {@const totalVal = bill.total ?? 0}
      <section>
        <div class="sm:hidden">
          <Card padding="sm">
            <div class="flex items-baseline justify-between">
              <span class="text-[0.875rem] text-text-muted">Subtotal</span>
              <Amount value={subtotalVal} size="md" />
            </div>
            <div class="mt-1 flex items-baseline justify-between">
              <span class="text-[0.875rem] text-text-muted">Tax &amp; fees</span>
              <Amount value={taxVal} size="md" />
            </div>
            <div class="mt-2 flex items-baseline justify-between border-t border-border pt-2">
              <span class="font-medium text-text">Total</span>
              <Amount value={totalVal} size="lg" />
            </div>
          </Card>
        </div>
        <div class="hidden gap-3 sm:grid sm:grid-cols-3">
          <Card padding="sm">
            <div class="text-[0.6875rem] uppercase tracking-wider text-text-muted">Subtotal</div>
            <div class="mt-1"><Amount value={subtotalVal} size="lg" /></div>
          </Card>
          <Card padding="sm">
            <div class="text-[0.6875rem] uppercase tracking-wider text-text-muted">Tax &amp; fees</div>
            <div class="mt-1"><Amount value={taxVal} size="lg" /></div>
          </Card>
          <Card padding="sm">
            <div class="text-[0.6875rem] uppercase tracking-wider text-text-muted">Total</div>
            <div class="mt-1"><Amount value={totalVal} size="xl" /></div>
          </Card>
        </div>
      </section>

      {#if (bill.items ?? []).length > 0}
        <section class="flex flex-col gap-2">
          <h2 class="font-serif text-xl font-semibold text-text">Items</h2>
          <ul class="divide-y divide-border overflow-hidden rounded-card border border-border bg-surface-elevated">
            {#each bill.items as item, i (item.description + ':' + i)}
              <li class="flex items-start justify-between gap-3 px-4 py-3">
                <div class="flex flex-col">
                  <span class="font-medium text-text">{item.description || 'Item'}</span>
                  <span class="text-[0.8125rem] text-text-muted">
                    {(item.participantIds ?? []).join(', ') || '—'}
                  </span>
                </div>
                <Amount value={item.amount ?? 0} size="md" />
              </li>
            {/each}
          </ul>
        </section>
      {/if}

      <section class="flex flex-col gap-3">
        <h2 class="font-serif text-xl font-semibold text-text">Who owes what</h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each bill.participants ?? [] as p (p.displayName)}
            {@const raw = bill.split?.splits?.[p.displayName] ?? {}}
            {@const subT = raw.subtotal ?? 0}
            {@const taxT = raw.tax ?? 0}
            {@const totalT = raw.total ?? 0}
            {@const personItems = raw.items ?? []}
            <Card padding="sm">
              <div class="flex items-baseline justify-between gap-2">
                <h3 class="font-medium text-text">{p.displayName}</h3>
                <Amount value={totalT} size="lg" />
              </div>
              {#if personItems.length > 0}
                <ul class="mt-2 flex flex-col gap-1 text-[0.875rem]">
                  {#each personItems as it}
                    <li class="flex justify-between text-text-muted">
                      <span>{it.description}</span>
                      <span class="tabular-nums">{formatMoney(it.amount)}</span>
                    </li>
                  {/each}
                </ul>
              {/if}
              <div class="mt-3 border-t border-border pt-2 text-[0.75rem] text-text-muted">
                <div class="flex justify-between">
                  <span>Subtotal</span>
                  <span class="tabular-nums">{formatMoney(subT)}</span>
                </div>
                <div class="flex justify-between">
                  <span>Tax</span>
                  <span class="tabular-nums">{formatMoney(taxT)}</span>
                </div>
              </div>
            </Card>
          {/each}
        </div>
      </section>

      <section class="flex flex-col gap-2">
        <h2 class="font-serif text-xl font-semibold text-text">Participants</h2>
        <p class="text-text-muted">
          {(bill.participants ?? []).map((p) => p.displayName).join(', ')}
        </p>
      </section>
    {/if}
  {/if}
</main>

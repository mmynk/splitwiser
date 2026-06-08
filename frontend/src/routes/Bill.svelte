<script lang="ts">
  import { onMount } from 'svelte';
  import { push } from 'svelte-spa-router';
  import { fade, slide } from 'svelte/transition';
  import { Copy, Check, Pencil, Trash2, ArrowLeft } from 'lucide-svelte';
  import { getBill, updateBill, deleteBill } from '$lib/api/split';
  import { ApiError } from '$lib/api/client';
  import { toasts } from '$lib/stores/toast';
  import { currentUser } from '$lib/stores/auth';
  import { formatMoney, formatDateTime } from '$lib/util/format';
  import type { GetBillResponse } from '$lib/api/types';
  import BillForm from '$lib/components/BillForm.svelte';

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
    const ok = confirm('Delete this bill? This cannot be undone.');
    if (!ok) return;
    deleting = true;
    try {
      await deleteBill(billId);
      toasts.success('Bill deleted.');
      if (bill.groupId) push(`/group/${bill.groupId}`);
      else push('/');
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : 'Failed to delete bill.';
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

<main class="mx-auto flex max-w-4xl flex-col gap-6 px-4 py-6">
  {#if loading}
    <div class="flex flex-col gap-3">
      <div class="h-8 w-1/3 animate-pulse rounded bg-surface-elevated ring-1 ring-border"></div>
      <div class="h-24 animate-pulse rounded-lg bg-surface-elevated ring-1 ring-border"></div>
      <div class="h-48 animate-pulse rounded-lg bg-surface-elevated ring-1 ring-border"></div>
    </div>
  {:else if loadError || !bill}
    <section class="flex flex-col items-start gap-3 rounded-lg bg-surface-elevated p-6 ring-1 ring-border">
      <h1 class="text-2xl font-semibold text-text">Bill not found</h1>
      <p class="text-text-muted">{loadError || "This bill doesn't exist or has been deleted."}</p>
      <a
        href="#/"
        class="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-hover"
      >
        <ArrowLeft size={14} /> Back to bills
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
          <button
            type="button"
            onclick={copySummary}
            class="inline-flex items-center gap-1 rounded-md border border-border bg-surface-elevated px-3 py-1.5 text-sm font-medium text-text hover:bg-surface-sunken"
          >
            {#if copied}
              <Check size={14} /> Copied
            {:else}
              <Copy size={14} /> Copy summary
            {/if}
          </button>
          <button
            type="button"
            onclick={enterEdit}
            class="inline-flex items-center gap-1 rounded-md border border-border bg-surface-elevated px-3 py-1.5 text-sm font-medium text-text hover:bg-surface-sunken"
          >
            <Pencil size={14} /> Edit
          </button>
          <button
            type="button"
            onclick={confirmDelete}
            disabled={deleting}
            class="inline-flex items-center gap-1 rounded-md border border-danger/30 bg-surface-elevated px-3 py-1.5 text-sm font-medium text-danger hover:bg-danger-soft disabled:opacity-60"
          >
            <Trash2 size={14} /> {deleting ? 'Deleting…' : 'Delete'}
          </button>
        </div>
      {/if}
    </header>

    {#if editMode}
      <section
        class="rounded-lg bg-surface-elevated p-5 ring-1 ring-border"
        transition:slide={{ duration: 180 }}
      >
        <div class="flex items-center justify-between">
          <h2 class="text-xl font-semibold text-text">Edit bill</h2>
          <button
            type="button"
            onclick={cancelEdit}
            class="rounded-md px-2.5 py-1.5 text-sm font-medium text-text-muted hover:bg-surface-sunken"
          >
            Cancel
          </button>
        </div>

        <div class="mt-4 flex flex-col gap-6">
          <label class="flex flex-col gap-1 text-sm">
            <span class="font-medium text-text">
              Bill name <span class="text-text-subtle">(optional)</span>
            </span>
            <input
              type="text"
              bind:value={editTitle}
              placeholder="e.g. Team Lunch, Grocery Run"
              class="rounded-md border border-border px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
            />
          </label>

          <BillForm
            bind:this={billForm}
            currentUser={$currentUser}
            initial={buildInitial(bill)}
            showGroupSelector={false}
          />

          {#if editError}
            <div
              role="alert"
              class="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700"
              transition:fade={{ duration: 100 }}
            >
              {editError}
            </div>
          {/if}

          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              onclick={saveEdit}
              disabled={saving}
              class="rounded-md bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-hover disabled:opacity-60"
            >
              {saving ? 'Saving…' : 'Save changes'}
            </button>
            <button
              type="button"
              onclick={cancelEdit}
              disabled={saving}
              class="rounded-md border border-border bg-surface-elevated px-4 py-2 text-sm font-medium text-text hover:bg-surface-sunken disabled:opacity-60"
            >
              Cancel
            </button>
          </div>
        </div>
      </section>
    {:else}
      <section class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div class="rounded-lg bg-surface-elevated p-4 ring-1 ring-border">
          <div class="text-xs uppercase tracking-wide text-text-muted">Subtotal</div>
          <div class="mt-1 text-xl font-semibold tabular-nums text-text">
            {formatMoney(bill.subtotal ?? bill.total ?? 0)}
          </div>
        </div>
        <div class="rounded-lg bg-surface-elevated p-4 ring-1 ring-border">
          <div class="text-xs uppercase tracking-wide text-text-muted">Tax &amp; fees</div>
          <div class="mt-1 text-xl font-semibold tabular-nums text-text">
            {formatMoney((bill.total ?? 0) - (bill.subtotal ?? bill.total ?? 0))}
          </div>
        </div>
        <div class="rounded-lg bg-surface-elevated p-4 ring-1 ring-border">
          <div class="text-xs uppercase tracking-wide text-text-muted">Total</div>
          <div class="mt-1 text-xl font-semibold tabular-nums text-text">
            {formatMoney(bill.total ?? 0)}
          </div>
        </div>
      </section>

      {#if (bill.items ?? []).length > 0}
        <section class="flex flex-col gap-2">
          <h2 class="text-xl font-semibold text-text">Items</h2>
          <ul class="divide-y divide-border overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
            {#each bill.items as item}
              <li class="flex items-start justify-between gap-3 px-4 py-3">
                <div class="flex flex-col">
                  <span class="font-medium text-text">{item.description || 'Item'}</span>
                  <span class="text-sm text-text-muted">
                    {(item.participantIds ?? []).join(', ') || '—'}
                  </span>
                </div>
                <span class="tabular-nums font-semibold text-text">
                  {formatMoney(item.amount ?? 0)}
                </span>
              </li>
            {/each}
          </ul>
        </section>
      {/if}

      <section class="flex flex-col gap-3">
        <h2 class="text-xl font-semibold text-text">Who owes what</h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each bill.participants ?? [] as p (p.displayName)}
            {@const raw = bill.split?.splits?.[p.displayName] ?? {}}
            {@const subT = raw.subtotal ?? 0}
            {@const taxT = raw.tax ?? 0}
            {@const totalT = raw.total ?? 0}
            {@const personItems = raw.items ?? []}
            <div class="rounded-lg bg-surface-elevated p-4 ring-1 ring-border">
              <div class="flex items-baseline justify-between gap-2">
                <h3 class="font-medium text-text">{p.displayName}</h3>
                <span class="text-lg font-semibold tabular-nums text-text">
                  {formatMoney(totalT)}
                </span>
              </div>
              {#if personItems.length > 0}
                <ul class="mt-2 flex flex-col gap-1 text-sm">
                  {#each personItems as it}
                    <li class="flex justify-between text-text-muted">
                      <span>{it.description}</span>
                      <span class="tabular-nums">{formatMoney(it.amount)}</span>
                    </li>
                  {/each}
                </ul>
              {/if}
              <div class="mt-3 border-t border-border pt-2 text-xs text-text-muted">
                <div class="flex justify-between">
                  <span>Subtotal</span>
                  <span class="tabular-nums">{formatMoney(subT)}</span>
                </div>
                <div class="flex justify-between">
                  <span>Tax</span>
                  <span class="tabular-nums">{formatMoney(taxT)}</span>
                </div>
              </div>
            </div>
          {/each}
        </div>
      </section>

      <section class="flex flex-col gap-2">
        <h2 class="text-xl font-semibold text-text">Participants</h2>
        <p class="text-text-muted">
          {(bill.participants ?? []).map((p) => p.displayName).join(', ')}
        </p>
      </section>
    {/if}
  {/if}
</main>

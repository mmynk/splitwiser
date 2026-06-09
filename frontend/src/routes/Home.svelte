<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { push, querystring } from 'svelte-spa-router';
  import { fade, fly, slide } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { Plus, ClipboardPaste, ChevronRight, Copy, Check, Receipt, HandCoins } from 'lucide-svelte';
  import { calculateSplit, createBill, listMyBills } from '$lib/api/split';
  import { getMyBalances, listGroups, settleUpWithPerson } from '$lib/api/groups';
  import type {
    BillSummary,
    GetMyBalancesResponse,
    Group,
    PersonBalance,
    PersonSplit,
  } from '$lib/api/types';
  import { currentUser } from '$lib/stores/auth';
  import { toasts } from '$lib/stores/toast';
  import { confirmAction } from '$lib/stores/confirm';
  import { ApiError } from '$lib/api/client';
  import { formatDate, formatMoney } from '$lib/util/format';
  import { dur, durFast, ease, rise } from '$lib/motion';
  import BillForm from '$lib/components/BillForm.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
  import Amount from '$lib/components/ui/Amount.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import EmptyState from '$lib/components/ui/EmptyState.svelte';
  import Alert from '$lib/components/ui/Alert.svelte';

  const PROMPT_TEMPLATE = `Convert this receipt to JSON with this exact structure:
{
  "title": "short restaurant or store name",
  "total": 00.00,
  "subtotal": 00.00,
  "participants": ["Person1", "Person2"],
  "items": [
    { "description": "Item name", "amount": 00.00 },
    { "description": "Another item", "amount": 00.00 }
  ]
}

Rules:
- total is the final amount including tax and tip
- subtotal is the pre-tax amount (omit if unknown)
- participants is the list of everyone splitting this bill
- Omit items array if you just want an equal split
- Return only valid JSON, no explanation`;

  let groups: Group[] = $state([]);
  let bills: BillSummary[] = $state([]);
  let balances = $state<GetMyBalancesResponse | null>(null);
  let billsLoading = $state(true);
  let balancesLoading = $state(true);

  let formOpen = $state(false);
  let billTitle = $state('');
  let calculating = $state(false);
  let saving = $state(false);
  let formError = $state('');
  let splitResult: { splits: Record<string, PersonSplit>; participantNames: string[] } | null =
    $state(null);

  let billForm: BillForm | null = $state(null);

  let expandedPersons = $state(new Set<string>());

  let importOpen = $state(false);
  let importText = $state('');
  let importError = $state('');
  let promptCopied = $state(false);
  let promptDetailsOpen = $state(false);
  let promptCopiedTimer: ReturnType<typeof setTimeout> | null = null;

  onDestroy(() => {
    if (promptCopiedTimer) clearTimeout(promptCopiedTimer);
  });

  onMount(async () => {
    await Promise.all([loadGroups(), loadMyBills(), loadBalances()]);
  });

  let appliedGroupQuery = '';
  // Preselect group from `#/?group=X` once groups have loaded.
  $effect(() => {
    const qs = $querystring ?? '';
    if (!qs || !billForm || groups.length === 0) return;
    const g = new URLSearchParams(qs).get('group');
    if (!g || g === appliedGroupQuery) return;
    if (groups.find((x) => x.id === g)) {
      formOpen = true;
      billForm.setGroup(g);
      appliedGroupQuery = g;
    }
  });

  async function loadGroups(): Promise<void> {
    try {
      const r = await listGroups();
      groups = r.groups ?? [];
    } catch (e) {
      console.error('Failed to load groups', e);
    }
  }

  async function loadMyBills(): Promise<void> {
    billsLoading = true;
    try {
      const r = await listMyBills();
      bills = r.bills ?? [];
    } catch (e) {
      console.error('Failed to load bills', e);
      bills = [];
    } finally {
      billsLoading = false;
    }
  }

  async function loadBalances(): Promise<void> {
    balancesLoading = true;
    try {
      balances = await getMyBalances();
    } catch (e) {
      console.error('Failed to load balances', e);
      balances = null;
    } finally {
      balancesLoading = false;
    }
  }

  function openForm(): void {
    formOpen = true;
    formError = '';
    splitResult = null;
  }

  function closeForm(): void {
    formOpen = false;
    formError = '';
    splitResult = null;
    billTitle = '';
    billForm?.reset();
  }

  function describeAmount(net: number): { dir: string; color: string } {
    if (net > 0) return { dir: 'owes you', color: 'text-success' };
    if (net < 0) return { dir: 'you owe', color: 'text-danger' };
    return { dir: 'settled', color: 'text-text-muted' };
  }

  function togglePerson(displayName: string): void {
    if (expandedPersons.has(displayName)) {
      expandedPersons.delete(displayName);
    } else {
      expandedPersons.add(displayName);
    }
    expandedPersons = new Set(expandedPersons);
  }

  async function handleSettleUp(person: PersonBalance): Promise<void> {
    if (!person.userId) return;
    const ok = await confirmAction({
      title: `Settle up with ${person.displayName}?`,
      body: 'Creates settlement records across every group you share. Can be undone per record.',
      confirmLabel: 'Settle up',
      tone: 'default',
    });
    if (!ok) return;

    // Optimistic: subtract this person's net from the totals, mark them settled,
    // and recover the previous snapshot on failure.
    const snapshot = balances ? structuredClone(balances) : null;
    if (balances) {
      const net = person.netAmount ?? 0;
      const youOwe = balances.totalYouOwe ?? 0;
      const owedYou = balances.totalOwedToYou ?? 0;
      balances = {
        ...balances,
        totalYouOwe: net < 0 ? Math.max(0, youOwe - Math.abs(net)) : youOwe,
        totalOwedToYou: net > 0 ? Math.max(0, owedYou - net) : owedYou,
        personBalances: (balances.personBalances ?? []).map((p) =>
          p.userId && p.userId === person.userId
            ? { ...p, netAmount: 0, groupBalances: (p.groupBalances ?? []).map((g) => ({ ...g, netAmount: 0 })) }
            : p,
        ),
      };
    }

    try {
      await settleUpWithPerson(person.userId);
      toasts.success('Settled.');
      await loadBalances();
    } catch (e) {
      if (snapshot) balances = snapshot;
      const msg = e instanceof ApiError ? e.message : 'Could not settle up. Try again.';
      toasts.error(msg);
    }
  }

  async function handleCalculate(): Promise<void> {
    formError = '';
    splitResult = null;
    if (!billForm) return;
    const data = billForm.getData();
    if (data.participants.length === 0) {
      formError = 'Please add at least one participant with a name.';
      return;
    }
    if (data.total <= 0) {
      formError = 'Please enter a valid total amount.';
      return;
    }
    const participantNames = data.participants.map((p) => p.displayName);
    calculating = true;
    try {
      const r = await calculateSplit({
        items: data.items,
        total: data.total,
        subtotal: data.subtotal,
        participantIds: participantNames,
      });
      splitResult = { splits: r.splits ?? {}, participantNames };
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Failed to calculate split.';
    } finally {
      calculating = false;
    }
  }

  async function handleSave(): Promise<void> {
    formError = '';
    if (!billForm) return;
    const data = billForm.getData();
    if (data.participants.length === 0) {
      formError = 'Please add at least one participant with a name.';
      return;
    }
    if (data.total <= 0) {
      formError = 'Please enter a valid total amount.';
      return;
    }
    saving = true;
    try {
      const r = await createBill({
        title: billTitle.trim(),
        items: data.items,
        total: data.total,
        subtotal: data.subtotal,
        participants: data.participants,
        payerId: data.payerId || undefined,
        groupId: data.groupId || undefined,
      });
      toasts.success('Bill saved.');
      push(`/bill/${r.billId}`);
    } catch (e) {
      formError = e instanceof ApiError ? e.message : 'Failed to save bill.';
    } finally {
      saving = false;
    }
  }

  function openImport(): void {
    importText = '';
    importError = '';
    promptCopied = false;
    importOpen = true;
  }

  function closeImport(): void {
    importOpen = false;
  }

  function handleImportFile(e: Event): void {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      importText = String(ev.target?.result ?? '');
      input.value = '';
      importError = '';
    };
    reader.onerror = () => {
      importError = 'Failed to read file.';
    };
    reader.readAsText(file);
  }

  async function copyPrompt(): Promise<void> {
    try {
      await navigator.clipboard.writeText(PROMPT_TEMPLATE);
      promptCopied = true;
      if (promptCopiedTimer) clearTimeout(promptCopiedTimer);
      promptCopiedTimer = setTimeout(() => {
        promptCopied = false;
        promptCopiedTimer = null;
      }, 2000);
    } catch {
      promptCopied = false;
    }
  }

  function confirmImport(): void {
    const raw = importText.trim();
    if (!raw) {
      importError = 'Please paste JSON or upload a file.';
      return;
    }
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch (e) {
      importError = `Invalid JSON: ${e instanceof Error ? e.message : String(e)}`;
      return;
    }
    try {
      billForm?.importJson(parsed);
      formOpen = true;
      closeImport();
    } catch (e) {
      importError = e instanceof Error ? e.message : String(e);
    }
  }

  let sortedPersonBalances = $derived(
    (balances?.personBalances ?? []).slice().sort((a, b) => (b.netAmount ?? 0) - (a.netAmount ?? 0)),
  );

  let hasBalanceData = $derived(
    !!balances &&
      ((balances.totalYouOwe ?? 0) > 0 ||
        (balances.totalOwedToYou ?? 0) > 0 ||
        (balances.personBalances ?? []).length > 0),
  );
</script>

<main class="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-6 sm:px-6">
  <!-- Balance summary -->
  {#if balancesLoading}
    <section class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <Skeleton height="h-20" rounded="card" />
      <Skeleton height="h-20" rounded="card" />
    </section>
  {:else if hasBalanceData}
    <section class="flex flex-col gap-4">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <Card padding="sm">
          <div class="text-[0.6875rem] uppercase tracking-wider text-text-muted">You owe</div>
          <div class="mt-1">
            <Amount value={balances?.totalYouOwe ?? 0} signed={false} size="xl" animate />
          </div>
        </Card>
        <Card padding="sm">
          <div class="text-[0.6875rem] uppercase tracking-wider text-text-muted">You're owed</div>
          <div class="mt-1">
            <span class="text-success">
              <Amount value={balances?.totalOwedToYou ?? 0} signed={false} size="xl" animate />
            </span>
          </div>
        </Card>
      </div>

      {#if sortedPersonBalances.length > 0}
        <ul class="divide-y divide-border overflow-hidden rounded-card border border-border bg-surface-elevated">
          {#each sortedPersonBalances as person (person.displayName)}
            {@const net = person.netAmount ?? 0}
            {@const meta = describeAmount(net)}
            {@const canSettle = Math.abs(net) > 0.005 && !!person.userId}
            {@const groupBalances = person.groupBalances ?? []}
            {@const expandable = groupBalances.length > 1}
            {@const expanded = expandedPersons.has(person.displayName)}
            <li animate:flip={{ duration: durFast }}>
              <div class="flex items-center gap-3 px-4 py-3">
                {#if expandable}
                  <button
                    type="button"
                    aria-label={expanded ? 'Collapse' : 'Expand'}
                    aria-expanded={expanded}
                    onclick={() => togglePerson(person.displayName)}
                    class="text-text-subtle transition-transform duration-150 hover:text-text-muted"
                    class:rotate-90={expanded}
                  >
                    <ChevronRight size={16} strokeWidth={1.75} />
                  </button>
                {:else}
                  <span class="inline-block w-4"></span>
                {/if}
                <div class="flex flex-1 flex-wrap items-baseline gap-x-3">
                  <span class="font-medium text-text">{person.displayName}</span>
                  <span class="text-[0.8125rem] text-text-muted">{meta.dir}</span>
                  <span class={meta.color}>
                    <Amount value={Math.abs(net)} size="lg" animate />
                  </span>
                </div>
                {#if canSettle}
                  <Button variant="secondary" size="sm" onclick={() => handleSettleUp(person)}>
                    <HandCoins size={14} strokeWidth={1.75} />
                    Settle up
                  </Button>
                {/if}
              </div>

              {#if expandable && expanded}
                <ul class="border-t border-border bg-surface-sunken/60 px-4 py-2 text-sm" transition:slide={{ duration: durFast, easing: ease }}>
                  {#each groupBalances as gb (gb.groupId)}
                    {@const gbNet = gb.netAmount ?? 0}
                    {@const gbMeta = describeAmount(gbNet)}
                    {@const settleQs = gbNet < 0 && Math.abs(gbNet) > 0 && $currentUser
                      ? `?settleFrom=${encodeURIComponent($currentUser.displayName)}&settleTo=${encodeURIComponent(person.displayName)}&amount=${Math.abs(gbNet).toFixed(2)}`
                      : ''}
                    <li>
                      <a
                        href={`#/group/${gb.groupId}${settleQs}`}
                        class="flex items-center justify-between rounded-input px-2 py-1 transition-colors hover:bg-surface-elevated"
                      >
                        <span class="text-text">{gb.groupName}</span>
                        <span class="tabular-nums {gbMeta.color}">
                          {gbMeta.dir} {formatMoney(Math.abs(gbNet))}
                        </span>
                      </a>
                    </li>
                  {/each}
                </ul>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {:else}
    <section in:fly={rise}>
      <EmptyState title="All square. Nice." hint="Add a bill and balances show up here." />
    </section>
  {/if}

  <!-- My Bills -->
  <section class="flex flex-col gap-3">
    <div class="flex items-center justify-between">
      <h2 class="font-serif text-2xl font-semibold text-text">My bills</h2>
      {#if !formOpen}
        <Button variant="primary" size="sm" onclick={openForm}>
          <Plus size={14} strokeWidth={1.75} />
          New bill
        </Button>
      {/if}
    </div>

    {#if billsLoading}
      <div class="flex flex-col gap-2">
        <Skeleton height="h-14" rounded="card" />
        <Skeleton height="h-14" rounded="card" />
        <Skeleton height="h-14" rounded="card" />
      </div>
    {:else if bills.length === 0}
      <EmptyState
        icon={Receipt}
        title="No bills yet."
        hint="The first one's always the awkward one — just add it."
      >
        <Button variant="primary" size="sm" onclick={openForm}>
          <Plus size={14} strokeWidth={1.75} />
          Add the first one
        </Button>
      </EmptyState>
    {:else}
      <div class="overflow-hidden rounded-card border border-border bg-surface-elevated">
        <div class="hidden border-b border-border bg-surface-sunken px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-wider text-text-muted sm:grid sm:grid-cols-[1fr_1fr_auto_auto_auto] sm:gap-4">
          <span>Bill</span>
          <span>Group</span>
          <span class="text-right">Total</span>
          <span class="text-right">Participants</span>
          <span class="text-right">Date</span>
        </div>
        <ul class="divide-y divide-border">
          {#each bills as bill (bill.billId)}
            <li>
              <a
                href={`#/bill/${bill.billId}`}
                class="grid grid-cols-1 gap-1 px-4 py-3 transition-colors hover:bg-surface-sunken sm:grid-cols-[1fr_1fr_auto_auto_auto] sm:items-center sm:gap-4"
              >
                <span class="font-medium text-text">{bill.title || 'Untitled'}</span>
                <span class="text-[0.8125rem] text-text-muted">
                  {#if bill.groupName}
                    {bill.groupName}
                  {:else}
                    <span class="text-text-subtle">—</span>
                  {/if}
                </span>
                <span class="text-[0.875rem] tabular-nums sm:text-right">{formatMoney(bill.total)}</span>
                <span class="text-[0.875rem] tabular-nums text-text-muted sm:text-right">
                  {bill.participantCount ?? 0}
                </span>
                <span class="text-[0.8125rem] text-text-muted sm:text-right">{formatDate(bill.createdAt)}</span>
              </a>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </section>

  <!-- New / Edit Bill Form -->
  {#if formOpen}
    <section
      class="rounded-card border border-border bg-surface-elevated p-5"
      transition:slide={{ duration: dur, easing: ease }}
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-serif text-xl font-semibold text-text">Bill details</h2>
        <div class="flex items-center gap-2">
          <Button variant="secondary" size="sm" onclick={openImport}>
            <ClipboardPaste size={14} strokeWidth={1.75} />
            Import JSON
          </Button>
          <Button variant="ghost" size="sm" onclick={closeForm}>Cancel</Button>
        </div>
      </div>

      <div class="mt-4 flex flex-col gap-6">
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-text">
            Bill name <span class="text-text-subtle">(optional)</span>
          </span>
          <input
            type="text"
            bind:value={billTitle}
            placeholder="e.g. Team lunch, grocery run"
            class="rounded-input border border-border bg-surface-elevated px-3 py-2 outline-none transition-colors focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
          />
          <small class="text-text-muted">Leave empty for an auto-generated name.</small>
        </label>

        <BillForm bind:this={billForm} {groups} currentUser={$currentUser} />

        {#if formError}
          <Alert>{formError}</Alert>
        {/if}

        <div class="flex flex-wrap gap-2">
          <Button onclick={handleCalculate} loading={calculating} disabled={saving}>
            {calculating ? 'Calculating…' : 'Calculate split'}
          </Button>
          <Button variant="secondary" onclick={handleSave} loading={saving} disabled={calculating}>
            {saving ? 'Saving…' : 'Save & share'}
          </Button>
        </div>
      </div>
    </section>
  {/if}

  <!-- Split Results -->
  {#if splitResult}
    <section class="flex flex-col gap-3" transition:fade={{ duration: durFast, easing: ease }}>
      <h2 class="font-serif text-xl font-semibold text-text">Split results</h2>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {#each splitResult.participantNames as name}
          {@const raw = splitResult.splits[name] ?? {}}
          {@const subT = raw.subtotal ?? 0}
          {@const taxT = raw.tax ?? 0}
          {@const totalT = raw.total ?? 0}
          {@const personItems = raw.items ?? []}
          <Card padding="sm">
            <div class="flex items-baseline justify-between">
              <h3 class="font-medium text-text">{name}</h3>
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
              <div class="flex justify-between"><span>Subtotal</span><span class="tabular-nums">{formatMoney(subT)}</span></div>
              <div class="flex justify-between"><span>Tax</span><span class="tabular-nums">{formatMoney(taxT)}</span></div>
            </div>
          </Card>
        {/each}
      </div>
    </section>
  {/if}
</main>

<!-- Import JSON modal -->
<Modal open={importOpen} title="Import from JSON" onClose={closeImport} maxWidth="max-w-2xl">
  <div class="flex flex-col gap-4">
    <details class="rounded-input border border-border bg-surface-sunken px-3 py-2" bind:open={promptDetailsOpen}>
      <summary class="cursor-pointer text-sm font-medium text-text">
        Get AI prompt template
      </summary>
      <div class="mt-2 flex flex-col gap-2">
        <p class="text-xs text-text-muted">
          Copy this prompt and paste it into Claude or ChatGPT along with your receipt photo or text.
        </p>
        <pre class="max-h-56 overflow-auto whitespace-pre-wrap rounded-input border border-border bg-surface-elevated p-2 text-xs text-text">{PROMPT_TEMPLATE}</pre>
        <Button variant="secondary" size="sm" onclick={copyPrompt}>
          {#if promptCopied}<Check size={12} strokeWidth={1.75} /> Copied{:else}<Copy size={12} strokeWidth={1.75} /> Copy prompt{/if}
        </Button>
      </div>
    </details>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Upload .json file</span>
      <input
        type="file"
        accept=".json,application/json"
        onchange={handleImportFile}
        class="text-sm"
      />
    </label>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-text">Or paste JSON directly</span>
      <textarea
        rows="8"
        bind:value={importText}
        placeholder={'{"title": "Dinner", "total": 45.50, "participants": ["Alice", "Bob"]}'}
        class="rounded-input border border-border bg-surface-elevated px-3 py-2 font-mono text-xs outline-none transition-colors focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
      ></textarea>
    </label>

    {#if importError}
      <Alert>{importError}</Alert>
    {/if}
  </div>

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button variant="ghost" size="sm" onclick={closeImport}>Cancel</Button>
      <Button variant="primary" size="sm" onclick={confirmImport}>Import</Button>
    </div>
  {/snippet}
</Modal>

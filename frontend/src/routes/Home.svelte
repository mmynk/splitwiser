<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { push, querystring } from 'svelte-spa-router';
  import { fade, slide } from 'svelte/transition';
  import { Plus, ClipboardPaste, ChevronRight, Copy, Check, Receipt } from 'lucide-svelte';
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
  import { ApiError } from '$lib/api/client';
  import { formatDate, formatMoney } from '$lib/util/format';
  import BillForm from '$lib/components/BillForm.svelte';
  import Modal from '$lib/components/Modal.svelte';

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
    if (net > 0) return { dir: 'owes you', color: 'text-emerald-700' };
    if (net < 0) return { dir: 'you owe', color: 'text-rose-700' };
    return { dir: 'settled', color: 'text-slate-500' };
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
    const ok = confirm(
      `Settle up all debts with ${person.displayName}? This will create settlement records across every group you share with them.`,
    );
    if (!ok) return;
    try {
      await settleUpWithPerson(person.userId);
      toasts.success(`Settled up with ${person.displayName}.`);
      await loadBalances();
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : 'Failed to settle up.';
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

<main class="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-6">
  <!-- Balance summary -->
  {#if balancesLoading}
    <section class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      {#each [1, 2] as _}
        <div class="h-20 animate-pulse rounded-lg bg-white ring-1 ring-slate-200"></div>
      {/each}
    </section>
  {:else if hasBalanceData}
    <section class="flex flex-col gap-4">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="rounded-lg bg-white p-4 ring-1 ring-slate-200">
          <div class="text-xs uppercase tracking-wide text-slate-500">You owe</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums text-rose-700">
            {formatMoney(balances?.totalYouOwe)}
          </div>
        </div>
        <div class="rounded-lg bg-white p-4 ring-1 ring-slate-200">
          <div class="text-xs uppercase tracking-wide text-slate-500">You're owed</div>
          <div class="mt-1 text-2xl font-semibold tabular-nums text-emerald-700">
            {formatMoney(balances?.totalOwedToYou)}
          </div>
        </div>
      </div>

      {#if sortedPersonBalances.length > 0}
        <ul class="divide-y divide-slate-200 overflow-hidden rounded-lg bg-white ring-1 ring-slate-200">
          {#each sortedPersonBalances as person (person.displayName)}
            {@const net = person.netAmount ?? 0}
            {@const meta = describeAmount(net)}
            {@const canSettle = Math.abs(net) > 0.005 && !!person.userId}
            {@const groupBalances = person.groupBalances ?? []}
            {@const expandable = groupBalances.length > 1}
            {@const expanded = expandedPersons.has(person.displayName)}
            <li>
              <div class="flex items-center gap-3 px-4 py-3">
                {#if expandable}
                  <button
                    type="button"
                    aria-label={expanded ? 'Collapse' : 'Expand'}
                    aria-expanded={expanded}
                    onclick={() => togglePerson(person.displayName)}
                    class="text-slate-400 transition-transform hover:text-slate-600"
                    class:rotate-90={expanded}
                  >
                    <ChevronRight size={16} />
                  </button>
                {:else}
                  <span class="inline-block w-4"></span>
                {/if}
                <div class="flex flex-1 flex-wrap items-baseline gap-x-3">
                  <span class="font-medium text-slate-900">{person.displayName}</span>
                  <span class="text-sm text-slate-500">{meta.dir}</span>
                  <span class="tabular-nums font-semibold {meta.color}">
                    {formatMoney(Math.abs(net))}
                  </span>
                </div>
                {#if canSettle}
                  <button
                    type="button"
                    onclick={() => handleSettleUp(person)}
                    class="rounded-md border border-emerald-300 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-800 hover:bg-emerald-100"
                  >
                    Settle up
                  </button>
                {/if}
              </div>

              {#if expandable && expanded}
                <ul class="border-t border-slate-100 bg-slate-50/60 px-4 py-2 text-sm" transition:slide={{ duration: 150 }}>
                  {#each groupBalances as gb (gb.groupId)}
                    {@const gbNet = gb.netAmount ?? 0}
                    {@const gbMeta = describeAmount(gbNet)}
                    {@const settleQs = gbNet < 0 && Math.abs(gbNet) > 0 && $currentUser
                      ? `?settleFrom=${encodeURIComponent($currentUser.displayName)}&settleTo=${encodeURIComponent(person.displayName)}&amount=${Math.abs(gbNet).toFixed(2)}`
                      : ''}
                    <li>
                      <a
                        href={`#/group/${gb.groupId}${settleQs}`}
                        class="flex items-center justify-between rounded px-2 py-1 hover:bg-white"
                      >
                        <span class="text-slate-700">{gb.groupName}</span>
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
  {/if}

  <!-- My Bills -->
  <section class="flex flex-col gap-3">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-semibold text-slate-900">My Bills</h2>
      {#if !formOpen}
        <button
          type="button"
          onclick={openForm}
          class="inline-flex items-center gap-1 rounded-md bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus size={14} /> New Bill
        </button>
      {/if}
    </div>

    {#if billsLoading}
      <div class="overflow-hidden rounded-lg bg-white ring-1 ring-slate-200">
        {#each [1, 2, 3] as _}
          <div class="h-12 animate-pulse border-b border-slate-100 last:border-b-0"></div>
        {/each}
      </div>
    {:else if bills.length === 0}
      <div class="flex flex-col items-center gap-2 rounded-lg bg-white px-6 py-10 text-center ring-1 ring-slate-200">
        <Receipt size={28} class="text-slate-400" />
        <p class="text-slate-600">No bills yet.</p>
        <button
          type="button"
          onclick={openForm}
          class="mt-1 inline-flex items-center gap-1 rounded-md border border-brand-300 bg-brand-50 px-3 py-1.5 text-sm font-medium text-brand-700 hover:bg-brand-100"
        >
          <Plus size={14} /> Create your first bill
        </button>
      </div>
    {:else}
      <div class="overflow-hidden rounded-lg bg-white ring-1 ring-slate-200">
        <div class="hidden border-b border-slate-200 bg-slate-50 px-4 py-2 text-xs font-medium uppercase tracking-wide text-slate-500 sm:grid sm:grid-cols-[1fr_1fr_auto_auto_auto] sm:gap-4">
          <span>Bill</span>
          <span>Group</span>
          <span class="text-right">Total</span>
          <span class="text-right">Participants</span>
          <span class="text-right">Date</span>
        </div>
        <ul class="divide-y divide-slate-100">
          {#each bills as bill (bill.billId)}
            <li>
              <a
                href={`#/bill/${bill.billId}`}
                class="grid grid-cols-1 gap-1 px-4 py-3 hover:bg-slate-50 sm:grid-cols-[1fr_1fr_auto_auto_auto] sm:items-center sm:gap-4"
              >
                <span class="font-medium text-slate-900">{bill.title || 'Untitled'}</span>
                <span class="text-sm text-slate-600">
                  {#if bill.groupName}
                    {bill.groupName}
                  {:else}
                    <span class="text-slate-400">—</span>
                  {/if}
                </span>
                <span class="text-sm tabular-nums sm:text-right">{formatMoney(bill.total)}</span>
                <span class="text-sm tabular-nums text-slate-600 sm:text-right">
                  {bill.participantCount ?? 0}
                </span>
                <span class="text-sm text-slate-500 sm:text-right">{formatDate(bill.createdAt)}</span>
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
      class="rounded-lg bg-white p-5 ring-1 ring-slate-200"
      transition:slide={{ duration: 200 }}
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-xl font-semibold text-slate-900">Bill Details</h2>
        <div class="flex items-center gap-2">
          <button
            type="button"
            onclick={openImport}
            class="inline-flex items-center gap-1 rounded-md border border-slate-300 px-2.5 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
          >
            <ClipboardPaste size={14} /> Import JSON
          </button>
          <button
            type="button"
            onclick={closeForm}
            class="rounded-md px-2.5 py-1.5 text-sm font-medium text-slate-500 hover:bg-slate-100"
          >
            Cancel
          </button>
        </div>
      </div>

      <div class="mt-4 flex flex-col gap-6">
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-slate-700">
            Bill name <span class="text-slate-400">(optional)</span>
          </span>
          <input
            type="text"
            bind:value={billTitle}
            placeholder="e.g. Team Lunch, Grocery Run"
            class="rounded-md border border-slate-300 px-3 py-2 outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
          />
          <small class="text-slate-500">Leave empty for an auto-generated name.</small>
        </label>

        <BillForm bind:this={billForm} {groups} currentUser={$currentUser} />

        {#if formError}
          <div role="alert" class="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700" transition:fade={{ duration: 100 }}>
            {formError}
          </div>
        {/if}

        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            onclick={handleCalculate}
            disabled={calculating || saving}
            class="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-60"
          >
            {calculating ? 'Calculating…' : 'Calculate Split'}
          </button>
          <button
            type="button"
            onclick={handleSave}
            disabled={calculating || saving}
            class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-60"
          >
            {saving ? 'Saving…' : 'Save & Share'}
          </button>
        </div>
      </div>
    </section>
  {/if}

  <!-- Split Results -->
  {#if splitResult}
    <section class="flex flex-col gap-3" transition:fade={{ duration: 120 }}>
      <h2 class="text-xl font-semibold text-slate-900">Split Results</h2>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {#each splitResult.participantNames as name}
          {@const raw = splitResult.splits[name] ?? {}}
          {@const subT = raw.subtotal ?? 0}
          {@const taxT = raw.tax ?? 0}
          {@const totalT = raw.total ?? 0}
          {@const personItems = raw.items ?? []}
          <div class="rounded-lg bg-white p-4 ring-1 ring-slate-200">
            <div class="flex items-baseline justify-between">
              <h3 class="font-medium text-slate-900">{name}</h3>
              <span class="text-lg font-semibold tabular-nums text-slate-900">
                {formatMoney(totalT)}
              </span>
            </div>
            {#if personItems.length > 0}
              <ul class="mt-2 flex flex-col gap-1 text-sm">
                {#each personItems as it}
                  <li class="flex justify-between text-slate-600">
                    <span>{it.description}</span>
                    <span class="tabular-nums">{formatMoney(it.amount)}</span>
                  </li>
                {/each}
              </ul>
            {/if}
            <div class="mt-3 border-t border-slate-100 pt-2 text-xs text-slate-500">
              <div class="flex justify-between"><span>Subtotal</span><span class="tabular-nums">{formatMoney(subT)}</span></div>
              <div class="flex justify-between"><span>Tax</span><span class="tabular-nums">{formatMoney(taxT)}</span></div>
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</main>

<!-- Import JSON modal -->
<Modal open={importOpen} title="Import from JSON" onClose={closeImport} maxWidth="max-w-2xl">
  <div class="flex flex-col gap-4">
    <details class="rounded-md border border-slate-200 bg-slate-50 px-3 py-2" bind:open={promptDetailsOpen}>
      <summary class="cursor-pointer text-sm font-medium text-slate-700">
        Get AI prompt template
      </summary>
      <div class="mt-2 flex flex-col gap-2">
        <p class="text-xs text-slate-500">
          Copy this prompt and paste it into Claude or ChatGPT along with your receipt photo or text:
        </p>
        <pre class="max-h-56 overflow-auto whitespace-pre-wrap rounded bg-white p-2 text-xs text-slate-700 ring-1 ring-slate-200">{PROMPT_TEMPLATE}</pre>
        <button
          type="button"
          onclick={copyPrompt}
          class="inline-flex w-fit items-center gap-1 rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-white"
        >
          {#if promptCopied}<Check size={12} /> Copied!{:else}<Copy size={12} /> Copy prompt{/if}
        </button>
      </div>
    </details>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-slate-700">Upload .json file</span>
      <input
        type="file"
        accept=".json,application/json"
        onchange={handleImportFile}
        class="text-sm"
      />
    </label>

    <label class="flex flex-col gap-1 text-sm">
      <span class="font-medium text-slate-700">Or paste JSON directly</span>
      <textarea
        rows="8"
        bind:value={importText}
        placeholder={'{"title": "Dinner", "total": 45.50, "participants": ["Alice", "Bob"]}'}
        class="rounded-md border border-slate-300 px-3 py-2 font-mono text-xs outline-none focus:border-brand-500 focus:ring-2 focus:ring-brand-100"
      ></textarea>
    </label>

    {#if importError}
      <div role="alert" class="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">
        {importError}
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <button
        type="button"
        onclick={closeImport}
        class="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50"
      >
        Cancel
      </button>
      <button
        type="button"
        onclick={confirmImport}
        class="rounded-md bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700"
      >
        Import
      </button>
    </div>
  {/snippet}
</Modal>

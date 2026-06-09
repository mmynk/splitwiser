<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { slide } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { link } from 'svelte-spa-router';
  import {
    Plus,
    Trash2,
    Pencil,
    ChevronDown,
    ChevronUp,
    Users,
    BadgeCheck,
  } from 'lucide-svelte';
  import { createGroup, deleteGroup, listGroups, updateGroup } from '$lib/api/groups';
  import { listBillsByGroup } from '$lib/api/split';
  import type { BillSummary, Group, GroupMember } from '$lib/api/types';
  import { currentUser } from '$lib/stores/auth';
  import { toasts } from '$lib/stores/toast';
  import { confirmAction } from '$lib/stores/confirm';
  import { apiMessage } from '$lib/api/client';
  import { formatDate, formatMoney } from '$lib/util/format';
  import { dur, durFast, ease } from '$lib/motion';
  import UserSearch, { type UserPick } from '$lib/components/UserSearch.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import IconButton from '$lib/components/ui/IconButton.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import EmptyState from '$lib/components/ui/EmptyState.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Alert from '$lib/components/ui/Alert.svelte';

  interface MemberRow {
    id: string;
    displayName: string;
    userId?: string;
    isCreator?: boolean;
  }

  type FormMode =
    | { kind: 'closed' }
    | { kind: 'create' }
    | { kind: 'edit'; id: string };

  let nextLocalId = 1;
  function nextId(): string {
    return `m${nextLocalId++}`;
  }

  let groups: Group[] = $state([]);
  let groupsLoading = $state(true);

  let mode: FormMode = $state({ kind: 'closed' });
  let groupName = $state('');
  let members: MemberRow[] = $state([]);
  let formError = $state('');
  let saving = $state(false);

  let billsState = $state<Record<string, { open: boolean; loading: boolean; bills: BillSummary[] }>>(
    {},
  );

  onMount(() => {
    loadGroups();
  });

  async function loadGroups(): Promise<void> {
    groupsLoading = true;
    try {
      const r = await listGroups();
      groups = r.groups ?? [];
      pruneBillsState();
    } catch (e) {
      toasts.error(apiMessage(e, 'Failed to load groups.'));
      groups = [];
    } finally {
      groupsLoading = false;
    }
  }

  function pruneBillsState(): void {
    const live = new Set(groups.map((g) => g.id));
    const next: typeof billsState = {};
    for (const [id, v] of Object.entries(billsState)) {
      if (live.has(id)) next[id] = v;
    }
    billsState = next;
  }

  function buildCreatorRow(): MemberRow {
    const u = $currentUser;
    return {
      id: nextId(),
      displayName: u?.displayName ?? u?.email ?? '',
      userId: u?.id,
      isCreator: true,
    };
  }

  function openCreate(): void {
    mode = { kind: 'create' };
    groupName = '';
    formError = '';
    members = [buildCreatorRow(), { id: nextId(), displayName: '' }];
  }

  function openEdit(group: Group): void {
    mode = { kind: 'edit', id: group.id };
    groupName = group.name;
    formError = '';
    members = (group.members ?? []).map((m) => ({
      id: nextId(),
      displayName: m.displayName ?? '',
      userId: m.userId,
    }));
    if (members.length === 0) members = [{ id: nextId(), displayName: '' }];
  }

  function closeForm(): void {
    mode = { kind: 'closed' };
    groupName = '';
    members = [];
    formError = '';
  }

  async function addMemberRow(): Promise<void> {
    members = [...members, { id: nextId(), displayName: '' }];
    await tick();
  }

  function removeMemberRow(id: string): void {
    const row = members.find((m) => m.id === id);
    if (!row || row.isCreator) return;
    if (members.filter((m) => !m.isCreator).length <= 1) return;
    members = members.filter((m) => m.id !== id);
  }

  function setMemberName(id: string, value: string): void {
    members = members.map((m) =>
      m.id === id && !m.isCreator ? { ...m, displayName: value, userId: undefined } : m,
    );
  }

  function setMemberUser(id: string, pick: UserPick): void {
    members = members.map((m) =>
      m.id === id && !m.isCreator
        ? { ...m, displayName: pick.displayName, userId: pick.userId }
        : m,
    );
  }

  function serializeMembers(): GroupMember[] {
    return members
      .filter((m) => m.displayName.trim())
      .map((m) => ({
        displayName: m.displayName.trim(),
        ...(m.userId ? { userId: m.userId } : {}),
      }));
  }

  async function submitForm(e: SubmitEvent): Promise<void> {
    e.preventDefault();
    formError = '';

    const name = groupName.trim();
    if (!name) {
      formError = 'Please enter a group name.';
      return;
    }

    const serialized = serializeMembers();
    if (serialized.length === 0) {
      formError = 'Please add at least one member.';
      return;
    }

    const names = serialized.map((m) => m.displayName.toLowerCase());
    if (names.length !== new Set(names).size) {
      formError = 'Duplicate member: remove repeated names.';
      return;
    }

    saving = true;
    try {
      if (mode.kind === 'create') {
        await createGroup({ name, members: serialized });
        toasts.success('Group created.');
      } else if (mode.kind === 'edit') {
        await updateGroup({ groupId: mode.id, name, members: serialized });
        toasts.success('Group updated.');
      }
      closeForm();
      await loadGroups();
    } catch (err) {
      formError = apiMessage(err, 'Failed to save group.');
    } finally {
      saving = false;
    }
  }

  async function handleDeleteGroup(group: Group): Promise<void> {
    const ok = await confirmAction({
      title: `Delete "${group.name}"?`,
      body: "Members, bills, and settlements stay; the group itself goes. Can't be undone.",
      confirmLabel: 'Delete group',
      tone: 'danger',
    });
    if (!ok) return;
    try {
      await deleteGroup(group.id);
      toasts.success('Group deleted.');
      if (mode.kind === 'edit' && mode.id === group.id) closeForm();
      const { [group.id]: _, ...rest } = billsState;
      billsState = rest;
      await loadGroups();
    } catch (err) {
      toasts.error(apiMessage(err, 'Failed to delete group.'));
    }
  }

  async function toggleBills(groupId: string): Promise<void> {
    const existing = billsState[groupId];
    if (existing?.open) {
      billsState = { ...billsState, [groupId]: { ...existing, open: false } };
      return;
    }
    billsState = {
      ...billsState,
      [groupId]: { open: true, loading: true, bills: existing?.bills ?? [] },
    };
    try {
      const r = await listBillsByGroup(groupId);
      billsState = {
        ...billsState,
        [groupId]: { open: true, loading: false, bills: r.bills ?? [] },
      };
    } catch (err) {
      billsState = {
        ...billsState,
        [groupId]: { open: true, loading: false, bills: [] },
      };
      toasts.error(apiMessage(err, 'Failed to load bills.'));
    }
  }

  let removableCount = $derived(members.filter((m) => !m.isCreator).length);
</script>

<main class="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-6 sm:px-6">
  <section class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="font-serif text-2xl font-semibold text-text">Groups</h1>
      <p class="text-[0.875rem] text-text-muted">Reusable lists of people you split bills with.</p>
    </div>
    {#if mode.kind === 'closed'}
      <Button variant="primary" size="sm" onclick={openCreate}>
        <Plus size={14} strokeWidth={1.75} /> New group
      </Button>
    {/if}
  </section>

  {#if mode.kind !== 'closed'}
    <section
      class="rounded-card border border-border bg-surface-elevated p-5"
      transition:slide={{ duration: dur, easing: ease }}
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-serif text-lg font-semibold text-text">
          {mode.kind === 'create' ? 'New group' : 'Edit group'}
        </h2>
        <Button variant="ghost" size="sm" onclick={closeForm}>Cancel</Button>
      </div>

      <form class="mt-4 flex flex-col gap-5" onsubmit={submitForm}>
        <label class="flex flex-col gap-1 text-sm">
          <span class="font-medium text-text">Group name</span>
          <input
            type="text"
            bind:value={groupName}
            placeholder="e.g. Roommates, Work Lunch"
            required
            class="rounded-md border border-border px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
          />
        </label>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-text">Members</span>
            <span class="text-xs text-text-subtle">Press Enter to add a new member</span>
          </div>

          <ul class="flex flex-col gap-2">
            {#each members as member (member.id)}
              <li class="flex items-center gap-2">
                {#if member.isCreator}
                  <div
                    class="flex flex-1 items-center gap-2 rounded-md border border-border bg-surface-sunken px-3 py-2 text-sm"
                  >
                    <span class="flex-1 truncate text-text">{member.displayName || '(you)'}</span>
                    <span
                      class="inline-flex items-center gap-1 rounded bg-primary-soft px-1.5 py-0.5 text-[0.7rem] font-medium text-primary"
                      title="You (group creator)"
                    >
                      You
                    </span>
                  </div>
                  <button
                    type="button"
                    disabled
                    class="rounded-md p-2 text-text-subtle"
                    aria-label="Remove member"
                  >
                    <Trash2 size={14} />
                  </button>
                {:else}
                  <div class="relative flex-1">
                    <UserSearch
                      value={member.displayName}
                      placeholder="Member name"
                      onInput={(v) => setMemberName(member.id, v)}
                      onSelect={(u) => setMemberUser(member.id, u)}
                      onEnter={() => addMemberRow()}
                      inputClass="w-full rounded-md border border-border bg-surface-elevated pl-3 pr-9 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
                    />
                    {#if member.userId}
                      <span
                        class="pointer-events-none absolute inset-y-0 right-2 flex items-center text-primary"
                        title="Linked to registered user"
                      >
                        <BadgeCheck size={16} />
                      </span>
                    {/if}
                  </div>
                  <button
                    type="button"
                    aria-label="Remove member"
                    onclick={() => removeMemberRow(member.id)}
                    disabled={removableCount <= 1}
                    class="rounded-md p-2 text-text-muted transition hover:bg-surface-sunken hover:text-danger disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-text-muted"
                  >
                    <Trash2 size={14} />
                  </button>
                {/if}
              </li>
            {/each}
          </ul>

          <button
            type="button"
            onclick={addMemberRow}
            class="inline-flex w-fit items-center gap-1 rounded-md border border-border bg-surface-elevated px-2.5 py-1.5 text-xs font-medium text-text hover:bg-surface-sunken"
          >
            <Plus size={12} /> Add member
          </button>
        </div>

        {#if formError}
          <Alert>{formError}</Alert>
        {/if}

        <div class="flex flex-wrap gap-2">
          <Button type="submit" loading={saving}>
            {saving
              ? mode.kind === 'create'
                ? 'Creating…'
                : 'Saving…'
              : mode.kind === 'create'
                ? 'Create group'
                : 'Save changes'}
          </Button>
          <Button variant="secondary" onclick={closeForm} disabled={saving}>Cancel</Button>
        </div>
      </form>
    </section>
  {/if}

  <section class="flex flex-col gap-3">
    {#if groupsLoading}
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {#each Array(4) as _, i (i)}
          <Skeleton height="h-28" rounded="card" />
        {/each}
      </div>
    {:else if groups.length === 0}
      <EmptyState
        icon={Users}
        title="No groups yet."
        hint="Make one for the people you split with most."
      >
        {#if mode.kind === 'closed'}
          <Button variant="primary" size="sm" onclick={openCreate}>
            <Plus size={14} strokeWidth={1.75} /> New group
          </Button>
        {/if}
      </EmptyState>
    {:else}
      <ul class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {#each groups as group (group.id)}
          {@const state = billsState[group.id]}
          {@const isEditing = mode.kind === 'edit' && mode.id === group.id}
          <li
            animate:flip={{ duration: durFast }}
            class="flex flex-col gap-3 rounded-card border bg-surface-elevated p-4 transition-colors"
            class:border-primary={isEditing}
            class:border-border={!isEditing}
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <a
                  use:link
                  href={`/group/${group.id}`}
                  class="display-wonk block truncate text-lg font-semibold text-text hover:text-primary"
                >
                  {group.name}
                </a>
              </div>
              <div class="flex flex-shrink-0 items-center gap-1">
                <IconButton ariaLabel="Edit group" title="Edit" size="sm" onclick={() => openEdit(group)}>
                  <Pencil size={14} strokeWidth={1.75} />
                </IconButton>
                <IconButton ariaLabel="Delete group" title="Delete" size="sm" variant="danger" onclick={() => handleDeleteGroup(group)}>
                  <Trash2 size={14} strokeWidth={1.75} />
                </IconButton>
              </div>
            </div>

            <div class="flex flex-wrap gap-1.5">
              {#each group.members ?? [] as m, i (m.userId ?? `name:${m.displayName}:${i}`)}
                <Badge tone={m.userId ? 'primary' : 'neutral'}>
                  {#if m.userId}
                    <BadgeCheck size={11} strokeWidth={1.75} />
                  {/if}
                  {m.displayName}
                </Badge>
              {/each}
              {#if (group.members ?? []).length === 0}
                <span class="italic text-text-subtle text-xs">No members</span>
              {/if}
            </div>

            <div class="border-t border-border pt-2">
              <button
                type="button"
                onclick={() => toggleBills(group.id)}
                class="inline-flex items-center gap-1 text-xs font-medium text-text-muted hover:text-text"
              >
                Bills
                {#if state?.open}<ChevronUp size={12} />{:else}<ChevronDown size={12} />{/if}
              </button>

              {#if state?.open}
                <div class="mt-2" transition:slide={{ duration: 150 }}>
                  {#if state.loading}
                    <p class="text-xs text-text-subtle">Loading…</p>
                  {:else if state.bills.length === 0}
                    <p class="text-xs italic text-text-subtle">No bills yet for this group.</p>
                  {:else}
                    <ul class="divide-y divide-border rounded-md ring-1 ring-border">
                      {#each state.bills as bill (bill.billId)}
                        <li>
                          <a
                            use:link
                            href={`/bill/${bill.billId}`}
                            class="grid grid-cols-[1fr_auto_auto] items-center gap-3 px-3 py-2 text-xs hover:bg-surface-sunken"
                          >
                            <span class="truncate font-medium text-text">{bill.title || 'Untitled'}</span>
                            <span class="tabular-nums text-text">{formatMoney(bill.total)}</span>
                            <span class="text-text-muted">{formatDate(bill.createdAt)}</span>
                          </a>
                        </li>
                      {/each}
                    </ul>
                  {/if}
                </div>
              {/if}
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</main>

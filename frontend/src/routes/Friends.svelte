<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade, slide } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { Check, X, UserPlus, UserMinus, Search, UsersRound } from 'lucide-svelte';
  import {
    listFriends,
    listFriendRequests,
    respondToFriendRequest,
    removeFriend,
    sendFriendRequest,
  } from '$lib/api/friends';
  import { searchUsers } from '$lib/api/split';
  import { ApiError } from '$lib/api/client';
  import { toasts } from '$lib/stores/toast';
  import { confirmAction } from '$lib/stores/confirm';
  import { dur, durFast, ease } from '$lib/motion';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import EmptyState from '$lib/components/ui/EmptyState.svelte';
  import type {
    Friend,
    FriendRequest,
    UserSearchResult,
  } from '$lib/api/types';

  let friends = $state<Friend[]>([]);
  let incoming = $state<FriendRequest[]>([]);
  let outgoing = $state<FriendRequest[]>([]);
  let loading = $state(true);

  let searchQuery = $state('');
  let searchResults = $state<UserSearchResult[]>([]);
  let searched = $state(false);
  let searching = $state(false);
  let pendingActionIds = $state(new Set<string>());

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let latestSearchId = 0;

  onMount(() => {
    loadAll();
  });

  onDestroy(() => {
    if (debounceTimer) clearTimeout(debounceTimer);
  });

  async function loadAll(): Promise<void> {
    loading = true;
    try {
      const [fr, inR, outR] = await Promise.all([
        listFriends(),
        listFriendRequests(true),
        listFriendRequests(false),
      ]);
      friends = fr.friends ?? [];
      incoming = inR.requests ?? [];
      outgoing = outR.requests ?? [];
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : 'Failed to load friends.';
      toasts.error(msg);
    } finally {
      loading = false;
    }
  }

  async function refreshOutgoing(): Promise<void> {
    try {
      const r = await listFriendRequests(false);
      outgoing = r.requests ?? [];
    } catch {
      // outgoing only affects status badges in search; failure is non-fatal.
    }
  }

  function classifyUser(u: UserSearchResult): 'friend' | 'pending' | 'stranger' {
    if (friends.some((f) => f.userId === u.userId)) return 'friend';
    if (outgoing.some((r) => r.addresseeId === u.userId && r.status === 'pending')) {
      return 'pending';
    }
    return 'stranger';
  }

  function handleSearchInput(): void {
    if (debounceTimer) clearTimeout(debounceTimer);
    const q = searchQuery.trim();
    if (q.length < 2) {
      searchResults = [];
      searched = false;
      latestSearchId++;
      return;
    }
    debounceTimer = setTimeout(() => runSearch(q), 300);
  }

  async function runSearch(query: string): Promise<void> {
    const id = ++latestSearchId;
    searching = true;
    try {
      const r = await searchUsers(query);
      if (id !== latestSearchId) return;
      searchResults = r.users ?? [];
      searched = true;
    } catch {
      if (id !== latestSearchId) return;
      searchResults = [];
      searched = true;
    } finally {
      if (id === latestSearchId) searching = false;
    }
  }

  async function withPending(
    id: string,
    action: () => Promise<unknown>,
    successMsg: string,
    failPrefix: string,
    after?: () => Promise<void> | void,
  ): Promise<void> {
    pendingActionIds.add(id);
    try {
      await action();
      toasts.success(successMsg);
      await after?.();
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : `${failPrefix}.`;
      toasts.error(msg);
    } finally {
      pendingActionIds.delete(id);
    }
  }

  function accept(req: FriendRequest): Promise<void> {
    return withPending(
      req.id,
      () => respondToFriendRequest(req.id, true),
      `You and ${req.requesterDisplayName} are now friends.`,
      'Failed to accept request',
      loadAll,
    );
  }

  function decline(req: FriendRequest): Promise<void> {
    return withPending(
      req.id,
      () => respondToFriendRequest(req.id, false),
      'Request declined.',
      'Failed to decline request',
      loadAll,
    );
  }

  async function unfriend(f: Friend): Promise<void> {
    const ok = await confirmAction({
      title: `Remove ${f.displayName}?`,
      body: 'They stay registered; just unlinked from you.',
      confirmLabel: 'Unfriend',
      tone: 'danger',
    });
    if (!ok) return;
    await withPending(
      f.userId,
      () => removeFriend(f.userId),
      `Removed ${f.displayName}.`,
      'Failed to remove friend',
      async () => {
        await loadAll();
        const q = searchQuery.trim();
        if (q.length >= 2) runSearch(q);
      },
    );
  }

  function addFriend(u: UserSearchResult): Promise<void> {
    return withPending(
      u.userId,
      () => sendFriendRequest(u.userId),
      `Request sent to ${u.displayName}.`,
      'Failed to send request',
      refreshOutgoing,
    );
  }
</script>

<main class="mx-auto flex max-w-3xl flex-col gap-8 px-4 py-6 sm:px-6">
  <header class="flex flex-col gap-1">
    <h1 class="font-serif text-3xl font-semibold text-text">Friends</h1>
    <p class="text-[0.875rem] text-text-muted">People you split bills with.</p>
  </header>

  <!-- Incoming requests -->
  {#if incoming.length > 0}
    <section class="flex flex-col gap-2" transition:slide={{ duration: dur, easing: ease }}>
      <h2 class="font-serif text-xl font-semibold text-text">Incoming requests</h2>
      <ul class="divide-y divide-border overflow-hidden rounded-card border border-border bg-surface-elevated">
        {#each incoming as req (req.id)}
          {@const busy = pendingActionIds.has(req.id)}
          <li
            animate:flip={{ duration: durFast }}
            class="flex flex-wrap items-center justify-between gap-3 px-4 py-3"
          >
            <span class="font-medium text-text">
              {req.requesterDisplayName || 'Unknown'}
            </span>
            <div class="flex gap-2">
              <Button variant="primary" size="sm" onclick={() => accept(req)} loading={busy}>
                <Check size={14} strokeWidth={1.75} /> Accept
              </Button>
              <Button variant="secondary" size="sm" onclick={() => decline(req)} disabled={busy}>
                <X size={14} strokeWidth={1.75} /> Decline
              </Button>
            </div>
          </li>
        {/each}
      </ul>
    </section>
  {/if}

  <!-- Add friends -->
  <section class="flex flex-col gap-2">
    <h2 class="font-serif text-xl font-semibold text-text">Add friends</h2>
    <p class="text-[0.875rem] text-text-muted">
      Search by email address. Friends can be added to bills and groups.
    </p>

    <div class="relative">
      <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle">
        <Search size={16} strokeWidth={1.75} />
      </span>
      <input
        type="search"
        bind:value={searchQuery}
        oninput={handleSearchInput}
        placeholder="friend@example.com"
        class="w-full rounded-input border border-border bg-surface-elevated py-2 pl-9 pr-3 outline-none transition-colors focus:border-primary focus:shadow-[0_0_0_3px_var(--color-primary-soft)]"
      />
    </div>

    {#if searched}
      <div transition:fade={{ duration: durFast, easing: ease }}>
        {#if searchResults.length === 0}
          <p class="rounded-card border border-border bg-surface-elevated px-4 py-3 text-[0.875rem] text-text-muted">
            {searching ? 'Searching…' : 'No one matches that search.'}
          </p>
        {:else}
          <ul class="divide-y divide-border overflow-hidden rounded-card border border-border bg-surface-elevated">
            {#each searchResults as u (u.userId)}
              {@const status = classifyUser(u)}
              {@const busy = pendingActionIds.has(u.userId)}
              <li
                animate:flip={{ duration: durFast }}
                class="flex flex-wrap items-center justify-between gap-3 px-4 py-3"
              >
                <span class="font-medium text-text">{u.displayName}</span>
                {#if status === 'friend'}
                  <Badge tone="success"><Check size={12} strokeWidth={1.75} /> Friends</Badge>
                {:else if status === 'pending'}
                  <Badge tone="neutral">Pending…</Badge>
                {:else}
                  <Button variant="primary" size="sm" onclick={() => addFriend(u)} loading={busy}>
                    <UserPlus size={14} strokeWidth={1.75} /> {busy ? 'Sending…' : 'Add friend'}
                  </Button>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}
  </section>

  <!-- My friends -->
  <section class="flex flex-col gap-2">
    <h2 class="font-serif text-xl font-semibold text-text">My friends</h2>
    {#if loading}
      <div class="flex flex-col gap-2">
        <Skeleton height="h-14" rounded="card" />
        <Skeleton height="h-14" rounded="card" />
        <Skeleton height="h-14" rounded="card" />
      </div>
    {:else if friends.length === 0}
      <EmptyState
        icon={UsersRound}
        title="You haven't added anyone."
        hint="Search above for someone you split with."
      />
    {:else}
      <ul class="divide-y divide-border overflow-hidden rounded-card border border-border bg-surface-elevated">
        {#each friends as f (f.userId)}
          {@const busy = pendingActionIds.has(f.userId)}
          <li
            animate:flip={{ duration: durFast }}
            class="flex flex-wrap items-center justify-between gap-3 px-4 py-3"
          >
            <div class="flex flex-col">
              <span class="font-medium text-text">{f.displayName || 'Unknown'}</span>
              {#if f.email}
                <span class="text-[0.75rem] text-text-muted">{f.email}</span>
              {/if}
            </div>
            <Button variant="ghost" size="sm" onclick={() => unfriend(f)} loading={busy}>
              <UserMinus size={14} strokeWidth={1.75} /> {busy ? 'Removing…' : 'Unfriend'}
            </Button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</main>

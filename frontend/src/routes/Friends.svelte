<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade, slide } from 'svelte/transition';
  import { Check, X, UserPlus, UserMinus, Search, Users } from 'lucide-svelte';
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
    if (!confirm(`Remove ${f.displayName} from your friends?`)) return;
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

<main class="mx-auto flex max-w-3xl flex-col gap-8 px-4 py-6">
  <header class="flex flex-col gap-1">
    <h1 class="font-serif text-3xl font-semibold text-text">Friends</h1>
    <p class="text-sm text-text-muted">People you split bills with.</p>
  </header>

  <!-- Incoming requests -->
  {#if incoming.length > 0}
    <section class="flex flex-col gap-2" transition:slide={{ duration: 180 }}>
      <h2 class="text-xl font-semibold text-text">Incoming requests</h2>
      <ul class="divide-y divide-border overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
        {#each incoming as req (req.id)}
          {@const busy = pendingActionIds.has(req.id)}
          <li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
            <span class="font-medium text-text">
              {req.requesterDisplayName || 'Unknown'}
            </span>
            <div class="flex gap-2">
              <button
                type="button"
                onclick={() => accept(req)}
                disabled={busy}
                class="inline-flex items-center gap-1 rounded-md border border-success/30 bg-success-soft px-3 py-1.5 text-sm font-medium text-success hover:bg-success-soft disabled:opacity-60"
              >
                <Check size={14} /> Accept
              </button>
              <button
                type="button"
                onclick={() => decline(req)}
                disabled={busy}
                class="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm font-medium text-text-muted hover:bg-surface-sunken disabled:opacity-60"
              >
                <X size={14} /> Decline
              </button>
            </div>
          </li>
        {/each}
      </ul>
    </section>
  {/if}

  <!-- Add friends -->
  <section class="flex flex-col gap-2">
    <h2 class="text-xl font-semibold text-text">Add friends</h2>
    <p class="text-sm text-text-muted">
      Search by email address. Friends can be added to bills and groups.
    </p>

    <div class="relative">
      <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-subtle">
        <Search size={16} />
      </span>
      <input
        type="search"
        bind:value={searchQuery}
        oninput={handleSearchInput}
        placeholder="friend@example.com"
        class="w-full rounded-md border border-border bg-surface-elevated py-2 pl-9 pr-3 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft"
      />
    </div>

    {#if searched}
      <div transition:fade={{ duration: 100 }}>
        {#if searchResults.length === 0}
          <p class="rounded-md bg-surface-elevated px-4 py-3 text-sm text-text-muted ring-1 ring-border">
            {searching ? 'Searching…' : 'No users match that search.'}
          </p>
        {:else}
          <ul class="divide-y divide-border overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
            {#each searchResults as u (u.userId)}
              {@const status = classifyUser(u)}
              {@const busy = pendingActionIds.has(u.userId)}
              <li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
                <span class="font-medium text-text">{u.displayName}</span>
                {#if status === 'friend'}
                  <span class="inline-flex items-center gap-1 rounded-md bg-success-soft px-2.5 py-1 text-xs font-medium text-success">
                    <Check size={12} /> Friends
                  </span>
                {:else if status === 'pending'}
                  <span class="inline-flex items-center gap-1 rounded-md bg-surface-sunken px-2.5 py-1 text-xs font-medium text-text-muted">
                    Pending…
                  </span>
                {:else}
                  <button
                    type="button"
                    onclick={() => addFriend(u)}
                    disabled={busy}
                    class="inline-flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-hover disabled:opacity-60"
                  >
                    <UserPlus size={14} /> {busy ? 'Sending…' : 'Add friend'}
                  </button>
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
    <h2 class="text-xl font-semibold text-text">My friends</h2>
    {#if loading}
      <div class="overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
        {#each [1, 2, 3] as _}
          <div class="h-12 animate-pulse border-b border-border last:border-b-0"></div>
        {/each}
      </div>
    {:else if friends.length === 0}
      <div class="flex flex-col items-center gap-2 rounded-lg bg-surface-elevated px-6 py-10 text-center ring-1 ring-border">
        <Users size={28} class="text-text-subtle" />
        <p class="text-text-muted">No friends yet.</p>
        <p class="text-xs text-text-subtle">Search above to send your first request.</p>
      </div>
    {:else}
      <ul class="divide-y divide-border overflow-hidden rounded-lg bg-surface-elevated ring-1 ring-border">
        {#each friends as f (f.userId)}
          {@const busy = pendingActionIds.has(f.userId)}
          <li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
            <div class="flex flex-col">
              <span class="font-medium text-text">{f.displayName || 'Unknown'}</span>
              {#if f.email}
                <span class="text-xs text-text-muted">{f.email}</span>
              {/if}
            </div>
            <button
              type="button"
              onclick={() => unfriend(f)}
              disabled={busy}
              class="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm font-medium text-text-muted hover:bg-surface-sunken disabled:opacity-60"
            >
              <UserMinus size={14} /> {busy ? 'Removing…' : 'Unfriend'}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</main>

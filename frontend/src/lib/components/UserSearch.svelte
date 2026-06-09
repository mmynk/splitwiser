<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { searchFriends } from '$lib/api/friends';
  import { searchUsers as searchAllUsers } from '$lib/api/split';
  import { currentUser } from '$lib/stores/auth';

  export interface UserPick {
    userId: string;
    displayName: string;
    self?: boolean;
  }

  interface Props {
    value?: string;
    placeholder?: string;
    excludeIds?: string[];
    /** If true, search all registered users by exact email instead of friends by display name. */
    global?: boolean;
    /** Minimum chars before searching. */
    minChars?: number;
    /** Debounce delay in ms. */
    debounceMs?: number;
    inputClass?: string;
    disabled?: boolean;
    autofocus?: boolean;
    onInput?: (value: string) => void;
    onSelect: (user: UserPick) => void;
    onEnter?: () => void;
  }

  let {
    value = $bindable(''),
    placeholder = 'Search by name…',
    excludeIds = [],
    global = false,
    minChars = 2,
    debounceMs = 300,
    inputClass = '',
    disabled = false,
    autofocus = false,
    onInput,
    onSelect,
    onEnter,
  }: Props = $props();

  let results: UserPick[] = $state([]);
  let open = $state(false);
  let searched = $state(false);
  let activeIndex = $state(-1);
  let inputEl: HTMLInputElement | null = $state(null);

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let blurTimer: ReturnType<typeof setTimeout> | null = null;
  let latestRequestId = 0;

  function clearDebounce() {
    if (debounceTimer !== null) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
  }

  function clearBlur() {
    if (blurTimer !== null) {
      clearTimeout(blurTimer);
      blurTimer = null;
    }
  }

  function runSearch(query: string) {
    clearDebounce();
    const q = query.trim();
    if (q.length < minChars) {
      latestRequestId++;
      results = [];
      open = false;
      searched = false;
      activeIndex = -1;
      return;
    }
    debounceTimer = setTimeout(async () => {
      const requestId = ++latestRequestId;
      try {
        const r = global ? await searchAllUsers(q) : await searchFriends(q);
        if (requestId !== latestRequestId) return;
        const users = r.users ?? [];
        const filtered: UserPick[] = users
          .filter((u) => !excludeIds.includes(u.userId))
          .map((u) => ({ userId: u.userId, displayName: u.displayName }));
        const me = $currentUser;
        if (
          me &&
          me.displayName &&
          !excludeIds.includes(me.id) &&
          me.displayName.toLowerCase().includes(q.toLowerCase()) &&
          !filtered.some((u) => u.userId === me.id)
        ) {
          filtered.unshift({ userId: me.id, displayName: me.displayName, self: true });
        }
        results = filtered;
        searched = true;
        open = true;
        activeIndex = results.length > 0 ? 0 : -1;
      } catch {
        if (requestId !== latestRequestId) return;
        results = [];
        searched = true;
        open = true;
        activeIndex = -1;
      }
    }, debounceMs);
  }

  function handleInput(e: Event) {
    const v = (e.target as HTMLInputElement).value;
    value = v;
    onInput?.(v);
    runSearch(v);
  }

  function pick(user: UserPick) {
    onSelect(user);
    results = [];
    open = false;
    searched = false;
    activeIndex = -1;
    latestRequestId++;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (open && results.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        activeIndex = (activeIndex + 1) % results.length;
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        activeIndex = (activeIndex - 1 + results.length) % results.length;
        return;
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        if (activeIndex >= 0) pick(results[activeIndex]);
        return;
      }
      if (e.key === 'Escape') {
        open = false;
        activeIndex = -1;
        return;
      }
    } else if (e.key === 'Enter' && onEnter) {
      e.preventDefault();
      onEnter();
    }
  }

  function handleBlur() {
    clearBlur();
    blurTimer = setTimeout(() => {
      open = false;
      blurTimer = null;
    }, 150);
  }

  function handleFocus() {
    clearBlur();
    if (searched) open = true;
  }

  onMount(() => {
    if (autofocus) inputEl?.focus();
  });

  onDestroy(() => {
    clearDebounce();
    clearBlur();
  });
</script>

<div class="relative">
  <input
    bind:this={inputEl}
    type="text"
    {placeholder}
    {disabled}
    value={value}
    autocomplete="off"
    class={inputClass || 'w-full rounded-md border border-border px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary-soft'}
    oninput={handleInput}
    onkeydown={handleKeydown}
    onblur={handleBlur}
    onfocus={handleFocus}
  />

  {#if open && searched}
    <ul
      class="absolute left-0 right-0 top-full z-30 mt-1 max-h-56 overflow-y-auto rounded-md border border-border bg-surface-elevated shadow-lg"
      role="listbox"
    >
      {#if results.length === 0}
        <li class="px-3 py-2 text-sm text-text-muted">No matches</li>
      {:else}
        {#each results as user, i (user.userId)}
          <li>
            <button
              type="button"
              role="option"
              aria-selected={i === activeIndex}
              class="block w-full px-3 py-2 text-left text-sm"
              class:bg-primary-soft={i === activeIndex}
              class:hover:bg-surface-sunken={i !== activeIndex}
              onmousedown={(e) => {
                e.preventDefault();
                pick(user);
              }}
              onmouseenter={() => (activeIndex = i)}
            >
              <strong class="font-medium text-text">{user.displayName}</strong>
              {#if user.self}
                <span class="ml-2 inline-flex items-center rounded-pill bg-primary-soft px-2 py-0.5 align-middle text-[0.7rem] font-medium text-primary">
                  you
                </span>
              {/if}
            </button>
          </li>
        {/each}
      {/if}
    </ul>
  {/if}
</div>

<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { dur, durFast, ease } from '$lib/motion';
  import { Keyboard } from 'lucide-svelte';

  let open = $state(false);

  function onKey(e: KeyboardEvent) {
    const tag = (e.target as HTMLElement | null)?.tagName?.toLowerCase();
    const editable =
      tag === 'input' || tag === 'textarea' || tag === 'select' ||
      (e.target as HTMLElement | null)?.isContentEditable;
    if (e.key === 'Escape' && open) {
      e.preventDefault();
      open = false;
      return;
    }
    if (!editable && e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      open = !open;
    }
  }

  const SHORTCUTS: { keys: string[]; label: string }[] = [
    { keys: ['?'], label: 'Show or hide this help' },
    { keys: ['Esc'], label: 'Close modals and dialogs' },
    { keys: ['Enter'], label: 'Add the next participant or item row' },
    { keys: ['Enter'], label: 'Confirm a dialog' },
  ];
</script>

<svelte:window onkeydown={onKey} />

<button
  type="button"
  aria-label="Keyboard shortcuts (?)"
  title="Keyboard shortcuts — press ?"
  class="fixed bottom-4 left-4 z-30 hidden h-9 w-9 items-center justify-center rounded-pill border border-border bg-surface-elevated text-text-muted hover:text-text hover:border-border-strong sm:inline-flex"
  onclick={() => (open = !open)}
>
  <Keyboard size={16} strokeWidth={1.75} />
</button>

{#if open}
  <div class="fixed inset-0 z-40 flex items-end justify-center p-4 sm:items-center">
    <button
      type="button"
      tabindex="-1"
      aria-label="Close"
      class="absolute inset-0 cursor-default bg-text/40"
      transition:fade={{ duration: durFast, easing: ease }}
      onclick={() => (open = false)}
    ></button>

    <div
      role="dialog"
      aria-labelledby="shortcuts-title"
      aria-modal="true"
      class="relative z-10 w-full max-w-sm overflow-hidden rounded-card border border-border bg-surface-elevated shadow-modal"
      transition:fly={{ y: 8, duration: dur, easing: ease }}
    >
      <header class="flex items-center justify-between border-b border-border px-5 py-3">
        <h2 id="shortcuts-title" class="font-serif text-base font-semibold text-text">
          Keyboard shortcuts
        </h2>
        <span class="text-[0.75rem] text-text-subtle">press ? to toggle</span>
      </header>
      <ul class="divide-y divide-border">
        {#each SHORTCUTS as s}
          <li class="flex items-center justify-between gap-3 px-5 py-3">
            <span class="text-[0.875rem] text-text">{s.label}</span>
            <span class="flex items-center gap-1">
              {#each s.keys as k}
                <kbd class="rounded border border-border bg-surface-sunken px-1.5 py-0.5 font-mono text-[0.6875rem] text-text">{k}</kbd>
              {/each}
            </span>
          </li>
        {/each}
      </ul>
    </div>
  </div>
{/if}

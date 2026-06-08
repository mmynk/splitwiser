<script lang="ts">
  import { fade, scale } from 'svelte/transition';
  import { X } from 'lucide-svelte';
  import type { Snippet } from 'svelte';

  interface Props {
    open: boolean;
    title?: string;
    onClose: () => void;
    children: Snippet;
    footer?: Snippet;
    closeOnOverlay?: boolean;
    maxWidth?: string;
  }

  let {
    open,
    title,
    onClose,
    children,
    footer,
    closeOnOverlay = true,
    maxWidth = 'max-w-lg',
  }: Props = $props();

  let dialogEl: HTMLDivElement | null = $state(null);
  const headingId = `modal-title-${crypto.randomUUID()}`;

  function handleKey(e: KeyboardEvent) {
    if (open && e.key === 'Escape') {
      e.preventDefault();
      onClose();
    }
  }

  function handleOverlayClick() {
    if (closeOnOverlay) onClose();
  }

  $effect(() => {
    if (open && dialogEl) dialogEl.focus();
  });
</script>

<svelte:window onkeydown={handleKey} />

{#if open}
  <div
    class="fixed inset-0 z-40 flex items-center justify-center p-4"
    transition:fade={{ duration: 120 }}
  >
    <button
      type="button"
      tabindex="-1"
      aria-label="Close modal"
      class="absolute inset-0 cursor-default bg-slate-900/40"
      onclick={handleOverlayClick}
    ></button>

    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? headingId : undefined}
      aria-label={title ? undefined : 'Dialog'}
      tabindex="-1"
      class="relative z-10 flex w-full {maxWidth} max-h-[90vh] flex-col overflow-hidden rounded-lg bg-white shadow-xl outline-none ring-1 ring-slate-200"
      transition:scale={{ duration: 150, start: 0.96 }}
    >
      {#if title}
        <header class="flex items-center justify-between border-b border-slate-200 px-4 py-3">
          <h2 id={headingId} class="text-base font-semibold text-slate-900">{title}</h2>
          <button
            type="button"
            aria-label="Close"
            class="rounded p-1 text-slate-500 hover:bg-slate-100"
            onclick={onClose}
          >
            <X size={16} />
          </button>
        </header>
      {/if}

      <div class="flex-1 overflow-y-auto px-4 py-4">
        {@render children()}
      </div>

      {#if footer}
        <footer class="border-t border-slate-200 px-4 py-3">
          {@render footer()}
        </footer>
      {/if}
    </div>
  </div>
{/if}

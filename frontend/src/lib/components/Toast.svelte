<script lang="ts">
  import { fly } from 'svelte/transition';
  import { toasts, type ToastKind } from '$lib/stores/toast';
  import { X } from 'lucide-svelte';

  const KIND_CLASSES: Record<ToastKind, string> = {
    info: 'bg-surface-elevated ring-border text-text',
    success: 'bg-success-soft ring-emerald-200 text-success',
    error: 'bg-danger-soft ring-red-200 text-danger',
  };
</script>

<div
  class="pointer-events-none fixed inset-x-4 bottom-4 z-50 flex flex-col gap-2 sm:left-auto sm:right-4 sm:w-full sm:max-w-sm"
  aria-live="polite"
  aria-atomic="false"
>
  {#each $toasts as t (t.id)}
    <div
      class="pointer-events-auto flex items-start gap-2 rounded-md px-3 py-2 shadow-md ring-1 {KIND_CLASSES[t.kind]}"
      role={t.kind === 'error' ? 'alert' : 'status'}
      transition:fly={{ y: 8, duration: 180 }}
    >
      <div class="flex-1 text-sm">{t.message}</div>
      <button
        type="button"
        class="rounded p-0.5 text-text-muted hover:bg-black/5"
        aria-label="Dismiss notification"
        onclick={() => toasts.dismiss(t.id)}
      >
        <X size={14} />
      </button>
    </div>
  {/each}
</div>

<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { confirmState, resolveConfirm } from '$lib/stores/confirm';
  import Button from '$lib/components/ui/Button.svelte';
  import { dur, durFast, ease } from '$lib/motion';

  function onKey(e: KeyboardEvent) {
    if (!$confirmState) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      resolveConfirm(false);
      return;
    }
    if (e.key !== 'Enter') return;
    // Don't hijack Enter when the user is typing in a body input. (No inputs
    // exist in the dialog today, but a future caller might add one.)
    const target = e.target as HTMLElement | null;
    if (target?.closest('input,textarea,[contenteditable="true"]')) return;
    e.preventDefault();
    resolveConfirm(true);
  }
</script>

<svelte:window onkeydown={onKey} />

{#if $confirmState}
  {@const req = $confirmState}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <button
      type="button"
      tabindex="-1"
      aria-label="Dismiss"
      class="absolute inset-0 cursor-default bg-text/50"
      transition:fade={{ duration: durFast, easing: ease }}
      onclick={() => resolveConfirm(false)}
    ></button>

    <div
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-title"
      class="relative z-10 w-full max-w-sm overflow-hidden rounded-card border border-border bg-surface-elevated shadow-modal"
      transition:fly={{ y: 8, duration: dur, easing: ease }}
    >
      <div class="flex flex-col gap-2 px-5 pt-5 pb-3">
        <h2 id="confirm-title" class="font-serif text-lg font-semibold text-text">
          {req.title}
        </h2>
        {#if req.body}
          <p class="text-[0.875rem] text-text-muted">{req.body}</p>
        {/if}
      </div>
      <div class="flex justify-end gap-2 border-t border-border bg-surface-sunken/60 px-5 py-3">
        <Button variant="ghost" size="sm" onclick={() => resolveConfirm(false)}>
          {req.cancelLabel}
        </Button>
        <Button
          variant={req.tone === 'danger' ? 'danger' : 'primary'}
          size="sm"
          onclick={() => resolveConfirm(true)}
        >
          {req.confirmLabel}
        </Button>
      </div>
    </div>
  </div>
{/if}

<script lang="ts">
  import { fly, fade } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { CircleCheck, CircleAlert, Info, X } from 'lucide-svelte';
  import { toasts, type ToastKind } from '$lib/stores/toast';
  import { dur, durFast, ease } from '$lib/motion';

  const KIND: Record<ToastKind, { surface: string; accent: string; icon: typeof Info; iconLabel: string }> = {
    info: {
      surface: 'bg-surface-elevated border-border text-text',
      accent: 'text-text-muted',
      icon: Info,
      iconLabel: 'Info',
    },
    success: {
      surface: 'bg-success-soft border-success/30 text-success',
      accent: 'text-success',
      icon: CircleCheck,
      iconLabel: 'Success',
    },
    error: {
      surface: 'bg-danger-soft border-danger/30 text-danger',
      accent: 'text-danger',
      icon: CircleAlert,
      iconLabel: 'Error',
    },
  };
</script>

<div
  class="pointer-events-none fixed inset-x-4 bottom-4 z-50 flex flex-col gap-2 sm:left-auto sm:right-6 sm:w-full sm:max-w-sm"
  aria-live="polite"
  aria-atomic="false"
>
  {#each $toasts as t (t.id)}
    {@const kind = KIND[t.kind]}
    {@const Icon = kind.icon}
    <div
      animate:flip={{ duration: durFast }}
      in:fly={{ y: 8, duration: dur, easing: ease }}
      out:fade={{ duration: durFast, easing: ease }}
      class={[
        'pointer-events-auto flex items-start gap-2.5 rounded-card border px-3.5 py-2.5',
        kind.surface,
      ]}
      role={t.kind === 'error' ? 'alert' : 'status'}
    >
      <span class={['mt-0.5 shrink-0', kind.accent]} aria-label={kind.iconLabel}>
        <Icon size={16} strokeWidth={1.75} aria-hidden="true" />
      </span>
      <div class="flex-1 text-[0.875rem] leading-snug">{t.message}</div>
      <button
        type="button"
        class="rounded-input p-0.5 text-text-muted hover:bg-surface-sunken focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        aria-label="Dismiss notification"
        onclick={() => toasts.dismiss(t.id)}
      >
        <X size={14} strokeWidth={1.75} />
      </button>
    </div>
  {/each}
</div>

<script lang="ts">
  import type { Snippet } from 'svelte';
  import { fade } from 'svelte/transition';
  import { durFast, ease } from '$lib/motion';

  type Tone = 'danger' | 'warning' | 'success' | 'info';

  interface Props {
    tone?: Tone;
    role?: 'alert' | 'status';
    children: Snippet;
  }

  let { tone = 'danger', role = 'alert', children }: Props = $props();

  const TONE: Record<Tone, string> = {
    danger: 'border-danger/30 bg-danger-soft text-danger',
    warning: 'border-warning/30 bg-warning-soft text-warning',
    success: 'border-success/30 bg-success-soft text-success',
    info: 'border-border bg-surface-sunken text-text',
  };
</script>

<div
  {role}
  transition:fade={{ duration: durFast, easing: ease }}
  class={['rounded-input border px-3 py-2 text-[0.875rem]', TONE[tone]]}
>
  {@render children()}
</div>

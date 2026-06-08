<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    elevated?: boolean;
    padding?: 'none' | 'sm' | 'md' | 'lg';
    as?: 'div' | 'section' | 'article';
    children: Snippet;
  }

  let { elevated = false, padding = 'md', as = 'div', children }: Props = $props();

  const PAD = {
    none: '',
    sm: 'p-3',
    md: 'p-5',
    lg: 'p-6 sm:p-8',
  } as const;
</script>

<svelte:element
  this={as}
  class={[
    'rounded-card bg-surface-elevated border border-border',
    elevated && 'shadow-pop',
    PAD[padding],
  ]}
>
  {@render children()}
</svelte:element>

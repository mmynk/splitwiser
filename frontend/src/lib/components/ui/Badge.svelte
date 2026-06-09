<script lang="ts">
  import type { Snippet } from 'svelte';

  type Tone = 'neutral' | 'primary' | 'success' | 'danger' | 'warning';

  interface Props {
    tone?: Tone;
    children: Snippet;
    title?: string;
  }

  let { tone = 'neutral', children, title }: Props = $props();

  const TONE: Record<Tone, string> = {
    neutral: 'bg-surface-sunken text-text-muted border-border',
    primary: 'bg-primary-soft text-primary border-primary/20',
    success: 'bg-success-soft text-success border-success/20',
    danger: 'bg-danger-soft text-danger border-danger/20',
    warning: 'bg-warning-soft text-warning border-warning/20',
  };
</script>

<span
  {title}
  class={[
    'inline-flex items-center gap-1 rounded-pill border px-2 py-0.5 text-[0.6875rem] font-medium',
    TONE[tone],
  ]}
>
  {@render children()}
</span>

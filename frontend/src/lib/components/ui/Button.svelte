<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Loader2 } from 'lucide-svelte';

  type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
  type Size = 'sm' | 'md' | 'lg';

  interface Props {
    variant?: Variant;
    size?: Size;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    loading?: boolean;
    fullWidth?: boolean;
    onclick?: (e: MouseEvent) => void;
    ariaLabel?: string;
    form?: string;
    children: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    type = 'button',
    disabled = false,
    loading = false,
    fullWidth = false,
    onclick,
    ariaLabel,
    form,
    children,
  }: Props = $props();

  const VARIANT: Record<Variant, string> = {
    primary:
      'bg-primary text-primary-foreground hover:bg-primary-hover active:bg-primary-hover',
    secondary:
      'bg-surface-elevated text-text border border-border hover:bg-surface-sunken active:bg-surface-sunken',
    ghost:
      'bg-transparent text-text hover:bg-surface-sunken active:bg-surface-sunken',
    danger:
      'bg-danger text-white hover:opacity-90 active:opacity-90',
  };

  const SIZE: Record<Size, string> = {
    sm: 'h-8 px-3 text-[0.8125rem] gap-1.5',
    md: 'h-10 px-4 text-[0.9375rem] gap-2',
    lg: 'h-12 px-5 text-base gap-2',
  };

  const SPINNER_SIZE: Record<Size, number> = { sm: 14, md: 16, lg: 18 };
</script>

<button
  {type}
  {form}
  disabled={disabled || loading}
  onclick={onclick}
  aria-label={ariaLabel}
  aria-busy={loading || undefined}
  class={[
    'inline-flex items-center justify-center rounded-input font-medium',
    'transition-[background-color,color,opacity] duration-150',
    'disabled:cursor-not-allowed disabled:opacity-55',
    'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
    fullWidth && 'w-full',
    VARIANT[variant],
    SIZE[size],
  ]}
>
  {#if loading}
    <Loader2 size={SPINNER_SIZE[size]} strokeWidth={1.75} class="animate-spin" />
  {/if}
  {@render children()}
</button>

<script lang="ts">
  import type { Snippet } from 'svelte';

  type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
  type Size = 'sm' | 'md' | 'lg';

  interface Props {
    variant?: Variant;
    size?: Size;
    type?: 'button' | 'submit';
    disabled?: boolean;
    onclick?: (e: MouseEvent) => void;
    ariaLabel: string;
    title?: string;
    children: Snippet;
  }

  let {
    variant = 'ghost',
    size = 'md',
    type = 'button',
    disabled = false,
    onclick,
    ariaLabel,
    title,
    children,
  }: Props = $props();

  const VARIANT: Record<Variant, string> = {
    primary: 'bg-primary text-primary-foreground hover:bg-primary-hover',
    secondary: 'bg-surface-elevated text-text border border-border hover:bg-surface-sunken',
    ghost: 'bg-transparent text-text-muted hover:bg-surface-sunken hover:text-text',
    danger: 'bg-transparent text-danger hover:bg-danger-soft',
  };

  const SIZE: Record<Size, string> = {
    sm: 'h-7 w-7',
    md: 'h-9 w-9',
    lg: 'h-11 w-11',
  };
</script>

<button
  {type}
  {disabled}
  {onclick}
  {title}
  aria-label={ariaLabel}
  class={[
    'inline-flex items-center justify-center rounded-input',
    'transition-[background-color,color] duration-150',
    'disabled:cursor-not-allowed disabled:opacity-55',
    'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
    VARIANT[variant],
    SIZE[size],
  ]}
>
  {@render children()}
</button>

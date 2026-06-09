<script lang="ts">
  import { tweened } from 'svelte/motion';
  import { cubicOut } from 'svelte/easing';
  import { dur } from '$lib/motion';

  interface Props {
    value: number;
    signed?: boolean;
    showPlus?: boolean;
    currency?: string;
    abbreviate?: boolean;
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'display';
    animate?: boolean;
  }

  let {
    value,
    signed = false,
    showPlus = false,
    currency = '$',
    abbreviate = false,
    size = 'md',
    animate = false,
  }: Props = $props();

  const SIZE = {
    sm: 'text-[0.8125rem]',
    md: 'text-[0.9375rem]',
    lg: 'text-lg',
    xl: 'text-2xl',
    display: 'text-3xl sm:text-4xl',
  } as const;

  // Tween only when `animate` is enabled. The first effect run snaps the tween
  // to the current value so initial paint never counts up.
  const tween = tweened(0, { duration: dur * 1.6, easing: cubicOut });
  let mounted = $state(false);
  $effect(() => {
    if (!animate) return;
    const v = value;
    tween.set(v, mounted ? undefined : { duration: 0 });
    mounted = true;
  });

  const display = $derived(animate ? $tween : value);

  const colorClass = $derived(
    signed
      ? value > 0
        ? 'text-success'
        : value < 0
          ? 'text-danger'
          : 'text-text-muted'
      : 'text-text',
  );

  const formatted = $derived.by(() => {
    const abs = Math.abs(display);
    let body: string;
    // Only abbreviate well above bill-splitting range — never round away cents
    // on a settlement-relevant amount by default.
    if (abbreviate && abs >= 100_000) {
      body = `${Math.round(abs / 1000)}k`;
    } else {
      body = abs.toLocaleString(undefined, {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      });
    }
    const sign = display < 0 ? '−' : showPlus && display > 0 ? '+' : '';
    return `${sign}${currency}${body}`;
  });
</script>

<span class="money {SIZE[size]} {colorClass}">{formatted}</span>

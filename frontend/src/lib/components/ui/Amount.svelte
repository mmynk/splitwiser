<script lang="ts">
  interface Props {
    value: number;
    signed?: boolean;
    showPlus?: boolean;
    currency?: string;
    abbreviate?: boolean;
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'display';
  }

  let {
    value,
    signed = false,
    showPlus = false,
    currency = '$',
    abbreviate = false,
    size = 'md',
  }: Props = $props();

  const SIZE = {
    sm: 'text-[0.8125rem]',
    md: 'text-[0.9375rem]',
    lg: 'text-lg',
    xl: 'text-2xl',
    display: 'text-3xl sm:text-4xl',
  } as const;

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
    const abs = Math.abs(value);
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
    const sign = value < 0 ? '−' : showPlus && value > 0 ? '+' : '';
    return `${sign}${currency}${body}`;
  });
</script>

<span class="money {SIZE[size]} {colorClass}">{formatted}</span>

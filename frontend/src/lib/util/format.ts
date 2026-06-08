export function formatMoney(n: number | undefined | null): string {
  return `$${(n ?? 0).toFixed(2)}`;
}

export function formatDate(unixSeconds: number | undefined | null): string {
  if (!unixSeconds) return '';
  return new Date(unixSeconds * 1000).toLocaleDateString();
}

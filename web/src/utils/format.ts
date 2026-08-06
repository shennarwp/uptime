export function formatResponseTime(ms: number): string {
  if (ms <= 9999) return `${ms}ms`;
  const seconds = ms / 1000;
  return `${Math.round(seconds * 100) / 100}s`;
}

import { describe, it, expect } from 'vitest';
import { formatDateTime } from './datetime';

const LOCAL = /^[A-Z][a-z]{2}, \d{2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2}:\d{2}$/;

describe('formatDateTime', () => {
  it('produces RFC 2822 style output without an offset', () => {
    const out = formatDateTime(new Date('2026-08-06T12:00:00Z'));
    expect(out).toMatch(LOCAL);
    expect(out).not.toMatch(/[+-]\d{4}$/);
  });

  it('round-trips through Date.parse', () => {
    const date = new Date('2026-08-06T12:00:00Z');
    const out = formatDateTime(date);
    expect(Date.parse(out)).toBe(date.getTime());
  });

  it('uses the local wall-clock time', () => {
    const date = new Date('2026-08-06T12:00:00Z');
    const out = formatDateTime(date);
    expect(out.endsWith(`${String(date.getHours()).padStart(2, '0')}:00:00`)).toBe(true);
  });

  it('pads day, month, and time with leading zeros', () => {
    const out = formatDateTime(new Date(2026, 7, 6, 8, 5, 3));
    expect(out).toMatch(/^[A-Z][a-z]{2}, 06 Aug 2026 08:05:03$/);
  });
});

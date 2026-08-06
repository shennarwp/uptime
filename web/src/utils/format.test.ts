import { describe, expect, it } from 'vitest';
import { formatResponseTime } from './format';

describe('formatResponseTime', () => {
  it('shows milliseconds up to four digits', () => {
    expect(formatResponseTime(0)).toBe('0ms');
    expect(formatResponseTime(45)).toBe('45ms');
    expect(formatResponseTime(9999)).toBe('9999ms');
  });

  it('shows seconds above four digits', () => {
    expect(formatResponseTime(10000)).toBe('10s');
    expect(formatResponseTime(15000)).toBe('15s');
    expect(formatResponseTime(12345)).toBe('12.35s');
  });
});

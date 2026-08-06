import { describe, expect, it } from 'vitest';
import { formatResponseTime } from './format';

describe('formatResponseTime', () => {
  it('shows milliseconds up to three digits', () => {
    expect(formatResponseTime(0)).toBe('0ms');
    expect(formatResponseTime(45)).toBe('45ms');
    expect(formatResponseTime(999)).toBe('999ms');
  });

  it('shows seconds above three digits', () => {
    expect(formatResponseTime(1000)).toBe('1s');
    expect(formatResponseTime(1234)).toBe('1.23s');
    expect(formatResponseTime(15000)).toBe('15s');
    expect(formatResponseTime(12345)).toBe('12.35s');
  });
});

import { render, screen } from '@testing-library/react';
import { CheckHistoryBar } from './CheckHistoryBar';
import { describe, it, expect, vi } from 'vitest';

describe('CheckHistoryBar component', () => {
  it('renders history bar with checks', () => {
    const checks = [
      {
        id: 1,
        target_id: 1,
        status_code: 200,
        response_time_ms: 50,
        is_up: true,
        checked_at: new Date().toISOString(),
      },
      {
        id: 2,
        target_id: 1,
        status_code: 500,
        response_time_ms: 120,
        is_up: false,
        error_message: 'Server Error',
        checked_at: new Date().toISOString(),
      },
    ];

    render(<CheckHistoryBar checks={checks} />);

    expect(screen.getByText('Recent Checks History (Oldest → Newest):')).toBeInTheDocument();
  });

  it('reports the rendered history width', () => {
    const onWidthChange = vi.fn();
    const previousResizeObserver = globalThis.ResizeObserver;
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(private readonly callback: ResizeObserverCallback) {}

        observe() {
          this.callback(
            [{ contentRect: { width: 864 } } as ResizeObserverEntry],
            this as unknown as ResizeObserver,
          );
        }

        disconnect() {}

        unobserve() {}
      },
    );

    try {
      render(<CheckHistoryBar checks={[]} onWidthChange={onWidthChange} />);

      expect(onWidthChange).toHaveBeenCalledWith(864);
    } finally {
      vi.stubGlobal('ResizeObserver', previousResizeObserver);
    }
  });
});

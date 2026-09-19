import { render, screen, fireEvent } from '@testing-library/react';
import App from './App';
import { describe, it, expect, vi, afterEach } from 'vitest';

const fetchMock = vi.fn().mockResolvedValue({
  ok: true,
  json: async () => [],
});

vi.stubGlobal('fetch', fetchMock);
const eventSources: Array<{ close: ReturnType<typeof vi.fn>; onmessage: (() => void) | null }> = [];
vi.stubGlobal(
  'EventSource',
  class {
    close = vi.fn();
    onmessage: (() => void) | null = null;

    constructor() {
      eventSources.push(this);
    }
  },
);

afterEach(() => {
  fetchMock.mockClear();
  eventSources.length = 0;
  localStorage.clear();
});

describe('App add-target tile', () => {
  it('does not show the add tile when logged out', async () => {
    render(<App />);

    expect(await screen.findByText('No targets configured.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add new target' })).not.toBeInTheDocument();
  });

  it('shows the add tile when logged in and opens the add modal', async () => {
    localStorage.setItem('uptimeApiToken', 'tok');
    render(<App />);

    const tile = await screen.findByRole('button', { name: 'Add new target' });
    fireEvent.click(tile);

    expect(screen.getByText('Add Target')).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^Name/ })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^URL/ })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^Schedule/ })).toBeInTheDocument();
  });

  it('refreshes from server-sent events and closes the stream on unmount', async () => {
    const { unmount } = render(<App />);
    await screen.findByText('No targets configured.');
    expect(eventSources).toHaveLength(1);
    eventSources[0].onmessage?.();
    expect(fetchMock).toHaveBeenCalled();
    unmount();
    expect(eventSources[0].close).toHaveBeenCalled();
  });
});

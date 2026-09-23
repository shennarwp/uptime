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
  window.history.pushState({}, '', '/');
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

  it('loads incidents and marks them all as read from the header', async () => {
    localStorage.setItem('uptimeApiToken', 'tok');
    fetchMock.mockImplementation((url: string) =>
      Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/incidents')
            ? [
                {
                  id: 1,
                  type: 'going_down',
                  target_name: 'Example',
                  target_url: 'https://example.com',
                  timestamp: '2026-09-22T12:00:00Z',
                  started_at: '2026-09-22T12:00:00Z',
                  is_read: false,
                },
              ]
            : [],
      }),
    );
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: 'View incidents' }));
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Mark all as read' }));
    await screen.findByRole('button', { name: 'Mark all as read' });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/incidents/read',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('marks one incident as read from the header', async () => {
    localStorage.setItem('uptimeApiToken', 'tok');
    fetchMock.mockImplementation((url: string) =>
      Promise.resolve({
        ok: true,
        json: async () =>
          url.includes('/incidents')
            ? [
                {
                  id: 1,
                  type: 'going_down',
                  target_name: 'Example',
                  target_url: 'https://example.com',
                  timestamp: '2026-09-22T12:00:00Z',
                  started_at: '2026-09-22T12:00:00Z',
                  is_read: false,
                },
              ]
            : [],
      }),
    );
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: 'View incidents' }));
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Mark Target went down as read' }));
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/incident/1/read',
      expect.objectContaining({ method: 'PATCH' }),
    );
  });
});

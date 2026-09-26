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

  it('renders a target from the dashboard data', async () => {
    fetchMock.mockImplementation((url: string) =>
      Promise.resolve({
        ok: true,
        json: async () =>
          url === '/api/v1/targets'
            ? [
                {
                  id: 1,
                  name: 'Example target',
                  url: 'https://example.com',
                  schedule: '@every 1m',
                  checks: [],
                },
              ]
            : [],
      }),
    );

    render(<App />);

    expect((await screen.findAllByText('Example target')).length).toBe(2);
  });

  it('requests buffered history for the measured history width', async () => {
    const target = {
      id: 1,
      name: 'Example target',
      url: 'https://example.com',
      schedule: '@every 1m',
      checks: [],
    };
    fetchMock.mockImplementation((url: string) =>
      Promise.resolve({
        ok: true,
        json: async () => (url.includes('/targets') ? [target] : []),
      }),
    );
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
      render(<App />);

      expect(await screen.findAllByText('Example target')).toHaveLength(2);
      expect(fetchMock).toHaveBeenCalledWith('/api/v1/targets?checks_limit=135');
    } finally {
      vi.stubGlobal('ResizeObserver', previousResizeObserver);
    }
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

  it('returns from the incidents page and responds to browser history', async () => {
    localStorage.setItem('uptimeApiToken', 'tok');
    render(<App />);

    fireEvent.click(await screen.findByRole('button', { name: 'View incidents' }));
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '← Back to targets' }));
    expect(await screen.findByText('No targets configured.')).toBeInTheDocument();

    window.history.pushState({}, '', '/incidents');
    window.dispatchEvent(new PopStateEvent('popstate'));
    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
  });

  it('starts on the incidents page when the URL already points there', async () => {
    localStorage.setItem('uptimeApiToken', 'tok');
    window.history.pushState({}, '', '/incidents');

    render(<App />);

    expect(await screen.findByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
  });

  it('keeps incident state unchanged when read mutations fail', async () => {
    localStorage.setItem('uptimeApiToken', 'tok');
    fetchMock.mockImplementation((url: string) =>
      Promise.resolve({
        ok: url === '/api/v1/targets',
        json: async () =>
          url === '/api/v1/incidents'
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
    fireEvent.click(screen.getByRole('button', { name: 'Mark Target went down as read' }));
    fireEvent.click(screen.getByRole('button', { name: 'Mark all as read' }));

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/incident/1/read',
      expect.objectContaining({ method: 'PATCH' }),
    );
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/incidents/read',
      expect.objectContaining({ method: 'POST' }),
    );
  });
});

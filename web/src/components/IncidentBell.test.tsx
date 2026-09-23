import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { IncidentBell, type Incident } from './IncidentBell';

const incidents: Incident[] = [
  {
    id: 1,
    type: 'going_down',
    target_name: 'Example',
    target_url: 'https://example.com',
    timestamp: '2026-09-22T12:00:00Z',
    started_at: '2026-09-22T12:00:00Z',
    is_read: false,
  },
  {
    id: 2,
    type: 'cert_expired',
    target_name: 'Docs',
    target_url: 'https://docs.example.com',
    timestamp: '2026-09-21T12:00:00Z',
    started_at: '2026-09-21T12:00:00Z',
    is_read: true,
  },
];

describe('IncidentBell', () => {
  it('shows unread incidents and marks one incident as read', async () => {
    const onMarkRead = vi.fn().mockResolvedValue(undefined);
    render(<IncidentBell incidents={incidents} onMarkRead={onMarkRead} onMarkAllRead={vi.fn()} />);

    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }));
    expect(screen.getByText('Target went down')).toBeInTheDocument();
    expect(screen.getByText('Certificate expired')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Mark Target went down as read' }));

    await waitFor(() => expect(onMarkRead).toHaveBeenCalledWith(1));
  });

  it('marks all incidents as read and closes on outside click', async () => {
    const onMarkAllRead = vi.fn().mockResolvedValue(undefined);
    render(
      <IncidentBell incidents={incidents} onMarkRead={vi.fn()} onMarkAllRead={onMarkAllRead} />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }));
    fireEvent.click(screen.getByRole('button', { name: 'Mark all as read' }));
    await waitFor(() => expect(onMarkAllRead).toHaveBeenCalled());
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('dialog', { name: 'Incidents' })).not.toBeInTheDocument();
  });

  it('renders an empty state', () => {
    render(<IncidentBell incidents={[]} onMarkRead={vi.fn()} onMarkAllRead={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }));
    expect(screen.getByText('No incidents yet.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeDisabled();
  });
});

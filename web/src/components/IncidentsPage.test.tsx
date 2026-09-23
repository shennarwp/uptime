import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { IncidentsPage } from './IncidentsPage';

const incident = {
  id: 1,
  type: 'going_down',
  target_name: 'Example',
  target_url: 'https://example.com',
  timestamp: '2026-09-23 10:38:00',
  started_at: '2026-09-23 10:38:00',
  is_read: false,
};

describe('IncidentsPage', () => {
  it('renders incidents with target-style timestamps and marks one read', async () => {
    const onMarkRead = vi.fn().mockResolvedValue(undefined);
    render(
      <IncidentsPage
        incidents={[incident]}
        onBack={vi.fn()}
        onMarkRead={onMarkRead}
        onMarkAllRead={vi.fn()}
      />,
    );

    expect(screen.getByRole('heading', { name: 'Incidents' })).toBeInTheDocument();
    expect(screen.getByText('Wed, 23 Sep 2026 10:38:00')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Mark Target went down as read' }));

    await waitFor(() => expect(onMarkRead).toHaveBeenCalledWith(1));
  });

  it('marks all read and navigates back', async () => {
    const onBack = vi.fn();
    const onMarkAllRead = vi.fn().mockResolvedValue(undefined);
    render(
      <IncidentsPage
        incidents={[incident]}
        onBack={onBack}
        onMarkRead={vi.fn()}
        onMarkAllRead={onMarkAllRead}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Mark all as read' }));
    fireEvent.click(screen.getByRole('button', { name: '← Back to targets' }));

    await waitFor(() => expect(onMarkAllRead).toHaveBeenCalled());
    expect(onBack).toHaveBeenCalled();
  });

  it('renders the empty state', () => {
    render(
      <IncidentsPage
        incidents={[]}
        onBack={vi.fn()}
        onMarkRead={vi.fn()}
        onMarkAllRead={vi.fn()}
      />,
    );

    expect(screen.getByText('No incidents yet.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeDisabled();
  });
});

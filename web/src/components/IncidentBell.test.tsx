import { fireEvent, render, screen } from '@testing-library/react';
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
  it('shows the unread dot and opens the incidents page', () => {
    const onOpenIncidents = vi.fn();
    render(<IncidentBell incidents={incidents} onOpenIncidents={onOpenIncidents} />);

    fireEvent.click(screen.getByRole('button', { name: 'View incidents' }));

    expect(onOpenIncidents).toHaveBeenCalled();
    expect(screen.getByLabelText('1 unread incidents')).toBeInTheDocument();
  });

  it('does not show the unread dot when all incidents are read', () => {
    render(
      <IncidentBell
        incidents={incidents.map((incident) => ({ ...incident, is_read: true }))}
        onOpenIncidents={vi.fn()}
      />,
    );

    expect(screen.queryByLabelText(/unread incidents/)).not.toBeInTheDocument();
  });
});

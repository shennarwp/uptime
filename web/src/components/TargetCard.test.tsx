import { render, screen, fireEvent } from '@testing-library/react';
import { TargetCard } from './TargetCard';
import { describe, it, expect, vi } from 'vitest';

describe('TargetCard component', () => {
  const target = {
    id: 1,
    name: 'Example Target',
    url: 'https://example.com',
    schedule: '@every 1m',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    checks: [
      {
        id: 1,
        target_id: 1,
        status_code: 200,
        response_time_ms: 45,
        is_up: true,
        checked_at: new Date().toISOString(),
      },
    ],
  };

  it('renders target name, url, status, and response time', () => {
    render(<TargetCard target={target} isHighlighted={false} canEdit={true} onUpdate={vi.fn()} />);

    expect(screen.getByText('Example Target')).toBeInTheDocument();
    expect(screen.getByText('UP')).toBeInTheDocument();
    expect(screen.getByText('45ms')).toBeInTheDocument();
  });

  it('hides the edit button when not logged in', () => {
    render(<TargetCard target={target} isHighlighted={false} canEdit={false} onUpdate={vi.fn()} />);
    expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument();
  });

  it('opens the edit popup and calls onUpdate on save', async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    render(<TargetCard target={target} isHighlighted={false} canEdit={true} onUpdate={onUpdate} />);

    fireEvent.click(screen.getByRole('button', { name: /edit/i }));

    const nameInput = screen.getByRole('textbox', { name: /^Name/ });
    const scheduleInput = screen.getByRole('textbox', { name: /^Schedule/ });
    expect(nameInput).toHaveValue('Example Target');
    expect(scheduleInput).toHaveValue('@every 1m');

    expect(screen.getByLabelText('Name restrictions')).toHaveTextContent(/100 characters/);
    expect(screen.getByLabelText('Schedule format')).toHaveTextContent(/6-field cron/);

    fireEvent.change(nameInput, { target: { value: 'Renamed' } });
    fireEvent.change(scheduleInput, { target: { value: '0 0 */3 * * *' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(onUpdate).toHaveBeenCalledWith(1, 'Renamed', '0 0 */3 * * *');
  });

  it('closes the edit popup when cancel is clicked', () => {
    const onUpdate = vi.fn();
    render(<TargetCard target={target} isHighlighted={false} canEdit={true} onUpdate={onUpdate} />);

    fireEvent.click(screen.getByRole('button', { name: /edit/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('textbox', { name: /^Name/ })).not.toBeInTheDocument();
    expect(onUpdate).not.toHaveBeenCalled();
  });

  it('shows the certificate expiry when present', () => {
    const certTarget = { ...target, cert_expires_at: '2999-01-01T00:00:00Z' };
    render(
      <TargetCard target={certTarget} isHighlighted={false} canEdit={false} onUpdate={vi.fn()} />,
    );

    expect(screen.getByText(/Cert Expires/)).toBeInTheDocument();
    expect(screen.getByText(/2999/)).toBeInTheDocument();
  });

  it('does not show certificate expiry when absent', () => {
    render(<TargetCard target={target} isHighlighted={false} canEdit={false} onUpdate={vi.fn()} />);

    expect(screen.queryByText(/Cert Expires/)).not.toBeInTheDocument();
  });

  it('warns when the certificate is expired', () => {
    const certTarget = { ...target, cert_expires_at: '2020-01-01T00:00:00Z' };
    render(
      <TargetCard target={certTarget} isHighlighted={false} canEdit={false} onUpdate={vi.fn()} />,
    );

    expect(screen.getByText(/expired \d+d ago/)).toHaveClass('cert-warn');
  });

  it('warns when the certificate expires within 10 days', () => {
    const soon = new Date(Date.now() + 5 * 86400000).toISOString();
    const certTarget = { ...target, cert_expires_at: soon };
    render(
      <TargetCard target={certTarget} isHighlighted={false} canEdit={false} onUpdate={vi.fn()} />,
    );

    expect(screen.getByText(/\d+d left/)).toHaveClass('cert-warn');
  });
});

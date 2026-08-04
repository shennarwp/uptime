import { render, screen, fireEvent } from '@testing-library/react';
import { TargetCard } from './TargetCard';
import { describe, it, expect, vi } from 'vitest';

describe('TargetCard component', () => {
  const target = {
    id: 1,
    name: 'Example Target',
    url: 'https://example.com',
    schedule: '@every 1m',
    created_at: { Time: new Date() },
    updated_at: { Time: new Date() },
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
    // @ts-expect-error Timestamp type compatibility in mock test
    render(<TargetCard target={target} isHighlighted={false} onUpdate={vi.fn()} />);

    expect(screen.getByText('Example Target')).toBeInTheDocument();
    expect(screen.getByText('UP')).toBeInTheDocument();
    expect(screen.getByText('45ms')).toBeInTheDocument();
  });

  it('opens the edit popup and calls onUpdate on save', async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    // @ts-expect-error Timestamp type compatibility in mock test
    render(<TargetCard target={target} isHighlighted={false} onUpdate={onUpdate} />);

    fireEvent.click(screen.getByRole('button', { name: /edit/i }));

    const nameInput = screen.getByLabelText('Name');
    const scheduleInput = screen.getByLabelText('Schedule');
    expect(nameInput).toHaveValue('Example Target');
    expect(scheduleInput).toHaveValue('@every 1m');

    fireEvent.change(nameInput, { target: { value: 'Renamed' } });
    fireEvent.change(scheduleInput, { target: { value: '0 0 */3 * * *' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(onUpdate).toHaveBeenCalledWith(1, 'Renamed', '0 0 */3 * * *');
  });

  it('closes the edit popup when cancel is clicked', () => {
    const onUpdate = vi.fn();
    // @ts-expect-error Timestamp type compatibility in mock test
    render(<TargetCard target={target} isHighlighted={false} onUpdate={onUpdate} />);

    fireEvent.click(screen.getByRole('button', { name: /edit/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByLabelText('Name')).not.toBeInTheDocument();
    expect(onUpdate).not.toHaveBeenCalled();
  });
});

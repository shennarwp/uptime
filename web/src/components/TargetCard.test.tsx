import { render, screen } from '@testing-library/react';
import { TargetCard } from './TargetCard';
import { describe, it, expect } from 'vitest';

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
    render(<TargetCard target={target} isHighlighted={false} />);

    expect(screen.getByText('Example Target')).toBeInTheDocument();
    expect(screen.getByText('UP')).toBeInTheDocument();
    expect(screen.getByText('45ms')).toBeInTheDocument();
  });
});

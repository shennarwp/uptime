import { render, screen, fireEvent } from '@testing-library/react';
import { Sidebar } from './Sidebar';
import { describe, it, expect, vi } from 'vitest';

describe('Sidebar component', () => {
  const targets = [
    { id: 1, name: 'Google' },
    { id: 2, name: 'GitHub' },
  ];

  it('renders targets', () => {
    const onSelect = vi.fn();
    render(<Sidebar targets={targets} selectedId={1} onSelect={onSelect} />);

    expect(screen.getByText('Google')).toBeInTheDocument();
    expect(screen.getByText('GitHub')).toBeInTheDocument();
  });

  it('calls onSelect when a target is clicked', () => {
    const onSelect = vi.fn();
    render(<Sidebar targets={targets} selectedId={1} onSelect={onSelect} />);

    fireEvent.click(screen.getByText('GitHub'));
    expect(onSelect).toHaveBeenCalledWith(2);
  });

  it('toggles open state and changes indicator when header is clicked', () => {
    const onSelect = vi.fn();
    render(<Sidebar targets={targets} selectedId={1} onSelect={onSelect} />);

    const header = screen.getByText('Targets').closest('div');
    const indicator = screen.getByText('▼');
    expect(indicator).toBeInTheDocument();

    if (header) {
      fireEvent.click(header);
    }
    expect(screen.getByText('▲')).toBeInTheDocument();
  });
});

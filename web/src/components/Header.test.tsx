import { render, screen } from '@testing-library/react';
import { Header } from './Header';
import { describe, it, expect } from 'vitest';

describe('Header component', () => {
  it('renders header text', () => {
    render(<Header />);
    expect(screen.getByText('Uptime')).toBeInTheDocument();
  });
});

import { render, screen, fireEvent } from '@testing-library/react';
import { TruncatedUrl } from './TruncatedUrl';
import { describe, it, expect } from 'vitest';

describe('TruncatedUrl component', () => {
  it('renders cleaned and truncated url', () => {
    render(<TruncatedUrl url="https://verylongdomainnameexample.com/some/path" />);
    const link = screen.getByRole('link');
    expect(link).toBeInTheDocument();
    expect(link.textContent).toContain('...');
  });

  it('expands url on mouse enter', () => {
    render(<TruncatedUrl url="https://verylongdomainnameexample.com/some/path" />);
    const link = screen.getByRole('link');

    fireEvent.mouseEnter(link);
    expect(link.textContent).not.toContain('...');
  });
});

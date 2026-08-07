import { render, screen, fireEvent } from '@testing-library/react';
import App from './App';
import { describe, it, expect, vi, afterEach } from 'vitest';

const fetchMock = vi.fn().mockResolvedValue({
  ok: true,
  json: async () => [],
});

vi.stubGlobal('fetch', fetchMock);

afterEach(() => {
  fetchMock.mockClear();
  localStorage.clear();
});

describe('App add-target tile', () => {
  it('does not show the add tile when logged out', async () => {
    render(<App />);

    expect(await screen.findByText('No targets configured.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add new target' })).not.toBeInTheDocument();
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
});

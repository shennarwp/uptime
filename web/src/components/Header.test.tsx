import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Header } from './Header';
import { describe, it, expect, vi } from 'vitest';

describe('Header component', () => {
  it('renders header text', () => {
    render(<Header isLoggedIn={false} onLogin={vi.fn()} onLogout={vi.fn()} />);
    expect(screen.getByText('Uptime')).toBeInTheDocument();
  });

  it('shows a logout button when logged in and logs out on click', () => {
    const onLogout = vi.fn();
    render(<Header isLoggedIn={true} onLogin={vi.fn()} onLogout={onLogout} />);
    fireEvent.click(screen.getByRole('button', { name: 'Logout' }));
    expect(onLogout).toHaveBeenCalled();
  });

  it('shows the incident bell beside logout when logged in', () => {
    render(
      <Header
        isLoggedIn
        onLogin={vi.fn()}
        onLogout={vi.fn()}
        incidents={[
          {
            id: 1,
            type: 'going_down',
            target_name: 'Example',
            target_url: 'https://example.com',
            timestamp: '2026-09-22T12:00:00Z',
            started_at: '2026-09-22T12:00:00Z',
            is_read: false,
          },
        ]}
        onOpenIncidents={vi.fn()}
      />,
    );
    expect(screen.getByRole('button', { name: 'View incidents' })).toBeInTheDocument();
  });

  it('supports the default incident navigation callback', () => {
    render(<Header isLoggedIn onLogin={vi.fn()} onLogout={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'View incidents' }));
  });

  it('toggles and persists dark mode', () => {
    render(<Header isLoggedIn={false} onLogin={vi.fn()} onLogout={vi.fn()} />);
    const toggle = screen.getByRole('button', { name: 'Switch to dark mode' });
    fireEvent.click(toggle);
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(localStorage.getItem('uptimeTheme')).toBe('dark');
    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
  });

  it('opens the login popup and calls onLogin with the token on save', async () => {
    const onLogin = vi.fn().mockResolvedValue(true);
    render(<Header isLoggedIn={false} onLogin={onLogin} onLogout={vi.fn()} />);

    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    const input = screen.getByLabelText('API Token');
    fireEvent.change(input, { target: { value: 'sekrit' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(onLogin).toHaveBeenCalledWith('sekrit'));
  });

  it('shows a warning when the token is wrong', async () => {
    const onLogin = vi.fn().mockResolvedValue(false);
    render(<Header isLoggedIn={false} onLogin={onLogin} onLogout={vi.fn()} />);

    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    fireEvent.change(screen.getByLabelText('API Token'), { target: { value: 'nope' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('The token is wrong. Please try again.')).toBeInTheDocument();
  });

  it('closes the login popup on successful login', async () => {
    const onLogin = vi.fn().mockResolvedValue(true);
    render(<Header isLoggedIn={false} onLogin={onLogin} onLogout={vi.fn()} />);

    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    fireEvent.change(screen.getByLabelText('API Token'), { target: { value: 'sekrit' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByLabelText('API Token')).not.toBeInTheDocument());
  });

  it('closes the login popup when clicking the overlay', () => {
    const { container } = render(
      <Header isLoggedIn={false} onLogin={vi.fn()} onLogout={vi.fn()} />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    fireEvent.click(container.querySelector('.modal-overlay')!);

    expect(screen.queryByLabelText('API Token')).not.toBeInTheDocument();
  });
});

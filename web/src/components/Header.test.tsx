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
});

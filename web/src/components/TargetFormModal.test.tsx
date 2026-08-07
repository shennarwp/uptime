import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { TargetFormModal } from './TargetFormModal';
import { describe, it, expect, vi } from 'vitest';

describe('TargetFormModal component', () => {
  const initial = { name: 'Example', url: 'https://example.com', schedule: '@every 1m' };

  it('renders name, url, and schedule fields when includeUrl is true', () => {
    render(
      <TargetFormModal
        title="Add Target"
        includeUrl
        initial={initial}
        onCancel={vi.fn()}
        onSubmit={vi.fn()}
      />,
    );

    expect(screen.getByRole('textbox', { name: /^Name/ })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^URL/ })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^Schedule/ })).toBeInTheDocument();
  });

  it('omits the url field when includeUrl is false', () => {
    render(
      <TargetFormModal
        title="Edit Target"
        initial={initial}
        onCancel={vi.fn()}
        onSubmit={vi.fn()}
      />,
    );

    expect(screen.getByRole('textbox', { name: /^Name/ })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /^Schedule/ })).toBeInTheDocument();
    expect(screen.queryByRole('textbox', { name: /^URL/ })).not.toBeInTheDocument();
  });

  it('calls onSubmit with the values and closes on success', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onCancel = vi.fn();
    render(
      <TargetFormModal
        title="Add Target"
        includeUrl
        initial={initial}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSubmit).toHaveBeenCalledWith(initial);
    await waitFor(() => expect(onCancel).toHaveBeenCalled());
  });

  it('disables save when a required field is empty', () => {
    const onSubmit = vi.fn();
    render(
      <TargetFormModal
        title="Add Target"
        includeUrl
        initial={{ name: '', url: '', schedule: '' }}
        onCancel={vi.fn()}
        onSubmit={onSubmit}
      />,
    );

    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
  });

  it('calls onCancel when cancel is clicked', () => {
    const onCancel = vi.fn();
    render(
      <TargetFormModal
        title="Add Target"
        includeUrl
        initial={initial}
        onCancel={onCancel}
        onSubmit={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalled();
  });

  it('shows an error when onSubmit rejects', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new Error('failed to add'));
    render(
      <TargetFormModal
        title="Add Target"
        includeUrl
        initial={initial}
        onCancel={vi.fn()}
        onSubmit={onSubmit}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('failed to add')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeEnabled();
  });
});

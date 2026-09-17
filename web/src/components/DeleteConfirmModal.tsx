import { useState } from 'react';

export function DeleteConfirmModal({
  title,
  message,
  onCancel,
  onConfirm,
}: {
  title: string;
  message: string;
  onCancel: () => void;
  onConfirm: () => Promise<void>;
}) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    if (deleting) {
      return;
    }
    setDeleting(true);
    setError(null);
    try {
      await onConfirm();
      onCancel();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete target');
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={() => !deleting && onCancel()}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3 className="modal-title">{title}</h3>
        <p className="modal-text">{message}</p>
        <div className="modal-actions">
          {error && <span className="modal-error">{error}</span>}
          <button className="modal-btn cancel" onClick={onCancel} disabled={deleting}>
            Cancel
          </button>
          <button
            className="modal-btn delete"
            onClick={() => void handleConfirm()}
            disabled={deleting}
          >
            {deleting ? 'Deleting...' : 'Delete'}
          </button>
        </div>
      </div>
    </div>
  );
}

import { useState } from 'react';

export type TargetFormValues = {
  name: string;
  url: string;
  schedule: string;
};

export function TargetFormModal({
  title,
  initial,
  includeUrl = false,
  onCancel,
  onSubmit,
}: {
  title: string;
  initial: TargetFormValues;
  includeUrl?: boolean;
  onCancel: () => void;
  onSubmit: (values: TargetFormValues) => Promise<void>;
}) {
  const [name, setName] = useState(initial.name);
  const [url, setUrl] = useState(initial.url);
  const [schedule, setSchedule] = useState(initial.schedule);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit = name.trim() && schedule.trim() && (!includeUrl || url.trim());

  const handleSave = async () => {
    if (saving || !canSubmit) {
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSubmit({ name, url, schedule });
      onCancel();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save target');
    } finally {
      setSaving(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      void handleSave();
    }
  };

  return (
    <div className="modal-overlay" onClick={() => !saving && onCancel()}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3 className="modal-title">{title}</h3>
        <label className="modal-label">
          Name
          <span className="modal-help" tabIndex={0} aria-label="Name restrictions">
            ?
            <span className="modal-tooltip">
              Up to 100 characters. Control characters (e.g. newline) are not allowed.
            </span>
          </span>
          <input
            className="modal-input"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={100}
            onKeyDown={handleKeyDown}
          />
        </label>
        {includeUrl && (
          <label className="modal-label">
            URL
            <span className="modal-help" tabIndex={0} aria-label="URL format">
              ?
              <span className="modal-tooltip">
                Full URL including the scheme, e.g. https://example.com.
              </span>
            </span>
            <input
              className="modal-input"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="https://example.com"
            />
          </label>
        )}
        <label className="modal-label">
          Schedule
          <span className="modal-help" tabIndex={0} aria-label="Schedule format">
            ?
            <span className="modal-tooltip">
              6-field cron: second minute hour day-of-month month day-of-week — e.g. 0 0 */3 * * *.
              Descriptors are also allowed, e.g. @every 5m or @hourly.
            </span>
          </span>
          <input
            className="modal-input"
            type="text"
            value={schedule}
            onChange={(e) => setSchedule(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="e.g. 0 0 */3 * * *"
          />
        </label>
        <div className="modal-actions">
          {error && <span className="modal-error">{error}</span>}
          <button className="modal-btn cancel" onClick={onCancel} disabled={saving}>
            Cancel
          </button>
          <button
            className="modal-btn save"
            onClick={() => void handleSave()}
            disabled={saving || !canSubmit}
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}

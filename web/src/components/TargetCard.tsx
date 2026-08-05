import { useState } from 'react';
import { TruncatedUrl } from './TruncatedUrl';
import { CheckHistoryBar } from './CheckHistoryBar';

type Check = {
  id: number;
  target_id: number;
  status_code?: number;
  response_time_ms?: number;
  is_up: boolean;
  error_message?: string;
  checked_at: string;
};

type TargetWithChecks = {
  id: number;
  name: string;
  url: string;
  schedule: string;
  created_at: string;
  updated_at: string;
  checks: Check[];
};

export function TargetCard({
  target,
  isHighlighted,
  canEdit,
  onUpdate,
}: {
  target: TargetWithChecks;
  isHighlighted: boolean;
  canEdit: boolean;
  onUpdate: (id: number, name: string, schedule: string) => Promise<void>;
}) {
  const lastCheck = target.checks && target.checks.length > 0 ? target.checks[0] : null;
  const isUp = lastCheck ? lastCheck.is_up : false;
  const lastCheckTime = lastCheck
    ? new Date(lastCheck.checked_at).toLocaleString()
    : 'No checks yet';

  const [showEdit, setShowEdit] = useState(false);
  const [name, setName] = useState(target.name);
  const [schedule, setSchedule] = useState(target.schedule);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  let cardClass = 'target-card';
  if (lastCheck) {
    cardClass += isUp ? ' up' : ' down';
  }
  if (isHighlighted) {
    cardClass += ' highlight';
  }

  let badgeClass = 'status-badge';
  let badgeText = 'UNKNOWN';
  if (lastCheck) {
    badgeClass += isUp ? ' up' : ' down';
    badgeText = isUp ? 'UP' : 'DOWN';
  } else {
    badgeClass += ' unknown';
  }

  const openEdit = () => {
    setName(target.name);
    setSchedule(target.schedule);
    setError(null);
    setShowEdit(true);
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await onUpdate(target.id, name, schedule);
      setShowEdit(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update target');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div id={`target-${target.id}`} className={cardClass}>
      <div className="target-header">
        <div className="target-title-row">
          <h3 className="target-title">{target.name}</h3>
          {canEdit && (
            <button className="edit-btn" onClick={openEdit} aria-label={`Edit ${target.name}`}>
              Edit
            </button>
          )}
        </div>
        <TruncatedUrl url={target.url} />
      </div>

      <div className="target-meta">
        <div>
          <span className="meta-label">Status: </span>
          <span className={badgeClass}>{badgeText}</span>
        </div>
        <div>
          <span className="meta-label">Last Check: </span>
          <span className="meta-value">{lastCheckTime}</span>
        </div>
        {lastCheck?.response_time_ms !== undefined && lastCheck?.response_time_ms !== null && (
          <div>
            <span className="meta-label">Response Time: </span>
            <span className="meta-value">{lastCheck.response_time_ms}ms</span>
          </div>
        )}
      </div>

      <CheckHistoryBar checks={target.checks} />

      {showEdit && (
        <div className="modal-overlay" onClick={() => !saving && setShowEdit(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="modal-title">Edit Target</h3>
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
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !saving && name.trim() && schedule.trim()) {
                    handleSave();
                  }
                }}
              />
            </label>
            <label className="modal-label">
              Schedule
              <span className="modal-help" tabIndex={0} aria-label="Schedule format">
                ?
                <span className="modal-tooltip">
                  6-field cron: second minute hour day-of-month month day-of-week — e.g. 0 0 */3 * *
                  *. Descriptors are also allowed, e.g. @every 5m or @hourly.
                </span>
              </span>
              <input
                className="modal-input"
                type="text"
                value={schedule}
                onChange={(e) => setSchedule(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !saving && name.trim() && schedule.trim()) {
                    handleSave();
                  }
                }}
                placeholder="e.g. 0 0 */3 * * *"
              />
            </label>
            <div className="modal-actions">
              {error && <span className="modal-error">{error}</span>}
              <button
                className="modal-btn cancel"
                onClick={() => setShowEdit(false)}
                disabled={saving}
              >
                Cancel
              </button>
              <button
                className="modal-btn save"
                onClick={handleSave}
                disabled={saving || !name.trim() || !schedule.trim()}
              >
                {saving ? 'Saving...' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

import { useState } from 'react';
import { TruncatedUrl } from './TruncatedUrl';
import { CheckHistoryBar } from './CheckHistoryBar';
import { TargetFormModal } from './TargetFormModal';
import { formatDateTime } from '../utils/datetime';
import { formatResponseTime } from '../utils/format';

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
  cert_expires_at?: string;
  created_at: string;
  updated_at: string;
  checks: Check[];
};

function certExpiryInfo(certExpiry: Date): { label: string; level: 'ok' | 'warn' | 'critical' } {
  const daysLeft = Math.ceil((certExpiry.getTime() - Date.now()) / 86400000);
  if (daysLeft < 0) {
    return {
      label: `expired ${-daysLeft}d ago`,
      level: 'critical',
    };
  }
  return {
    label: `${daysLeft}d left`,
    level: daysLeft <= 10 ? 'critical' : daysLeft <= 30 ? 'warn' : 'ok',
  };
}

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
    ? formatDateTime(new Date(lastCheck.checked_at))
    : 'No checks yet';

  const [showEdit, setShowEdit] = useState(false);

  const certExpiry = target.cert_expires_at ? new Date(target.cert_expires_at) : null;
  const certInfo = certExpiry ? certExpiryInfo(certExpiry) : null;

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
    setShowEdit(true);
  };

  const handleUpdate = async (values: { name: string; url: string; schedule: string }) => {
    await onUpdate(target.id, values.name, values.schedule);
  };

  return (
    <div id={`target-${target.id}`} className={cardClass}>
      <div className="target-header">
        <div className="target-title-row">
          <h3 className="target-title">{target.name}</h3>
          {canEdit && (
            <button className="edit-btn" onClick={openEdit} aria-label={`Edit ${target.name}`}>
              <img className="edit-icon" src="/edit-icon.svg" alt="" />
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
            <span className="meta-value">{formatResponseTime(lastCheck.response_time_ms)}</span>
          </div>
        )}
        {certInfo && (
          <div>
            <span className="meta-label">Cert Expires: </span>
            <span
              className={`meta-value${certInfo.level !== 'ok' ? ` cert-${certInfo.level}` : ''}`}
            >
              {certInfo.label}
            </span>
          </div>
        )}
      </div>

      <CheckHistoryBar checks={target.checks} />

      {showEdit && (
        <TargetFormModal
          title="Edit Target"
          initial={{ name: target.name, url: target.url, schedule: target.schedule }}
          onCancel={() => setShowEdit(false)}
          onSubmit={handleUpdate}
        />
      )}
    </div>
  );
}

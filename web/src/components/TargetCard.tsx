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
}: {
  target: TargetWithChecks;
  isHighlighted: boolean;
}) {
  const lastCheck = target.checks && target.checks.length > 0 ? target.checks[0] : null;
  const isUp = lastCheck ? lastCheck.is_up : false;
  const lastCheckTime = lastCheck
    ? new Date(lastCheck.checked_at).toLocaleString()
    : 'No checks yet';

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

  return (
    <div id={`target-${target.id}`} className={cardClass}>
      <div className="target-header">
        <h3 className="target-title">{target.name}</h3>
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
    </div>
  );
}

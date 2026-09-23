import type { Incident } from './IncidentBell';
import { formatDateTime } from '../utils/datetime';

const labels: Record<string, string> = {
  going_down: 'Target went down',
  going_up: 'Target is back up',
  cert_30_days: 'Certificate has 30 days left',
  cert_10_days: 'Certificate has 10 days left',
  cert_expired: 'Certificate expired',
  dns_error: 'DNS lookup failed',
  timeout_error: 'Target check timed out',
  connection_refused: 'Connection refused',
  tls_error: 'TLS connection failed',
  network_error: 'Network error',
};

export function IncidentsPage({
  incidents,
  onBack,
  onMarkRead,
  onMarkAllRead,
}: {
  incidents: Incident[];
  onBack: () => void;
  onMarkRead: (id: number) => Promise<void>;
  onMarkAllRead: () => Promise<void>;
}) {
  const unreadCount = incidents.filter((incident) => !incident.is_read).length;

  return (
    <main className="incidents-page">
      <div className="incidents-page-header">
        <div>
          <button className="incidents-back-button" onClick={onBack}>
            ← Back to targets
          </button>
          <h1>Incidents</h1>
        </div>
        <button
          className="incident-mark-all"
          onClick={() => void onMarkAllRead()}
          disabled={unreadCount === 0}
        >
          Mark all as read
        </button>
      </div>
      <div className="incident-page-list">
        {incidents.length === 0 ? (
          <p className="incident-empty">No incidents yet.</p>
        ) : (
          incidents.map((incident) => (
            <article
              className={`incident-page-item${incident.is_read ? '' : ' unread'}`}
              key={incident.id}
            >
              <div className="incident-item-content">
                <strong>{labels[incident.type] ?? incident.type}</strong>
                <a href={incident.target_url} target="_blank" rel="noreferrer">
                  {incident.target_name}
                </a>
                <time dateTime={incident.timestamp}>
                  {formatDateTime(new Date(incident.timestamp))}
                </time>
                {incident.cause && <span>{incident.cause}</span>}
              </div>
              {!incident.is_read && (
                <button
                  className="incident-read-button"
                  onClick={() => void onMarkRead(incident.id)}
                  aria-label={`Mark ${labels[incident.type] ?? 'incident'} as read`}
                >
                  ✓
                </button>
              )}
            </article>
          ))
        )}
      </div>
    </main>
  );
}

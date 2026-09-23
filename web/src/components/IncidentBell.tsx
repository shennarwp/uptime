import { useEffect, useRef, useState } from 'react';

export type Incident = {
  id: number;
  type: string;
  target_name: string;
  target_url: string;
  timestamp: string;
  started_at: string;
  is_read: boolean;
  cause?: string;
};

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

function formatTimestamp(value: string) {
  return new Date(value.replace(' ', 'T') + (value.endsWith('Z') ? '' : 'Z')).toLocaleString();
}

export function IncidentBell({
  incidents,
  onMarkRead,
  onMarkAllRead,
}: {
  incidents: Incident[];
  onMarkRead: (id: number) => Promise<void>;
  onMarkAllRead: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const unreadCount = incidents.filter((incident) => !incident.is_read).length;

  useEffect(() => {
    const closeOnOutsideClick = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', closeOnOutsideClick);
    return () => document.removeEventListener('mousedown', closeOnOutsideClick);
  }, []);

  return (
    <div className="incident-bell" ref={ref}>
      <button
        className="header-btn incident-bell-button"
        onClick={() => setOpen((current) => !current)}
        aria-label="Incidents"
        aria-expanded={open}
      >
        <span aria-hidden="true">🔔</span>
        {unreadCount > 0 && (
          <span className="incident-unread-dot" aria-label={`${unreadCount} unread incidents`} />
        )}
      </button>
      {open && (
        <div className="incident-panel" role="dialog" aria-label="Incidents">
          <div className="incident-panel-header">
            <h2>Incidents</h2>
            <button
              className="incident-mark-all"
              onClick={() => void onMarkAllRead()}
              disabled={unreadCount === 0}
            >
              Mark all as read
            </button>
          </div>
          <div className="incident-list">
            {incidents.length === 0 ? (
              <p className="incident-empty">No incidents yet.</p>
            ) : (
              incidents.map((incident) => (
                <div
                  className={`incident-item${incident.is_read ? '' : ' unread'}`}
                  key={incident.id}
                >
                  <div className="incident-item-content">
                    <strong>{labels[incident.type] ?? incident.type}</strong>
                    <a href={incident.target_url} target="_blank" rel="noreferrer">
                      {incident.target_name}
                    </a>
                    <time dateTime={incident.timestamp}>{formatTimestamp(incident.timestamp)}</time>
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
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}

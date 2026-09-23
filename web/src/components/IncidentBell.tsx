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

export function IncidentBell({
  incidents,
  onOpenIncidents,
}: {
  incidents: Incident[];
  onOpenIncidents: () => void;
}) {
  const unreadCount = incidents.filter((incident) => !incident.is_read).length;

  return (
    <button
      className="header-btn incident-bell-button"
      onClick={onOpenIncidents}
      aria-label="View incidents"
    >
      <span aria-hidden="true">🔔</span>
      {unreadCount > 0 && (
        <span className="incident-unread-dot" aria-label={`${unreadCount} unread incidents`} />
      )}
    </button>
  );
}

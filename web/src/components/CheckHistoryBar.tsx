import { useEffect, useState, useRef } from 'react';

type Check = {
  id: number;
  target_id: number;
  status_code?: number;
  response_time_ms?: number;
  is_up: boolean;
  error_message?: string;
  checked_at: string;
};

export function CheckHistoryBar({ checks }: { checks: Check[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [maxBoxes, setMaxBoxes] = useState(30);

  useEffect(() => {
    if (!containerRef.current) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const width = entry.contentRect.width;
        const count = Math.max(1, Math.floor(width / 13));
        setMaxBoxes(count);
      }
    });
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, []);

  const relevantChecks = [...checks].slice(0, maxBoxes);
  const chronologicalChecks = relevantChecks.reverse();
  const paddingCount = Math.max(0, maxBoxes - chronologicalChecks.length);
  const placeholders = Array.from({ length: paddingCount }, (_, i) => ({
    id: `placeholder-${i}`,
    isEmpty: true,
  }));
  const items = [...placeholders, ...chronologicalChecks];

  return (
    <div ref={containerRef} className="history-container">
      <div className="history-title">Recent Checks History (Oldest → Newest):</div>
      <div className="history-bar">
        {items.map((item) => {
          if ('isEmpty' in item) {
            return <div key={item.id} title="No check recorded" className="history-box empty" />;
          }
          const c = item as Check;
          return (
            <div
              key={c.id}
              title={`Check at ${new Date(c.checked_at).toLocaleString()} - ${c.is_up ? 'UP' : 'DOWN'}${c.status_code ? ` (Status: ${c.status_code})` : ''}`}
              className={`history-box ${c.is_up ? 'up' : 'down'}`}
            />
          );
        })}
      </div>
    </div>
  );
}

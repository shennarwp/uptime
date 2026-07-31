import { useEffect, useState } from 'react';

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

function App() {
  const [targets, setTargets] = useState<TargetWithChecks[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  useEffect(() => {
    fetch('/api/targets')
      .then((res) => res.json())
      .then((data) => {
        setTargets(data);
        if (data.length > 0 && selectedId === null) {
          setSelectedId(data[0].id);
        }
      });
  }, []);

  return (
    <div
      style={{
        fontFamily: '"EB Garamond", Garamond, serif',
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        margin: 0,
        backgroundColor: '#FFFFFF',
      }}
    >
      {/* Header */}
      <header
        style={{
          backgroundColor: '#000000',
          color: '#FFFFFF',
          padding: '16px 24px',
          fontSize: '22px',
          fontWeight: 'bold',
        }}
      >
        Uptime
      </header>

      {/* Body Layout: Sidebar (30%) & Main (70%) */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Left Sidebar (30%) */}
        <aside
          style={{
            width: '30%',
            backgroundColor: '#F9F9F9',
            borderRight: '1px solid #e5e7eb',
            padding: '16px',
            overflowY: 'auto',
          }}
        >
          <h2
            style={{ fontSize: '18px', fontWeight: '600', marginBottom: '12px', color: '#374151' }}
          >
            Targets
          </h2>
          <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
            {targets.map((t) => (
              <li
                key={t.id}
                onClick={() => setSelectedId(t.id)}
                style={{
                  padding: '10px 12px',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  marginBottom: '6px',
                  backgroundColor: selectedId === t.id ? '#eaeaea' : 'transparent',
                  color: selectedId === t.id ? '#000000' : '#374151',
                  fontWeight: selectedId === t.id ? 'bold' : 'normal',
                  border: selectedId === t.id ? '1px solid #d1d5db' : '1px solid transparent',
                }}
              >
                {t.name}
              </li>
            ))}
          </ul>
        </aside>

        {/* Main Content (70%) */}
        <main style={{ width: '70%', backgroundColor: '#FFF', padding: '24px', overflowY: 'auto' }}>
          {targets.length === 0 ? (
            <p style={{ color: '#6b7280' }}>No targets configured.</p>
          ) : (
            targets.map((t) => {
              const lastCheck = t.checks && t.checks.length > 0 ? t.checks[0] : null;
              const isUp = lastCheck ? lastCheck.is_up : false;
              const lastCheckTime = lastCheck
                ? new Date(lastCheck.checked_at).toLocaleString()
                : 'No checks yet';

              return (
                <div
                  key={t.id}
                  style={{
                    backgroundColor: '#FFF',
                    borderRadius: '8px',
                    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
                    padding: '20px',
                    marginBottom: '20px',
                    border: '1px solid #e5e7eb',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      marginBottom: '12px',
                    }}
                  >
                    <h3 style={{ margin: 0, fontSize: '20px', color: '#111827' }}>{t.name}</h3>
                    <a
                      href={t.url}
                      target="_blank"
                      rel="noreferrer"
                      style={{ color: '#000000', textDecoration: 'underline', fontSize: '15px' }}
                    >
                      {t.url} ↗
                    </a>
                  </div>

                  {/* Status & Last Check Time */}
                  <div
                    style={{
                      display: 'flex',
                      gap: '16px',
                      marginBottom: '16px',
                      alignItems: 'center',
                      fontSize: '15px',
                    }}
                  >
                    <div>
                      <span style={{ fontWeight: 'bold', color: '#4b5563' }}>Status: </span>
                      <span
                        style={{
                          padding: '2px 8px',
                          borderRadius: '12px',
                          fontSize: '13px',
                          fontWeight: 'bold',
                          backgroundColor: lastCheck ? (isUp ? '#dcfce7' : '#fee2e2') : '#f3f4f6',
                          color: lastCheck ? (isUp ? '#166534' : '#991b1b') : '#4b5563',
                        }}
                      >
                        {lastCheck ? (isUp ? 'UP' : 'DOWN') : 'UNKNOWN'}
                      </span>
                    </div>
                    <div>
                      <span style={{ fontWeight: 'bold', color: '#4b5563' }}>Last Check: </span>
                      <span style={{ color: '#1f2937' }}>{lastCheckTime}</span>
                    </div>
                    {lastCheck?.response_time_ms !== undefined &&
                      lastCheck?.response_time_ms !== null && (
                        <div>
                          <span style={{ fontWeight: 'bold', color: '#4b5563' }}>
                            Response Time:{' '}
                          </span>
                          <span style={{ color: '#1f2937' }}>{lastCheck.response_time_ms}ms</span>
                        </div>
                      )}
                  </div>

                  {/* Small individual boxes indicating checks */}
                  <div>
                    <div
                      style={{
                        fontSize: '13px',
                        fontWeight: 'bold',
                        color: '#6b7280',
                        marginBottom: '6px',
                      }}
                    >
                      Recent Checks History (Oldest → Newest):
                    </div>
                    <div
                      style={{
                        display: 'flex',
                        gap: '3px',
                        alignItems: 'center',
                        flexWrap: 'wrap',
                      }}
                    >
                      {t.checks && t.checks.length > 0 ? (
                        [...t.checks].reverse().map((c) => (
                          <div
                            key={c.id}
                            title={`Check at ${new Date(c.checked_at).toLocaleString()} - ${c.is_up ? 'UP' : 'DOWN'}${c.status_code ? ` (Status: ${c.status_code})` : ''}`}
                            style={{
                              width: '10px',
                              height: '20px',
                              borderRadius: '2px',
                              backgroundColor: c.is_up ? '#22c55e' : '#ef4444',
                              cursor: 'pointer',
                            }}
                          />
                        ))
                      ) : (
                        <span style={{ fontSize: '13px', color: '#9ca3af' }}>
                          No check history available
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </main>
      </div>
    </div>
  );
}

export default App;

import { useCallback, useEffect, useRef, useState } from 'react';
import './App.css';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { TargetCard } from './components/TargetCard';
import { TargetFormModal, type TargetFormValues } from './components/TargetFormModal';
import type { Incident } from './components/IncidentBell';
import { IncidentsPage } from './components/IncidentsPage';

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

const DEFAULT_CHECKS_LIMIT = 300;
const MAX_CHECKS_LIMIT = 500;
const CHECK_WIDTH_PX = 8;
const CHECKS_BUFFER = 1.25;
const RESIZE_FETCH_THRESHOLD = 16;

async function fetchTargets(checksLimit: number): Promise<TargetWithChecks[]> {
  const query = checksLimit === DEFAULT_CHECKS_LIMIT ? '' : `?checks_limit=${checksLimit}`;
  const response = await fetch(`/api/v1/targets${query}`);
  if (!response.ok) {
    throw new Error(`Failed to load targets (${response.status})`);
  }
  return response.json();
}

function App() {
  const [page, setPage] = useState<'targets' | 'incidents'>(() =>
    window.location.pathname === '/incidents' ? 'incidents' : 'targets',
  );
  const [targets, setTargets] = useState<TargetWithChecks[]>([]);
  const checksLimitRef = useRef(DEFAULT_CHECKS_LIMIT);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [highlightedId, setHighlightedId] = useState<number | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [isLoggedIn, setIsLoggedIn] = useState<boolean>(
    () => localStorage.getItem('uptimeApiToken') !== null,
  );

  useEffect(() => {
    const loadTargets = () => {
      fetchTargets(checksLimitRef.current).then((data) => {
        setTargets(data);
        setSelectedId((current) => current ?? (data.length > 0 ? data[0].id : null));
      });
    };

    loadTargets();
    const events = 'EventSource' in window ? new EventSource('/api/v1/events') : null;
    if (events) {
      events.onmessage = loadTargets;
    }
    const interval = setInterval(loadTargets, 30_000);
    return () => {
      events?.close();
      clearInterval(interval);
    };
  }, []);

  const handleHistoryWidthChange = useCallback((width: number) => {
    const visibleChecks = Math.max(1, Math.floor(width / CHECK_WIDTH_PX));
    const nextLimit = Math.min(
      MAX_CHECKS_LIMIT,
      Math.max(visibleChecks, Math.ceil(visibleChecks * CHECKS_BUFFER)),
    );
    if (Math.abs(nextLimit - checksLimitRef.current) < RESIZE_FETCH_THRESHOLD) {
      return;
    }

    checksLimitRef.current = nextLimit;
    fetchTargets(nextLimit).then((data) => {
      setTargets(data);
      setSelectedId((current) => current ?? (data.length > 0 ? data[0].id : null));
    });
  }, []);

  useEffect(() => {
    const handlePopState = () => {
      setPage(window.location.pathname === '/incidents' ? 'incidents' : 'targets');
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  useEffect(() => {
    const loadIncidents = () => {
      fetch('/api/v1/incidents')
        .then((res) => res.json())
        .then((data) => setIncidents(data));
    };
    loadIncidents();
    const interval = setInterval(loadIncidents, 30_000);
    return () => clearInterval(interval);
  }, []);

  const handleSelect = (id: number) => {
    setSelectedId(id);
    setHighlightedId(id);
    const el = document.getElementById(`target-${id}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
    setTimeout(() => {
      setHighlightedId((prev) => (prev === id ? null : prev));
    }, 1000);
  };

  const handleLogin = async (token: string): Promise<boolean> => {
    const res = await fetch('/api/v1/auth/verify', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
      return false;
    }
    localStorage.setItem('uptimeApiToken', token);
    setIsLoggedIn(true);
    return true;
  };

  const handleLogout = () => {
    localStorage.removeItem('uptimeApiToken');
    setIsLoggedIn(false);
  };

  const openIncidents = () => {
    window.history.pushState({}, '', '/incidents');
    setPage('incidents');
  };

  const showTargets = () => {
    window.history.pushState({}, '', '/');
    setPage('targets');
  };

  const markIncidentRead = async (id: number) => {
    const token = localStorage.getItem('uptimeApiToken') ?? '';
    const res = await fetch(`/api/v1/incident/${id}/read`, {
      method: 'PATCH',
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      setIncidents((current) =>
        current.map((incident) => (incident.id === id ? { ...incident, is_read: true } : incident)),
      );
    }
  };

  const markAllIncidentsRead = async () => {
    const token = localStorage.getItem('uptimeApiToken') ?? '';
    const res = await fetch('/api/v1/incidents/read', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      setIncidents((current) => current.map((incident) => ({ ...incident, is_read: true })));
    }
  };

  const handleUpdateTarget = async (id: number, name: string, schedule: string) => {
    const token = localStorage.getItem('uptimeApiToken') ?? '';
    const res = await fetch(`/api/v1/target/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ name, schedule }),
    });
    if (!res.ok) {
      if (res.status === 401) {
        localStorage.removeItem('uptimeApiToken');
        setIsLoggedIn(false);
      }
      const body = await res.text();
      throw new Error(body || `Failed to update target (${res.status})`);
    }
    const data = await fetchTargets(checksLimitRef.current);
    setTargets(data);
  };

  const handleCreateTarget = async (values: TargetFormValues) => {
    const token = localStorage.getItem('uptimeApiToken') ?? '';
    const res = await fetch('/api/v1/targets', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ name: values.name, url: values.url, schedule: values.schedule }),
    });
    if (!res.ok) {
      if (res.status === 401) {
        localStorage.removeItem('uptimeApiToken');
        setIsLoggedIn(false);
      }
      const body = await res.text();
      throw new Error(body || `Failed to add target (${res.status})`);
    }
    const data = await fetchTargets(checksLimitRef.current);
    setTargets(data);
  };

  const handleDeleteTarget = async (id: number) => {
    const token = localStorage.getItem('uptimeApiToken') ?? '';
    const res = await fetch(`/api/v1/target/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
      if (res.status === 401) {
        localStorage.removeItem('uptimeApiToken');
        setIsLoggedIn(false);
      }
      const body = await res.text();
      throw new Error(body || `Failed to delete target (${res.status})`);
    }
    setSelectedId((current) => (current === id ? null : current));
    const data = await fetchTargets(checksLimitRef.current);
    setTargets(data);
    if (data.length > 0) {
      setSelectedId((current) => current ?? data[0].id);
    }
  };

  return (
    <div className="app-container">
      <Header
        isLoggedIn={isLoggedIn}
        onLogin={handleLogin}
        onLogout={handleLogout}
        incidents={incidents}
        onOpenIncidents={openIncidents}
      />
      {page === 'incidents' ? (
        <IncidentsPage
          incidents={incidents}
          onBack={showTargets}
          onMarkRead={markIncidentRead}
          onMarkAllRead={markAllIncidentsRead}
        />
      ) : (
        <div className="app-body">
          <Sidebar targets={targets} selectedId={selectedId} onSelect={handleSelect} />
          <main className="app-main">
            {targets.length === 0 ? (
              <p>No targets configured.</p>
            ) : (
              targets.map((t) => (
                <TargetCard
                  key={t.id}
                  target={t}
                  isHighlighted={highlightedId === t.id}
                  canEdit={isLoggedIn}
                  onUpdate={handleUpdateTarget}
                  onDelete={handleDeleteTarget}
                  onHistoryWidthChange={handleHistoryWidthChange}
                />
              ))
            )}
            {isLoggedIn && (
              <button
                className="add-target-card"
                onClick={() => setShowAdd(true)}
                aria-label="Add new target"
              >
                +
              </button>
            )}
          </main>
        </div>
      )}
      {showAdd && (
        <TargetFormModal
          title="Add Target"
          includeUrl
          initial={{ name: '', url: '', schedule: '' }}
          onCancel={() => setShowAdd(false)}
          onSubmit={handleCreateTarget}
        />
      )}
    </div>
  );
}

export default App;

import { useEffect, useState } from 'react';
import './App.css';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { TargetCard } from './components/TargetCard';

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
  const [highlightedId, setHighlightedId] = useState<number | null>(null);

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

  const handleUpdateTarget = async (id: number, name: string, schedule: string) => {
    const res = await fetch(`/api/target/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, schedule }),
    });
    if (!res.ok) {
      throw new Error(`Failed to update target: ${res.status}`);
    }
    const data = await fetch('/api/targets').then((r) => r.json());
    setTargets(data);
  };

  return (
    <div className="app-container">
      <Header />
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
                onUpdate={handleUpdateTarget}
              />
            ))
          )}
        </main>
      </div>
    </div>
  );
}

export default App;

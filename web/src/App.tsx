import { useEffect, useState } from 'react';

type Target = {
  id: number;
  name: string;
  url: string;
  schedule: string;
};

function App() {
  const [targets, setTargets] = useState<Target[]>([]);

  useEffect(() => {
    fetch('/api/targets')
      .then((res) => res.json())
      .then(setTargets);
  }, []);

  return (
    <div>
      <h1>Uptime</h1>
      {targets.length === 0 ? (
        <p>No targets configured.</p>
      ) : (
        <ul>
          {targets.map((t) => (
            <li key={t.id}>
              <strong>{t.name}</strong> –{' '}
              <a href={t.url} target="_blank" rel="noreferrer">
                {t.url}
              </a>{' '}
              (schedule: {t.schedule})
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export default App;

import { useState } from 'react';

type Target = {
  id: number;
  name: string;
};

type SidebarProps = {
  targets: Target[];
  selectedId: number | null;
  onSelect: (id: number) => void;
};

export function Sidebar({ targets, selectedId, onSelect }: SidebarProps) {
  const [isOpen, setIsOpen] = useState(false);

  const toggleOpen = () => {
    setIsOpen(!isOpen);
  };

  return (
    <aside className="app-sidebar">
      <div
        className="sidebar-header"
        onClick={toggleOpen}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            toggleOpen();
          }
        }}
        role="button"
        tabIndex={0}
        aria-expanded={isOpen}
      >
        <h2 className="sidebar-title">Targets</h2>
        <span className="sidebar-indicator">{isOpen ? '▲' : '▼'}</span>
      </div>
      <ul className={`sidebar-list ${isOpen ? 'open' : ''}`}>
        {(Array.isArray(targets) ? targets : []).map((t) => (
          <li
            key={t.id}
            onClick={() => onSelect(t.id)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                onSelect(t.id);
              }
            }}
            tabIndex={0}
            role="button"
            aria-current={selectedId === t.id ? 'true' : undefined}
            className={`sidebar-item ${selectedId === t.id ? 'active' : ''}`}
          >
            {t.name}
          </li>
        ))}
      </ul>
    </aside>
  );
}

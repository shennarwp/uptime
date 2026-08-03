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
      <div className="sidebar-header" onClick={toggleOpen}>
        <h2 className="sidebar-title">Targets</h2>
        <span className="sidebar-indicator">{isOpen ? '▲' : '▼'}</span>
      </div>
      <ul className={`sidebar-list ${isOpen ? 'open' : ''}`}>
        {(Array.isArray(targets) ? targets : []).map((t) => (
          <li
            key={t.id}
            onClick={() => onSelect(t.id)}
            className={`sidebar-item ${selectedId === t.id ? 'active' : ''}`}
          >
            {t.name}
          </li>
        ))}
      </ul>
    </aside>
  );
}

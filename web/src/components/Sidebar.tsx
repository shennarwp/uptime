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
  return (
    <aside className="app-sidebar">
      <h2 className="sidebar-title">Targets</h2>
      <ul className="sidebar-list">
        {targets.map((t) => (
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

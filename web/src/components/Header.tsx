import { useState } from 'react';

export function Header({
  isLoggedIn,
  onLogin,
  onLogout,
}: {
  isLoggedIn: boolean;
  onLogin: (token: string) => Promise<boolean>;
  onLogout: () => void;
}) {
  const [showLogin, setShowLogin] = useState(false);
  const [token, setToken] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const ok = await onLogin(token);
      if (ok) {
        setShowLogin(false);
        setToken('');
      } else {
        setError('The token is wrong. Please try again.');
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <header className="app-header">
      <span className="app-title">Uptime</span>
      <div className="header-actions">
        {isLoggedIn ? (
          <button className="header-btn" onClick={onLogout}>
            Logout
          </button>
        ) : (
          <button className="header-btn" onClick={() => setShowLogin(true)}>
            Login
          </button>
        )}
      </div>

      {showLogin && (
        <div className="modal-overlay" onClick={() => !saving && setShowLogin(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3 className="modal-title">Login</h3>
            <label className="modal-label">
              API Token
              <input
                className="modal-input"
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !saving && token.trim()) {
                    handleSave();
                  }
                }}
                placeholder="Enter your API token"
                autoFocus
              />
            </label>
            {error && <p className="modal-error-text">{error}</p>}
            <div className="modal-actions">
              <button
                className="modal-btn cancel"
                onClick={() => setShowLogin(false)}
                disabled={saving}
              >
                Cancel
              </button>
              <button
                className="modal-btn save"
                onClick={handleSave}
                disabled={saving || !token.trim()}
              >
                {saving ? 'Checking...' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}
    </header>
  );
}

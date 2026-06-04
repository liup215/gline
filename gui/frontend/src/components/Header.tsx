import { THEME } from '../theme';
import { AppStatus } from '../types';

interface HeaderProps {
  sidebarOpen: boolean;
  setSidebarOpen: (v: boolean) => void;
  activeTaskID: string | null;
  status: AppStatus;
  isLoading: boolean;
  onStop: () => void;
}

export function Header({ sidebarOpen, setSidebarOpen, activeTaskID, status, isLoading, onStop }: HeaderProps) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 24px', borderBottom: `1px solid ${THEME.border}`, background: THEME.bg }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <button
          onClick={() => setSidebarOpen(!sidebarOpen)}
          style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.2rem', padding: '4px', display: 'flex', alignItems: 'center' }}
          title="Toggle sidebar"
        >
          ☰
        </button>
        <span style={{ fontSize: '0.9rem', color: THEME.textMuted }}>
          {activeTaskID ? 'Conversation' : 'New Chat'}
        </span>
        <span
          title={`Working directory: ${status.cwd}`}
          style={{ fontSize: '0.75rem', color: THEME.textDim, maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          📁 {status.cwd}
        </span>
      </div>
      <div style={{ display: 'flex', gap: '8px' }}>
        {isLoading && (
          <button
            onClick={onStop}
            style={{ padding: '6px 14px', borderRadius: '6px', border: `1px solid ${THEME.toastError}`, background: 'transparent', color: THEME.toastError, cursor: 'pointer', fontSize: '0.8rem' }}
          >
            ⏹ Stop
          </button>
        )}
      </div>
    </div>
  );
}

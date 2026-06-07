import { THEME } from '../theme';

interface SidebarProps {
  sidebarOpen: boolean;
  history: any[];
  activeTaskID: string | null;
  onNewChat: () => void;
  onSelectTask: (id: string) => void;
  onDeleteTask: (e: React.MouseEvent, id: string) => void;
  onOpenSettings: () => void;
}

export function Sidebar({ sidebarOpen, history, activeTaskID, onNewChat, onSelectTask, onDeleteTask, onOpenSettings }: SidebarProps) {
  return (
    <div style={{
      width: sidebarOpen ? '280px' : '0px',
      minWidth: sidebarOpen ? '280px' : '0px',
      background: THEME.bgSidebar,
      borderRight: `1px solid ${THEME.border}`,
      display: 'flex',
      flexDirection: 'column',
      transition: 'width 0.25s ease, min-width 0.25s ease',
      overflow: 'hidden',
    }}>
      <div style={{ padding: '16px 20px', borderBottom: `1px solid ${THEME.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <h1 style={{ margin: 0, fontSize: '1.1rem', fontWeight: 700, background: `linear-gradient(90deg, ${THEME.logoGradientStart}, ${THEME.logoGradientEnd})`, WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>gline</h1>
        <button
          style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.3rem', padding: '2px 6px', borderRadius: '6px' }}
          onClick={onNewChat}
          title="New conversation"
        >
          +
        </button>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '12px 10px' }}>
        <div style={{ fontSize: '0.7rem', color: THEME.textDim, textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '8px', padding: '0 10px' }}>
          Recent Conversations
        </div>
        {history.length === 0 && (
          <div style={{ padding: '20px', color: THEME.textDim, fontSize: '0.85rem', textAlign: 'center' }}>
            No conversations yet
          </div>
        )}
        {history.map((task) => (
          <div
            key={task.ID}
            onClick={() => task.ID && onSelectTask(task.ID)}
            style={{
              padding: '10px 14px',
              borderRadius: '8px',
              marginBottom: '4px',
              cursor: 'pointer',
              background: activeTaskID === task.ID ? THEME.optionBg : 'transparent',
              color: activeTaskID === task.ID ? THEME.text : THEME.textMuted,
              fontSize: '0.85rem',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              transition: 'background 0.15s',
            }}
            onMouseEnter={e => { if (activeTaskID !== task.ID) e.currentTarget.style.background = THEME.optionHoverBg; }}
            onMouseLeave={e => { if (activeTaskID !== task.ID) e.currentTarget.style.background = 'transparent'; }}
          >
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
              {task.Title || 'Untitled conversation'}
            </span>
            <button
              onClick={(e) => task.ID && onDeleteTask(e, task.ID)}
              style={{ background: 'transparent', border: 'none', color: THEME.textDim, cursor: 'pointer', fontSize: '0.75rem', padding: '2px 4px', opacity: 0, transition: 'opacity 0.15s' }}
              onMouseEnter={e => e.currentTarget.style.opacity = '1'}
            >
              🗑
            </button>
          </div>
        ))}
      </div>

      {/* Sidebar Footer */}
      <div style={{ padding: '12px 20px', borderTop: `1px solid ${THEME.border}` }}>
        <button
          onClick={onOpenSettings}
          style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.85rem', display: 'flex', alignItems: 'center', gap: '8px', transition: 'background 0.15s' }}
          onMouseEnter={e => e.currentTarget.style.background = THEME.optionBg}
          onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
        >
          ⚙️ Settings
        </button>
      </div>
    </div>
  );
}

import { THEME } from '../theme';
import type { FileEntry } from '../hooks/useFileReference';

interface FilePickerProps {
  active: boolean;
  entries: FileEntry[];
  currentPath: string;
  loading: boolean;
  selectedIndex: number;
  onSelect: (entry: FileEntry) => void;
  onNavigate: (path: string) => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function FilePicker({
  active, entries, currentPath, loading, selectedIndex, onSelect, onNavigate,
}: FilePickerProps) {
  if (!active) return null;

  return (
    <div style={{
      position: 'absolute',
      bottom: '100%',
      left: 0,
      right: 0,
      marginBottom: '4px',
      background: '#1a2332',
      border: `1px solid ${THEME.border}`,
      borderRadius: '10px',
      maxHeight: '280px',
      overflowY: 'auto',
      boxShadow: '0 -4px 24px rgba(0,0,0,0.4)',
      zIndex: 100,
    }}>
      {/* Path breadcrumb */}
      <div style={{
        padding: '8px 14px',
        borderBottom: `1px solid ${THEME.border}`,
        fontSize: '0.75rem',
        color: THEME.textDim,
        display: 'flex',
        alignItems: 'center',
        gap: '4px',
      }}>
        📁 {currentPath ? `/${currentPath}` : '/'}
      </div>

      {loading ? (
        <div style={{ padding: '20px', textAlign: 'center', color: THEME.textDim, fontSize: '0.85rem' }}>
          ⏳ Loading...
        </div>
      ) : entries.length === 0 ? (
        <div style={{ padding: '20px', textAlign: 'center', color: THEME.textDim, fontSize: '0.85rem' }}>
          📭 Empty directory
        </div>
      ) : (
        entries.map((entry, idx) => (
          <div
            key={entry.path}
            onClick={() => onSelect(entry)}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '7px 14px',
              cursor: 'pointer',
              background: idx === selectedIndex ? 'rgba(59,130,246,0.15)' : 'transparent',
              color: idx === selectedIndex ? THEME.text : THEME.textMuted,
              transition: 'background 0.1s',
              fontSize: '0.85rem',
            }}
            onMouseEnter={() => {}}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0 }}>
              <span style={{ fontSize: '1rem', flexShrink: 0 }}>
                {entry.isDir ? '📁' : '📄'}
              </span>
              <span style={{
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                color: entry.isDir ? THEME.text : THEME.textMuted,
                fontWeight: entry.isDir ? 500 : 400,
              }}>
                {entry.name}
              </span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexShrink: 0 }}>
              {!entry.isDir && (
                <span style={{ fontSize: '0.72rem', color: THEME.textDim }}>
                  {formatSize(entry.size)}
                </span>
              )}
              {entry.isDir && (
                <span style={{ fontSize: '0.8rem', color: THEME.textDim }}>→</span>
              )}
            </div>
          </div>
        ))
      )}

      {/* Hint */}
      <div style={{
        padding: '6px 14px',
        borderTop: `1px solid ${THEME.border}`,
        fontSize: '0.7rem',
        color: THEME.textDim,
        display: 'flex',
        justifyContent: 'space-between',
      }}>
        <span>↑↓ navigate · Enter select · Esc close</span>
        <span>Backspace go up</span>
      </div>
    </div>
  );
}

import { useRef, useEffect, useMemo } from 'react';
import { THEME } from '../theme';
import type { FileEntry } from '../hooks/useFileReference';

interface FilePickerProps {
  active: boolean;
  entries: FileEntry[];
  currentPath: string;
  loading: boolean;
  selectedIndex: number;
  query: string;
  onSelect: (entry: FileEntry) => void;
  onNavigate: (path: string) => void;
  onQueryChange: (query: string) => void;
  onClose: () => void;
  onKeyDown?: (e: React.KeyboardEvent) => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function getFilteredEntries(entries: FileEntry[], query: string): FileEntry[] {
  if (!query) return entries;
  const q = query.toLowerCase();
  return entries.filter(e => e.name.toLowerCase().includes(q));
}

export function FilePicker({
  active, entries, currentPath, loading, selectedIndex, query,
  onSelect, onNavigate, onQueryChange, onClose, onKeyDown,
}: FilePickerProps) {
  const itemRefs = useRef<(HTMLDivElement | null)[]>([]);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const filtered = useMemo(
    () => getFilteredEntries(entries, query),
    [entries, query]
  );

  // Auto-scroll selected item into view
  useEffect(() => {
    if (active && filtered.length > 0 && selectedIndex >= 0 && selectedIndex < filtered.length) {
      const el = itemRefs.current[selectedIndex];
      if (el) {
        el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    }
  }, [active, selectedIndex, filtered.length]);

  // Auto-focus filter input when picker opens / dir changes
  useEffect(() => {
    if (active) {
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [active, currentPath]);

  if (!active) return null;

  const handleSelect = (entry: FileEntry) => {
    onSelect(entry);
    if (!entry.isDir) {
      onClose();
    }
  };

  const handleFilterKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Forward navigation keys to parent key handler (for selectedIndex updates)
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      onKeyDown?.(e);
      return;
    }
    if (e.key === 'Backspace' && query === '') {
      e.preventDefault();
      e.stopPropagation();
      onKeyDown?.(e);
      return;
    }
  };

  const showEmpty = !loading && filtered.length === 0 && entries.length > 0;
  const showNoItems = !loading && entries.length === 0;

  return (
    <div
      style={{
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
      }}
    >
      {/* Path breadcrumb */}
      <div
        style={{
          padding: '8px 14px',
          borderBottom: `1px solid ${THEME.border}`,
          fontSize: '0.75rem',
          color: THEME.textDim,
          display: 'flex',
          alignItems: 'center',
          gap: '4px',
        }}
      >
        📁 {currentPath ? `/${currentPath}` : '/'}
      </div>

      {/* Filter input */}
      <div style={{ padding: '6px 14px', borderBottom: `1px solid ${THEME.border}` }}>
        <input
          ref={inputRef}
          type="text"
          placeholder="Filter files..."
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          onKeyDown={handleFilterKeyDown}
          style={{
            width: '100%',
            padding: '5px 10px',
            borderRadius: '6px',
            border: `1px solid ${THEME.border}`,
            background: '#0f172a',
            color: THEME.text,
            fontSize: '0.82rem',
            outline: 'none',
            boxSizing: 'border-box',
          }}
        />
      </div>

      {loading ? (
        <div
          style={{
            padding: '20px',
            textAlign: 'center',
            color: THEME.textDim,
            fontSize: '0.85rem',
          }}
        >
          ⏳ Loading...
        </div>
      ) : showNoItems ? (
        <div
          style={{
            padding: '20px',
            textAlign: 'center',
            color: THEME.textDim,
            fontSize: '0.85rem',
          }}
        >
          📭 Empty directory
        </div>
      ) : showEmpty ? (
        <div
          style={{
            padding: '20px',
            textAlign: 'center',
            color: THEME.textDim,
            fontSize: '0.85rem',
          }}
        >
          🔍 No files match “{query}”
        </div>
      ) : (
        filtered.map((entry, idx) => (
          <div
            key={entry.path}
            ref={(el) => {
              itemRefs.current[idx] = el;
            }}
            onClick={() => handleSelect(entry)}
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
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0 }}>
              <span style={{ fontSize: '1rem', flexShrink: 0 }}>
                {entry.isDir ? '📁' : '📄'}
              </span>
              <span
                style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  color: entry.isDir ? THEME.text : THEME.textMuted,
                  fontWeight: entry.isDir ? 500 : 400,
                }}
              >
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
      <div
        style={{
          padding: '6px 14px',
          borderTop: `1px solid ${THEME.border}`,
          fontSize: '0.7rem',
          color: THEME.textDim,
          display: 'flex',
          justifyContent: 'space-between',
        }}
      >
        <span>↑↓ navigate · Enter select · Esc close</span>
        <span>Backspace go up</span>
      </div>
    </div>
  );
}

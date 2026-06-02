import React from 'react';
import type { SlashCommand } from './use-slash-commands';

interface SlashMenuProps {
  active: boolean;
  filtered: SlashCommand[];
  selectedIndex: number;
  onSelect: (cmd: SlashCommand) => void;
  inputRef: React.RefObject<HTMLInputElement | null>;
}

const THEME = {
  bg: '#0b1220',
  bgSidebar: '#0f172a',
  border: '#1e293b',
  accent: '#3b82f6',
  accentHover: '#2563eb',
  text: '#e2e8f0',
  textMuted: '#94a3b8',
  textDim: '#64748b',
  codeBg: '#1e1e2e',
};

export function SlashMenu({ active, filtered, selectedIndex, onSelect, inputRef }: SlashMenuProps) {
  if (!active || filtered.length === 0) return null;

  return (
    <div
      style={{
        position: 'absolute',
        bottom: '100%',
        left: 0,
        right: 0,
        marginBottom: '8px',
        background: THEME.bgSidebar,
        border: `1px solid ${THEME.border}`,
        borderRadius: '10px',
        overflow: 'hidden',
        maxHeight: '280px',
        overflowY: 'auto',
        boxShadow: '0 10px 40px rgba(0,0,0,0.4)',
        zIndex: 50,
      }}
    >
      {filtered.map((cmd, idx) => {
        const isSelected = idx === selectedIndex;
        return (
          <div
            key={cmd.name}
            onClick={() => onSelect(cmd)}
            onMouseEnter={() => {
              // Optional: highlight on hover (would require lifting hover state)
            }}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              padding: '10px 16px',
              cursor: 'pointer',
              borderBottom: `1px solid ${idx < filtered.length - 1 ? THEME.border : 'transparent'}`,
              background: isSelected ? 'rgba(59, 130, 246, 0.15)' : 'transparent',
              transition: 'background 0.1s',
            }}
          >
            <span
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: '26px',
                height: '26px',
                borderRadius: '6px',
                background: cmd.section === 'custom' ? 'rgba(168,85,247,0.15)' : 'rgba(59,130,246,0.15)',
                color: cmd.section === 'custom' ? '#c084fc' : '#60a5fa',
                fontSize: '0.75rem',
                fontWeight: 600,
                flexShrink: 0,
              }}
            >
              /
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: '0.9rem',
                  fontWeight: 500,
                  color: isSelected ? '#fff' : THEME.text,
                }}
              >
                {cmd.name}
              </div>
              <div
                style={{
                  fontSize: '0.78rem',
                  color: THEME.textMuted,
                  marginTop: '2px',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                {cmd.description}
              </div>
            </div>
            <span
              style={{
                fontSize: '0.7rem',
                color: THEME.textDim,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                flexShrink: 0,
              }}
            >
              {cmd.section}
            </span>
          </div>
        );
      })}
    </div>
  );
}

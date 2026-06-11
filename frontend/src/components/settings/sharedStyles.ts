import { THEME } from '../../theme';

export const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 14px',
  borderRadius: '8px',
  border: `1px solid ${THEME.border}`,
  background: THEME.inputBg,
  color: THEME.text,
  fontSize: '0.9rem',
  outline: 'none',
  boxSizing: 'border-box',
};

export const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '0.85rem',
  color: THEME.textMuted,
  marginBottom: '6px',
  fontWeight: 500,
};

export const selectStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 14px',
  borderRadius: '8px',
  border: `1px solid ${THEME.border}`,
  background: THEME.inputBg,
  color: THEME.text,
  fontSize: '0.9rem',
  outline: 'none',
  cursor: 'pointer',
  boxSizing: 'border-box',
};

import { useState } from 'react';
import { THEME } from '../theme';
import { formatContent } from '../utils/format';

const overlayStyle: React.CSSProperties = {
  position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
  background: THEME.overlayBg, backdropFilter: 'blur(4px)',
  display: 'flex', alignItems: 'center', justifyContent: 'center',
  zIndex: 1000,
};
const panelStyle: React.CSSProperties = {
  width: '440px', maxWidth: '90vw',
  background: THEME.bgSidebar,
  border: `1px solid ${THEME.border}`,
  borderRadius: '14px',
  padding: '24px 28px',
  color: THEME.text,
  boxShadow: '0 20px 60px rgba(0,0,0,0.4)',
};
const optionBtnStyle: React.CSSProperties = {
  width: '100%', padding: '10px 14px', borderRadius: '8px',
  border: `1px solid ${THEME.border}`,
  background: THEME.optionBg, color: THEME.text,
  cursor: 'pointer', fontSize: '0.9rem', textAlign: 'left',
  transition: 'background 0.15s',
};

export function FollowupModal({ question, options, onAnswer }: { question: string; options: string[]; onAnswer: (ans: string) => void }) {
  const [customMode, setCustomMode] = useState(false);
  const [customValue, setCustomValue] = useState('');

  return (
    <div style={overlayStyle}>
      <div style={panelStyle}>
        <div style={{ fontSize: '0.72rem', color: THEME.accentHover, marginBottom: '10px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>Question</div>
        <div
          className="md-rendered"
          style={{
            fontSize: '1rem',
            lineHeight: 1.5,
            marginBottom: '20px',
            maxHeight: '40vh',
            overflow: 'auto',
            paddingRight: '4px',
          }}
          dangerouslySetInnerHTML={{ __html: formatContent(question) }}
        />
        {customMode ? (
          <form onSubmit={(e) => { e.preventDefault(); onAnswer(customValue); }} style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <textarea
              autoFocus
              rows={3}
              value={customValue}
              onChange={e => setCustomValue(e.target.value)}
              placeholder="Type your own answer..."
              style={{ padding: '10px 12px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: THEME.inputBg, color: THEME.text, fontSize: '0.9rem', resize: 'vertical', fontFamily: 'inherit' }}
            />
            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button type="button" onClick={() => setCustomMode(false)} style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.85rem' }}>Back</button>
              <button type="submit" style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: THEME.accent, color: THEME.userTextColor, cursor: 'pointer', fontSize: '0.85rem' }}>Send</button>
            </div>
          </form>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {options.map((opt, i) => (
              <button
                key={i}
                onClick={() => onAnswer(opt)}
                style={optionBtnStyle}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = THEME.optionHoverBg; }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = THEME.optionBg; }}
              >
                {opt}
              </button>
            ))}
            {options.length > 0 && (
              <button
                onClick={() => setCustomMode(true)}
                style={optionBtnStyle}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = THEME.optionHoverBg; }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = THEME.optionBg; }}
              >
                ✎ 其他（自行输入）
              </button>
            )}
            {options.length === 0 && (
              <form onSubmit={(e) => { e.preventDefault(); const el = (e.target as HTMLFormElement).elements.namedItem('answer') as HTMLInputElement; onAnswer(el.value); }} style={{ display: 'flex', gap: '8px' }}>
                <input name="answer" autoFocus placeholder="Type your answer..." style={{ flex: 1, padding: '10px 12px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: THEME.inputBg, color: THEME.text, fontSize: '0.9rem' }} />
                <button type="submit" style={{ padding: '10px 18px', borderRadius: '8px', border: 'none', background: THEME.accent, color: THEME.userTextColor, cursor: 'pointer', fontSize: '0.9rem' }}>Send</button>
              </form>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

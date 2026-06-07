import { THEME } from '../theme';
import { getToolHint, parseToolInput } from '../utils/format';

interface ToolMessageProps {
  toolName: string | undefined;
  toolInput: string | undefined;
  toolResult: string | undefined;
}

export function ToolMessage({ toolName, toolInput, toolResult }: ToolMessageProps) {
  const done = !!toolResult;
  const hint = getToolHint(toolName || '', toolInput);
  const isQuestion = toolName === 'ask_followup_question' && hint.startsWith('💬');

  if (isQuestion) {
    const input = parseToolInput(toolInput);
    return (
      <div style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '12px' }}>
        <div style={{ maxWidth: '75%', background: THEME.optionBg, border: `1px solid ${THEME.linkColor}4d`, borderRadius: '14px 14px 14px 4px', padding: '12px 16px', lineHeight: 1.5, fontSize: '0.92rem', color: THEME.text }}>
          <div style={{ fontSize: '0.72rem', color: THEME.accentHover, marginBottom: '6px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>Question</div>
          <div>{hint.slice(2).trim()}</div>
          {input.options && input.options.length > 0 && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', marginTop: '10px' }}>
              {input.options.map((opt: string, oi: number) => (
                <span key={oi} style={{ padding: '4px 10px', borderRadius: '6px', background: THEME.optionBg, color: THEME.linkColor, fontSize: '0.82rem', border: `1px solid ${THEME.linkColor}40` }}>{opt}</span>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '8px' }}>
      <div
        title={toolInput || ''}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '8px',
          padding: '4px 12px',
          borderRadius: '20px',
          background: done ? THEME.statusSuccessBg : THEME.statusPendingBg,
          border: `1px solid ${done ? THEME.statusSuccessBorder : THEME.statusPendingBorder}`,
          color: done ? THEME.statusSuccessText : THEME.statusPendingText,
          fontSize: '0.78rem',
          fontFamily: '"SFMono-Regular", Consolas, monospace',
          lineHeight: 1.4,
          maxWidth: '70%',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        <span style={{ fontSize: '0.85em', flexShrink: 0 }}>{done ? '✓' : '◌'}</span>
        <span>{hint}</span>
      </div>
    </div>
  );
}

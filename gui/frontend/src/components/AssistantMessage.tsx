import { THEME } from '../theme';
import { formatContent } from '../utils/format';

interface AssistantMessageProps {
  content: string;
  streaming: boolean | undefined;
  isLast: boolean;
}

export function AssistantMessage({ content, streaming, isLast }: AssistantMessageProps) {
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '16px' }}>
      <div style={{ maxWidth: '85%', background: THEME.assistantBg, color: THEME.text, padding: '14px 20px', borderRadius: '18px 18px 18px 4px', lineHeight: 1.6, fontSize: '0.95rem', border: `1px solid ${THEME.border}`, boxShadow: '0 2px 8px rgba(0,0,0,0.15)' }}>
        <div className="md-rendered" dangerouslySetInnerHTML={{ __html: formatContent(content) }} />
        {streaming && isLast && <span style={{ color: THEME.accent, animation: 'blink 1s infinite' }}>▌</span>}
      </div>
    </div>
  );
}

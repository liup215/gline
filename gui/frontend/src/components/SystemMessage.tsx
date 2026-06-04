import { THEME } from '../theme';
import { formatContent } from '../utils/format';

export function SystemMessage({ content }: { content: string }) {
  // Render special interactive tool results with distinct styles
  const isCompletion = content.startsWith('📋');
  const isQuestion = content.startsWith('💬');

  const colors = {
    completion: {
      dark: { bg: 'rgba(34,197,94,0.15)', border: 'rgba(74,222,128,0.4)', text: '#4ade80' },
      light: { bg: 'rgba(34,197,94,0.1)', border: 'rgba(34,197,94,0.3)', text: '#166534' },
    },
    question: {
      dark: { bg: 'rgba(59,130,246,0.15)', border: 'rgba(96,165,250,0.4)', text: '#93c5fd' },
      light: { bg: 'rgba(59,130,246,0.1)', border: 'rgba(59,130,246,0.3)', text: '#1d4ed8' },
    },
    default: {
      dark: { bg: 'rgba(251,191,36,0.15)', border: 'rgba(251,191,36,0.4)', text: '#fbbf24' },
      light: { bg: 'rgba(245,158,11,0.1)', border: 'rgba(245,158,11,0.3)', text: '#92400e' },
    },
  };

  const themeKey = (THEME.bg === '#f8fafc' ? 'light' : 'dark') as 'dark' | 'light';

  const style = isCompletion
    ? colors.completion[themeKey]
    : isQuestion
      ? colors.question[themeKey]
      : colors.default[themeKey];

  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '0 24px', marginBottom: '12px' }}>
      <div
        className="md-rendered"
        style={{
          maxWidth: '80%',
          padding: '14px 18px',
          borderRadius: '8px',
          fontSize: '0.82rem',
          lineHeight: 1.5,
          background: style.bg,
          border: `1px solid ${style.border}`,
          color: style.text,
        }}
        dangerouslySetInnerHTML={{ __html: formatContent(content) }}
      />
    </div>
  );
}

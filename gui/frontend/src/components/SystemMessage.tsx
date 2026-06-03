import { formatContent } from '../utils/format';

export function SystemMessage({ content }: { content: string }) {
  // Render special interactive tool results with distinct styles
  const isCompletion = content.startsWith('📋');
  const isQuestion = content.startsWith('💬');

  const style = isCompletion
    ? { background: '#064e3b', border: '1px solid rgba(74,222,128,0.25)' }
    : isQuestion
      ? { background: '#1e3a8a', border: '1px solid rgba(96,165,250,0.25)' }
      : { background: '#451a03', border: '1px solid rgba(251,191,36,0.25)' };

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
          ...style,
        }}
        dangerouslySetInnerHTML={{ __html: formatContent(content) }}
      />
    </div>
  );
}

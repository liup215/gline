import { formatContent } from '../utils/format';

export function SystemMessage({ content }: { content: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '0 24px', marginBottom: '12px' }}>
      <div
        className="md-rendered"
        style={{
          maxWidth: '80%',
          background: '#451a03',
          padding: '14px 18px',
          borderRadius: '8px',
          fontSize: '0.82rem',
          lineHeight: 1.5,
          border: '1px solid rgba(251,191,36,0.25)',
        }}
        dangerouslySetInnerHTML={{ __html: formatContent(content) }}
      />
    </div>
  );
}

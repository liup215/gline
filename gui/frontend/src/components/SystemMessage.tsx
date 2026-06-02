export function SystemMessage({ content }: { content: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '0 24px', marginBottom: '12px' }}>
      <div style={{
        maxWidth: '80%',
        background: '#451a03',
        color: '#fbbf24',
        padding: '14px 18px',
        borderRadius: '8px',
        fontSize: '0.82rem',
        lineHeight: 1.6,
        whiteSpace: 'pre-wrap',
        fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
      }}>
        {content}
      </div>
    </div>
  );
}

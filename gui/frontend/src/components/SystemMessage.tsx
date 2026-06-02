export function SystemMessage({ content }: { content: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '0 24px', marginBottom: '12px' }}>
      <div style={{ maxWidth: '80%', background: '#451a03', color: '#fbbf24', padding: '10px 16px', borderRadius: '8px', fontSize: '0.85rem', textAlign: 'center' }}>
        {content}
      </div>
    </div>
  );
}

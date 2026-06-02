function isTitleLine(line: string): boolean {
  return line.endsWith(':') && !line.includes('  ') && line.length < 30;
}

function renderHelpLines(content: string) {
  return content.split('\n').map((raw, i) => {
    const line = raw.trimRight();
    const trimmed = line.trim();
    if (!trimmed) return null;

    if (isTitleLine(trimmed)) {
      return (
        <div key={i} style={{
          fontWeight: 700,
          fontSize: '0.88rem',
          color: '#fbbf24',
          marginTop: 8,
          marginBottom: 6,
          borderBottom: '1px solid rgba(251,191,36,0.3)',
          paddingBottom: 2,
          letterSpacing: '0.02em',
        }}>
          {trimmed}
        </div>
      );
    }

    // Try to split "  /command   description..." or "  Ctrl+N     description..."
    const match = trimmed.match(/^(\/\w+|Ctrl\+\w+)\s+(.*)$/);
    if (match) {
      return (
        <div key={i} style={{ display: 'flex', gap: 8, marginLeft: 6, marginBottom: 3, alignItems: 'flex-start' }}>
          <span style={{ color: '#a78bfa', marginTop: 2 }}>•</span>
          <span style={{ color: '#c084fc', fontWeight: 600, minWidth: 90 }}>{match[1]}</span>
          <span style={{ color: '#fb923c' }}>{match[2]}</span>
        </div>
      );
    }

    return (
      <div key={i} style={{ color: '#fcd34d', marginLeft: 16, marginBottom: 2 }}>
        {trimmed}
      </div>
    );
  }).filter(Boolean);
}

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
        lineHeight: 1.5,
        fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
      }}>
        {renderHelpLines(content)}
      </div>
    </div>
  );
}

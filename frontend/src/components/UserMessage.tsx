import { THEME } from '../theme';

export function UserMessage({ content }: { content: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '0 24px', marginBottom: '16px' }}>
      <div style={{ maxWidth: '70%', background: THEME.userBg, color: THEME.userTextColor, padding: '12px 18px', borderRadius: '18px 18px 4px 18px', lineHeight: 1.5, fontSize: '0.95rem', boxShadow: '0 2px 8px rgba(0,0,0,0.2)' }}>
        {content}
      </div>
    </div>
  );
}

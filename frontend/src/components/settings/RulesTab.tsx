import { THEME } from '../../theme';
import type { RuleInfo } from '../../hooks/useSettings';

interface RulesTabProps {
  rules: RuleInfo[];
  rulesMessage: string;
  loadingRules: boolean;
  onReloadRules: () => Promise<void>;
  formatFileSize: (bytes: number) => string;
  formatModTime: (ts: number) => string;
}

export function RulesTab({
  rules,
  rulesMessage,
  loadingRules,
  onReloadRules,
  formatFileSize,
  formatModTime,
}: RulesTabProps) {
  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '12px' }}>
        <button
          onClick={onReloadRules}
          disabled={loadingRules}
          style={{
            padding: '5px 12px',
            borderRadius: '6px',
            border: `1px solid ${THEME.border}`,
            background: 'transparent',
            color: THEME.textMuted,
            cursor: loadingRules ? 'not-allowed' : 'pointer',
            fontSize: '0.8rem',
            display: 'flex',
            alignItems: 'center',
            gap: '4px',
          }}
        >
          {loadingRules ? '⏳' : '🔄'} Reload
        </button>
      </div>

      {rulesMessage && (
        <div style={{
          padding: '8px 12px',
          borderRadius: '6px',
          background: rulesMessage.includes('✅') ? THEME.toastSuccessBg : THEME.toastErrorBg,
          color: rulesMessage.includes('✅') ? THEME.toastSuccess : THEME.toastError,
          fontSize: '0.8rem',
          marginBottom: '10px',
        }}>
          {rulesMessage}
        </div>
      )}

      {loadingRules ? (
        <div style={{ textAlign: 'center', padding: '20px', color: THEME.textDim, fontSize: '0.85rem' }}>
          ⏳ Loading rules...
        </div>
      ) : rules.length === 0 ? (
        <div style={{
          padding: '16px 14px',
          borderRadius: '8px',
          background: THEME.bgChat,
          color: THEME.textDim,
          fontSize: '0.85rem',
          textAlign: 'center',
          border: `1px solid ${THEME.border}`,
        }}>
          <div style={{ marginBottom: '6px' }}>📭 No custom rules found</div>
          <div style={{ fontSize: '0.75rem' }}>
            Create <code style={{ background: THEME.inputBg, padding: '2px 6px', borderRadius: '4px' }}>.clinerules</code> in your workspace
            or <code style={{ background: THEME.inputBg, padding: '2px 6px', borderRadius: '4px' }}>~/.gline/clinerules</code> globally.
          </div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {rules.map((rule, idx) => (
            <div
              key={idx}
              style={{
                padding: '10px 14px',
                borderRadius: '8px',
                background: THEME.inputBg,
                border: `1px solid ${THEME.border}`,
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
              }}
            >
              <span style={{ fontSize: '1.1rem' }}>📄</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{
                  fontSize: '0.85rem',
                  fontWeight: 500,
                  color: THEME.text,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}>
                  {rule.name}
                </div>
                <div style={{
                  fontSize: '0.72rem',
                  color: THEME.textDim,
                  display: 'flex',
                  gap: '8px',
                  marginTop: '2px',
                }}>
                  <span>{rule.source === 'global' ? '🌍 global' : '📁 workspace'}</span>
                  <span>·</span>
                  <span>{formatFileSize(rule.size)}</span>
                  <span>·</span>
                  <span>{formatModTime(rule.modTime)}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      <div style={{ fontSize: '0.75rem', color: THEME.textDim, marginTop: '12px', lineHeight: 1.5 }}>
        Rules are injected into the system prompt and guide the AI&apos;s behavior.
        Changes take effect on the next message after reloading.
      </div>
    </>
  );
}

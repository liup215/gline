import { useState, useEffect } from 'react';
import { THEME } from '../theme';
import { useTheme } from '../ThemeContext';
import type { RuleInfo } from '../hooks/useSettings';

type TabKey = 'provider' | 'general' | 'rules';

interface TabDef {
  key: TabKey;
  label: string;
  icon: string;
}

const TABS: TabDef[] = [
  { key: 'provider', label: 'Provider', icon: '🔗' },
  { key: 'general', label: 'General', icon: '🎨' },
  { key: 'rules', label: 'Rules', icon: '📋' },
];

interface SettingsPanelProps {
  config: any;
  onClose: () => void;
  onSave: (updates: Record<string, string>) => void;
  saveMessage: string;
  rules: RuleInfo[];
  rulesMessage: string;
  loadingRules: boolean;
  onLoadRules: () => Promise<void>;
  onReloadRules: () => Promise<void>;
  formatFileSize: (bytes: number) => string;
  formatModTime: (ts: number) => string;
}

export function SettingsPanel({
  config,
  onClose,
  onSave,
  saveMessage,
  rules,
  rulesMessage,
  loadingRules,
  onLoadRules,
  onReloadRules,
  formatFileSize,
  formatModTime,
}: SettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<TabKey>('provider');

  const [provider, setProvider] = useState(config?.Provider?.Default || 'anthropic');
  const [anthropicKey, setAnthropicKey] = useState(config?.Provider?.Anthropic?.APIKey || '');
  const [anthropicModel, setAnthropicModel] = useState(config?.Provider?.Anthropic?.Model || 'claude-3-sonnet');
  const [anthropicMaxTokens, setAnthropicMaxTokens] = useState(String(config?.Provider?.Anthropic?.MaxContextTokens || '0'));
  const [openaiKey, setOpenaiKey] = useState(config?.Provider?.OpenAI?.APIKey || '');
  const [openaiModel, setOpenaiModel] = useState(config?.Provider?.OpenAI?.Model || 'gpt-4');
  const [openaiBaseURL, setOpenaiBaseURL] = useState(config?.Provider?.OpenAI?.BaseURL || '');
  const [openaiMaxTokens, setOpenaiMaxTokens] = useState(String(config?.Provider?.OpenAI?.MaxContextTokens || '0'));
  const [uiTheme, setUiTheme] = useState(config?.UI?.Theme || 'default');

  useEffect(() => {
    if (config) {
      setProvider(config.Provider?.Default || 'anthropic');
      setAnthropicKey(config.Provider?.Anthropic?.APIKey || '');
      setAnthropicModel(config.Provider?.Anthropic?.Model || 'claude-3-sonnet');
      setAnthropicMaxTokens(String(config.Provider?.Anthropic?.MaxContextTokens || '0'));
      setOpenaiKey(config.Provider?.OpenAI?.APIKey || '');
      setOpenaiModel(config.Provider?.OpenAI?.Model || 'gpt-4');
      setOpenaiBaseURL(config.Provider?.OpenAI?.BaseURL || '');
      setOpenaiMaxTokens(String(config.Provider?.OpenAI?.MaxContextTokens || '0'));
      setUiTheme(config.UI?.Theme || 'default');
    }
  }, [config]);

  const handleSave = () => {
    const updates: Record<string, string> = {};
    updates['provider.default'] = provider;
    updates['provider.anthropic.api_key'] = anthropicKey;
    updates['provider.anthropic.model'] = anthropicModel;
    updates['provider.anthropic.max_context_tokens'] = anthropicMaxTokens;
    updates['provider.openai.api_key'] = openaiKey;
    updates['provider.openai.model'] = openaiModel;
    updates['provider.openai.base_url'] = openaiBaseURL;
    updates['provider.openai.max_context_tokens'] = openaiMaxTokens;
    updates['ui.theme'] = uiTheme;
    onSave(updates);
  };

  /* ── shared styles ── */
  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${THEME.border}`, background: THEME.inputBg,
    color: THEME.text, fontSize: '0.9rem', outline: 'none', boxSizing: 'border-box',
  };
  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: '0.85rem', color: THEME.textMuted,
    marginBottom: '6px', fontWeight: 500,
  };
  const selectStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${THEME.border}`, background: THEME.inputBg,
    color: THEME.text, fontSize: '0.9rem', outline: 'none',
    cursor: 'pointer', boxSizing: 'border-box',
  };

  /* ── tab content ── */
  const renderProviderTab = () => (
    <>
      <div style={{ marginBottom: '18px' }}>
        <label style={labelStyle}>Default Provider</label>
        <select style={selectStyle} value={provider} onChange={e => setProvider(e.target.value)}>
          <option value="anthropic">Anthropic (Claude)</option>
          <option value="openai">OpenAI / Compatible</option>
        </select>
      </div>

      {provider === 'anthropic' && (
        <>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Anthropic API Key</label>
            <input type="password" style={inputStyle} value={anthropicKey} onChange={e => setAnthropicKey(e.target.value)} placeholder="sk-ant-api03-..." />
          </div>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Model</label>
            <select style={selectStyle} value={anthropicModel} onChange={e => setAnthropicModel(e.target.value)}>
              <option value="claude-3-opus">Claude 3 Opus</option>
              <option value="claude-3-sonnet">Claude 3 Sonnet</option>
              <option value="claude-3-haiku">Claude 3 Haiku</option>
              <option value="claude-3-5-sonnet">Claude 3.5 Sonnet</option>
            </select>
          </div>
          <div style={{ marginBottom: '18px' }}>
            <label style={labelStyle}>Max Context Tokens (0 = auto ~262K)</label>
            <input type="number" style={inputStyle} value={anthropicMaxTokens} onChange={e => setAnthropicMaxTokens(e.target.value)} placeholder="262000" />
          </div>
        </>
      )}

      {provider === 'openai' && (
        <>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>API Key</label>
            <input type="password" style={inputStyle} value={openaiKey} onChange={e => setOpenaiKey(e.target.value)} placeholder="sk-..." />
          </div>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Model</label>
            <input type="text" style={inputStyle} value={openaiModel} onChange={e => setOpenaiModel(e.target.value)} placeholder="gpt-4, gpt-4-turbo..." />
          </div>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Base URL (optional)</label>
            <input type="text" style={inputStyle} value={openaiBaseURL} onChange={e => setOpenaiBaseURL(e.target.value)} placeholder="https://api.openai.com/v1" />
            <div style={{ fontSize: '0.75rem', color: THEME.textDim, marginTop: '4px' }}>Leave empty for OpenAI. For OpenRouter, use https://openrouter.ai/api/v1</div>
          </div>
          <div style={{ marginBottom: '18px' }}>
            <label style={labelStyle}>Max Context Tokens (0 = auto ~262K)</label>
            <input type="number" style={inputStyle} value={openaiMaxTokens} onChange={e => setOpenaiMaxTokens(e.target.value)} placeholder="262000" />
          </div>
        </>
      )}
    </>
  );

  const { themeName: currentTheme, setTheme } = useTheme();

  const renderGeneralTab = () => (
    <>
      {/* Chat Theme */}
      <div style={{ marginBottom: '14px' }}>
        <label style={labelStyle}>Chat Theme</label>
        <select
          style={selectStyle}
          value={currentTheme}
          onChange={e => {
            const name = e.target.value as 'dark' | 'light';
            setTheme(name);
            setUiTheme(name);
          }}
        >
          <option value="dark">Dark</option>
          <option value="light">Light</option>
        </select>
      </div>
    </>
  );

  const renderRulesTab = () => (
    <>
      {/* Reload button row */}
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

      {/* Toast message */}
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

      {/* Loading */}
      {loadingRules ? (
        <div style={{ textAlign: 'center', padding: '20px', color: THEME.textDim, fontSize: '0.85rem' }}>
          ⏳ Loading rules...
        </div>
      ) : rules.length === 0 ? (
        /* Empty state */
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
        /* Rules list */
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

      {/* Footer hint */}
      <div style={{ fontSize: '0.75rem', color: THEME.textDim, marginTop: '12px', lineHeight: 1.5 }}>
        Rules are injected into the system prompt and guide the AI&apos;s behavior.
        Changes take effect on the next message after reloading.
      </div>
    </>
  );

  const tabContent: Record<TabKey, () => JSX.Element> = {
    provider: renderProviderTab,
    general: renderGeneralTab,
    rules: renderRulesTab,
  };

  return (
    <div
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: THEME.overlayBg, backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        zIndex: 1000,
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: '520px', maxHeight: '85vh',
          background: THEME.cardBg, border: `1px solid ${THEME.border}`,
          borderRadius: '14px', padding: '24px 28px',
          color: THEME.text, boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
          display: 'flex', flexDirection: 'column',
          overflow: 'hidden',
        }}
        onClick={e => e.stopPropagation()}
      >
        {/* ── Header ── */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <h2 style={{ margin: 0, fontSize: '1.2rem', fontWeight: 600 }}>⚙️ Settings</h2>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.2rem' }}>✕</button>
        </div>

        {/* ── Save message toast ── */}
        {saveMessage && (
          <div style={{
            padding: '10px 14px', borderRadius: '8px',
            background: saveMessage.includes('success') ? THEME.toastSuccessBg : THEME.toastErrorBg,
            color: saveMessage.includes('success') ? THEME.toastSuccess : THEME.toastError,
            fontSize: '0.85rem', marginBottom: '12px',
          }}>
            {saveMessage}
          </div>
        )}

        {/* ── Tab bar ── */}
        <div style={{
          display: 'flex',
          borderBottom: `1px solid ${THEME.border}`,
          marginBottom: '16px',
          gap: '0',
        }}>
          {TABS.map(tab => {
            const isActive = activeTab === tab.key;
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                style={{
                  flex: 1,
                  padding: '10px 0',
                  background: 'transparent',
                  border: 'none',
                  borderBottom: isActive ? `2px solid ${THEME.accent}` : '2px solid transparent',
                  color: isActive ? THEME.text : THEME.textDim,
                  fontSize: '0.85rem',
                  fontWeight: isActive ? 600 : 400,
                  cursor: 'pointer',
                  transition: 'color 0.15s, border-color 0.15s',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '6px',
                }}
              >
                <span>{tab.icon}</span>
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* ── Tab content (scrollable) ── */}
        <div style={{ flex: 1, overflowY: 'auto', paddingRight: '4px' }}>
          {tabContent[activeTab]()}
        </div>

        {/* ── Footer buttons ── */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '16px', paddingTop: '12px', borderTop: `1px solid ${THEME.border}` }}>
          <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.9rem' }}>Cancel</button>
          <button onClick={handleSave} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: THEME.accent, color: THEME.userTextColor, cursor: 'pointer', fontSize: '0.9rem', fontWeight: 500 }}>Save Settings</button>
        </div>
      </div>
    </div>
  );
}

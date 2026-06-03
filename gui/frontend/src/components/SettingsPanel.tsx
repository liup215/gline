import { useState, useEffect } from 'react';
import { THEME } from '../theme';
import type { RuleInfo } from '../hooks/useSettings';

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

  const overlayStyle: React.CSSProperties = {
    position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
    background: 'rgba(0, 0, 0, 0.6)', backdropFilter: 'blur(4px)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    zIndex: 1000,
  };
  const panelStyle: React.CSSProperties = {
    width: '520px', maxHeight: '85vh', overflowY: 'auto',
    background: '#111827', border: `1px solid ${THEME.border}`,
    borderRadius: '14px', padding: '24px 28px',
    color: THEME.text, boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
  };
  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${THEME.border}`, background: '#1e293b',
    color: THEME.text, fontSize: '0.9rem', outline: 'none', boxSizing: 'border-box',
  };
  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: '0.85rem', color: THEME.textMuted,
    marginBottom: '6px', fontWeight: 500,
  };
  const selectStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${THEME.border}`, background: '#1e293b',
    color: THEME.text, fontSize: '0.9rem', outline: 'none',
    cursor: 'pointer', boxSizing: 'border-box',
  };

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={panelStyle} onClick={e => e.stopPropagation()}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <h2 style={{ margin: 0, fontSize: '1.2rem', fontWeight: 600 }}>⚙️ Settings</h2>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.2rem' }}>✕</button>
        </div>

        {saveMessage && (
          <div style={{ padding: '10px 14px', borderRadius: '8px', background: saveMessage.includes('success') ? 'rgba(34, 197, 94, 0.15)' : 'rgba(239, 68, 68, 0.15)', color: saveMessage.includes('success') ? '#4ade80' : '#f87171', fontSize: '0.85rem', marginBottom: '16px' }}>
            {saveMessage}
          </div>
        )}

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

        <div style={{ borderTop: `1px solid ${THEME.border}`, paddingTop: '18px', marginBottom: '14px' }}>
          <label style={labelStyle}>Chat Theme</label>
          <select style={selectStyle} value={uiTheme} onChange={e => setUiTheme(e.target.value)}>
            <option value="default">Default</option>
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </div>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '8px' }}>
          <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.9rem' }}>Cancel</button>
          <button onClick={handleSave} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: THEME.accent, color: '#fff', cursor: 'pointer', fontSize: '0.9rem', fontWeight: 500 }}>Save Settings</button>
        </div>
      </div>
    </div>
  );
}

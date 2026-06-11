import { useState, useEffect } from 'react';
import { THEME } from '../theme';
import type { RuleInfo } from '../hooks/useSettings';
import { ProviderTab } from './settings/ProviderTab';
import { MemoryTab } from './settings/MemoryTab';
import { GeneralTab } from './settings/GeneralTab';
import { RulesTab } from './settings/RulesTab';

type TabKey = 'provider' | 'memory' | 'general' | 'rules';

interface TabDef {
  key: TabKey;
  label: string;
  icon: string;
}

const TABS: TabDef[] = [
  { key: 'provider', label: 'Provider', icon: '🔗' },
  { key: 'memory', label: 'Memory', icon: '🧠' },
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
  onReloadRules,
  formatFileSize,
  formatModTime,
}: SettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<TabKey>('provider');

  /* ── provider state ── */
  const [provider, setProvider] = useState(config?.Provider?.Default || 'anthropic');
  const [anthropicKey, setAnthropicKey] = useState(config?.Provider?.Anthropic?.APIKey || '');
  const [anthropicModel, setAnthropicModel] = useState(config?.Provider?.Anthropic?.Model || 'claude-3-sonnet');
  const [anthropicMaxTokens, setAnthropicMaxTokens] = useState(String(config?.Provider?.Anthropic?.MaxContextTokens || '0'));
  const [openaiKey, setOpenaiKey] = useState(config?.Provider?.OpenAI?.APIKey || '');
  const [openaiModel, setOpenaiModel] = useState(config?.Provider?.OpenAI?.Model || 'gpt-4');
  const [openaiBaseURL, setOpenaiBaseURL] = useState(config?.Provider?.OpenAI?.BaseURL || '');
  const [openaiMaxTokens, setOpenaiMaxTokens] = useState(String(config?.Provider?.OpenAI?.MaxContextTokens || '0'));

  /* ── memory state ── */
  const [memEnabled, setMemEnabled] = useState(config?.Memory?.Enabled ?? true);
  const [memProvider, setMemProvider] = useState(config?.Memory?.Embedding?.Provider || 'openai');
  const [memModel, setMemModel] = useState(config?.Memory?.Embedding?.Model || 'text-embedding-3-small');
  const [memAPIKey, setMemAPIKey] = useState(config?.Memory?.Embedding?.APIKey || '');
  const [memBaseURL, setMemBaseURL] = useState(config?.Memory?.Embedding?.BaseURL || '');
  const [memTopK, setMemTopK] = useState(String(config?.Memory?.Retrieval?.TopK || '5'));
  const [memMinScore, setMemMinScore] = useState(String(config?.Memory?.Retrieval?.MinScore || '0.6'));
  const [memMaxTokens, setMemMaxTokens] = useState(String(config?.Memory?.Retrieval?.MaxTokens || '2000'));

  /* ── general state ── */
  const [uiTheme, setUiTheme] = useState(config?.UI?.Theme || 'default');

  useEffect(() => {
    if (!config) return;
    setProvider(config.Provider?.Default || 'anthropic');
    setAnthropicKey(config.Provider?.Anthropic?.APIKey || '');
    setAnthropicModel(config.Provider?.Anthropic?.Model || 'claude-3-sonnet');
    setAnthropicMaxTokens(String(config.Provider?.Anthropic?.MaxContextTokens || '0'));
    setOpenaiKey(config.Provider?.OpenAI?.APIKey || '');
    setOpenaiModel(config.Provider?.OpenAI?.Model || 'gpt-4');
    setOpenaiBaseURL(config.Provider?.OpenAI?.BaseURL || '');
    setOpenaiMaxTokens(String(config.Provider?.OpenAI?.MaxContextTokens || '0'));
    setMemEnabled(config.Memory?.Enabled ?? true);
    setMemProvider(config.Memory?.Embedding?.Provider || 'openai');
    setMemModel(config.Memory?.Embedding?.Model || 'text-embedding-3-small');
    setMemAPIKey(config.Memory?.Embedding?.APIKey || '');
    setMemBaseURL(config.Memory?.Embedding?.BaseURL || '');
    setMemTopK(String(config.Memory?.Retrieval?.TopK || '5'));
    setMemMinScore(String(config.Memory?.Retrieval?.MinScore || '0.6'));
    setMemMaxTokens(String(config.Memory?.Retrieval?.MaxTokens || '2000'));
    setUiTheme(config.UI?.Theme || 'default');
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
    updates['memory.enabled'] = memEnabled ? 'true' : 'false';
    updates['memory.embedding.provider'] = memProvider;
    updates['memory.embedding.model'] = memModel;
    updates['memory.embedding.api_key'] = memAPIKey;
    updates['memory.embedding.base_url'] = memBaseURL;
    updates['memory.retrieval.top_k'] = memTopK;
    updates['memory.retrieval.min_score'] = memMinScore;
    updates['memory.retrieval.max_tokens'] = memMaxTokens;
    onSave(updates);
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
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <h2 style={{ margin: 0, fontSize: '1.2rem', fontWeight: 600 }}>⚙️ Settings</h2>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.2rem' }}>✕</button>
        </div>

        {/* Save message toast */}
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

        {/* Tab bar */}
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

        {/* Tab content (scrollable) */}
        <div style={{ flex: 1, overflowY: 'auto', paddingRight: '4px' }}>
          {activeTab === 'provider' && (
            <ProviderTab
              provider={provider} setProvider={setProvider}
              anthropicKey={anthropicKey} setAnthropicKey={setAnthropicKey}
              anthropicModel={anthropicModel} setAnthropicModel={setAnthropicModel}
              anthropicMaxTokens={anthropicMaxTokens} setAnthropicMaxTokens={setAnthropicMaxTokens}
              openaiKey={openaiKey} setOpenaiKey={setOpenaiKey}
              openaiModel={openaiModel} setOpenaiModel={setOpenaiModel}
              openaiBaseURL={openaiBaseURL} setOpenaiBaseURL={setOpenaiBaseURL}
              openaiMaxTokens={openaiMaxTokens} setOpenaiMaxTokens={setOpenaiMaxTokens}
            />
          )}
          {activeTab === 'memory' && (
            <MemoryTab
              memEnabled={memEnabled} setMemEnabled={setMemEnabled}
              memProvider={memProvider} setMemProvider={setMemProvider}
              memModel={memModel} setMemModel={setMemModel}
              memAPIKey={memAPIKey} setMemAPIKey={setMemAPIKey}
              memBaseURL={memBaseURL} setMemBaseURL={setMemBaseURL}
              memTopK={memTopK} setMemTopK={setMemTopK}
              memMinScore={memMinScore} setMemMinScore={setMemMinScore}
              memMaxTokens={memMaxTokens} setMemMaxTokens={setMemMaxTokens}
            />
          )}
          {activeTab === 'general' && <GeneralTab uiTheme={uiTheme} setUiTheme={setUiTheme} />}
          {activeTab === 'rules' && (
            <RulesTab
              rules={rules}
              rulesMessage={rulesMessage}
              loadingRules={loadingRules}
              onReloadRules={onReloadRules}
              formatFileSize={formatFileSize}
              formatModTime={formatModTime}
            />
          )}
        </div>

        {/* Footer buttons */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '16px', paddingTop: '12px', borderTop: `1px solid ${THEME.border}` }}>
          <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.9rem' }}>Cancel</button>
          <button onClick={handleSave} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: THEME.accent, color: THEME.userTextColor, cursor: 'pointer', fontSize: '0.9rem', fontWeight: 500 }}>Save Settings</button>
        </div>
      </div>
    </div>
  );
}

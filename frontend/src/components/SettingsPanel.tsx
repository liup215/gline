import { useState, useEffect } from 'react';
import { THEME } from '../theme';
import type { RuleInfo } from '../hooks/useSettings';
import type { VersionCheckState } from '../hooks/useVersionCheck';
import { ProviderTab } from './settings/ProviderTab';
import { MemoryTab } from './settings/MemoryTab';
import { GeneralTab } from './settings/GeneralTab';
import { RulesTab } from './settings/RulesTab';
import { MCPTab } from './settings/MCPTab';
import { UpdatesTab } from './settings/UpdatesTab';

type TabKey = 'provider' | 'memory' | 'general' | 'rules' | 'mcp' | 'updates';

interface TabDef {
  key: TabKey;
  label: string;
  icon: string;
}

const TABS: TabDef[] = [
  { key: 'provider', label: 'Provider', icon: '🔗' },
  { key: 'memory', label: 'Memory', icon: '🧠' },
  { key: 'mcp', label: 'MCP', icon: '🤖' },
  { key: 'general', label: 'General', icon: '🎨' },
  { key: 'rules', label: 'Rules', icon: '📋' },
  { key: 'updates', label: 'Updates', icon: '⬆️' },
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
  versionCheck?: VersionCheckState;
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
  versionCheck,
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
    updates['memory.enabled'] = String(memEnabled);
    updates['memory.embedding.provider'] = memProvider;
    updates['memory.embedding.model'] = memModel;
    updates['memory.embedding.api_key'] = memAPIKey;
    updates['memory.embedding.base_url'] = memBaseURL;
    updates['memory.retrieval.top_k'] = memTopK;
    updates['memory.retrieval.min_score'] = memMinScore;
    updates['memory.retrieval.max_tokens'] = memMaxTokens;
    updates['ui.theme'] = uiTheme;
    onSave(updates);
  };

  return (
    <div style={{ position: 'fixed', inset: 0, background: THEME.overlayBg, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }} onClick={onClose}>
      <div style={{ background: THEME.bg, borderRadius: 12, width: 700, maxWidth: '90vw', maxHeight: '85vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.5)' }} onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px 20px', borderBottom: `1px solid ${THEME.border}` }}>
          <h2 style={{ margin: 0, fontSize: '1.25rem', fontWeight: 600 }}>Settings</h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: THEME.textMuted, fontSize: '1.25rem', cursor: 'pointer', lineHeight: 1 }}>×</button>
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', borderBottom: `1px solid ${THEME.border}`, background: THEME.bgSidebar }}>
          {TABS.map(tab => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              style={{
                padding: '12px 16px',
                background: activeTab === tab.key ? THEME.bg : 'transparent',
                border: 'none',
                borderBottom: activeTab === tab.key ? `2px solid ${THEME.accent}` : '2px solid transparent',
                color: activeTab === tab.key ? THEME.text : THEME.textMuted,
                cursor: 'pointer',
                fontSize: '0.9rem',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                transition: 'all 0.2s ease',
              }}
            >
              <span>{tab.icon}</span>
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Content */}
        <div style={{ padding: '20px', overflowY: 'auto', flex: 1 }}>
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
          {activeTab === 'mcp' && (
            <MCPTab
              servers={config?.MCP?.Servers || []}
              onSave={(servers) => {
                // Save MCP servers to config
                const updates: Record<string, string> = {};
                updates['mcp.servers'] = JSON.stringify(servers);
                onSave(updates);
              }}
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
          {activeTab === 'updates' && versionCheck && (
            <UpdatesTab
              isChecking={versionCheck.isChecking}
              updateResult={versionCheck.updateResult}
              error={versionCheck.error}
              onCheckForUpdates={versionCheck.checkForUpdates}
              onOpenReleasePage={versionCheck.openReleasePage}
            />
          )}
        </div>

        {/* Footer buttons */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '16px', padding: '12px 20px', borderTop: `1px solid ${THEME.border}` }}>
          {activeTab !== 'updates' && (
            <>
              <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.9rem' }}>Cancel</button>
              <button onClick={handleSave} style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: THEME.accent, color: THEME.userTextColor, cursor: 'pointer', fontSize: '0.9rem', fontWeight: 500 }}>Save Settings</button>
            </>
          )}
          {activeTab === 'updates' && (
            <button onClick={onClose} style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.9rem' }}>Close</button>
          )}
        </div>
        
        {/* Save message */}
        {saveMessage && (
          <div style={{ padding: '8px 20px', background: saveMessage.includes('Failed') ? THEME.toastErrorBg : THEME.toastSuccessBg, color: saveMessage.includes('Failed') ? THEME.toastError : THEME.toastSuccess, fontSize: '0.85rem', textAlign: 'center' }}>
            {saveMessage}
          </div>
        )}
      </div>
    </div>
  );
}

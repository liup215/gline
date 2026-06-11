import { inputStyle, labelStyle, selectStyle } from './sharedStyles';

interface ProviderTabProps {
  provider: string;
  setProvider: (v: string) => void;
  anthropicKey: string;
  setAnthropicKey: (v: string) => void;
  anthropicModel: string;
  setAnthropicModel: (v: string) => void;
  anthropicMaxTokens: string;
  setAnthropicMaxTokens: (v: string) => void;
  openaiKey: string;
  setOpenaiKey: (v: string) => void;
  openaiModel: string;
  setOpenaiModel: (v: string) => void;
  openaiBaseURL: string;
  setOpenaiBaseURL: (v: string) => void;
  openaiMaxTokens: string;
  setOpenaiMaxTokens: (v: string) => void;
}

export function ProviderTab({
  provider,
  setProvider,
  anthropicKey,
  setAnthropicKey,
  anthropicModel,
  setAnthropicModel,
  anthropicMaxTokens,
  setAnthropicMaxTokens,
  openaiKey,
  setOpenaiKey,
  openaiModel,
  setOpenaiModel,
  openaiBaseURL,
  setOpenaiBaseURL,
  openaiMaxTokens,
  setOpenaiMaxTokens,
}: ProviderTabProps) {
  return (
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
            <input
              type="password"
              style={inputStyle}
              value={anthropicKey}
              onChange={e => setAnthropicKey(e.target.value)}
              placeholder="sk-ant-api03-..."
            />
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
            <input
              type="number"
              style={inputStyle}
              value={anthropicMaxTokens}
              onChange={e => setAnthropicMaxTokens(e.target.value)}
              placeholder="262000"
            />
          </div>
        </>
      )}

      {provider === 'openai' && (
        <>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>API Key</label>
            <input
              type="password"
              style={inputStyle}
              value={openaiKey}
              onChange={e => setOpenaiKey(e.target.value)}
              placeholder="sk-..."
            />
          </div>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Model</label>
            <input
              type="text"
              style={inputStyle}
              value={openaiModel}
              onChange={e => setOpenaiModel(e.target.value)}
              placeholder="gpt-4, gpt-4-turbo..."
            />
          </div>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Base URL (optional)</label>
            <input
              type="text"
              style={inputStyle}
              value={openaiBaseURL}
              onChange={e => setOpenaiBaseURL(e.target.value)}
              placeholder="https://api.openai.com/v1"
            />
            <div style={{ fontSize: '0.75rem', color: '#888', marginTop: '4px' }}>
              Leave empty for OpenAI. For OpenRouter, use https://openrouter.ai/api/v1
            </div>
          </div>
          <div style={{ marginBottom: '18px' }}>
            <label style={labelStyle}>Max Context Tokens (0 = auto ~262K)</label>
            <input
              type="number"
              style={inputStyle}
              value={openaiMaxTokens}
              onChange={e => setOpenaiMaxTokens(e.target.value)}
              placeholder="262000"
            />
          </div>
        </>
      )}
    </>
  );
}

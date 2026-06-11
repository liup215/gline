import { inputStyle, labelStyle, selectStyle } from './sharedStyles';

interface MemoryTabProps {
  memEnabled: boolean;
  setMemEnabled: (v: boolean) => void;
  memProvider: string;
  setMemProvider: (v: string) => void;
  memModel: string;
  setMemModel: (v: string) => void;
  memAPIKey: string;
  setMemAPIKey: (v: string) => void;
  memBaseURL: string;
  setMemBaseURL: (v: string) => void;
  memTopK: string;
  setMemTopK: (v: string) => void;
  memMinScore: string;
  setMemMinScore: (v: string) => void;
  memMaxTokens: string;
  setMemMaxTokens: (v: string) => void;
}

export function MemoryTab({
  memEnabled,
  setMemEnabled,
  memProvider,
  setMemProvider,
  memModel,
  setMemModel,
  memAPIKey,
  setMemAPIKey,
  memBaseURL,
  setMemBaseURL,
  memTopK,
  setMemTopK,
  memMinScore,
  setMemMinScore,
  memMaxTokens,
  setMemMaxTokens,
}: MemoryTabProps) {
  return (
    <>
      <div style={{ marginBottom: '18px', display: 'flex', alignItems: 'center', gap: '10px' }}>
        <input
          type="checkbox"
          id="memEnabled"
          checked={memEnabled}
          onChange={e => setMemEnabled(e.target.checked)}
          style={{ width: '18px', height: '18px', cursor: 'pointer' }}
        />
        <label htmlFor="memEnabled" style={{ ...labelStyle, marginBottom: 0, cursor: 'pointer' }}>
          Enable Memory & Knowledge Base
        </label>
      </div>

      {memEnabled && (
        <>
          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Embedding Provider</label>
            <select style={selectStyle} value={memProvider} onChange={e => setMemProvider(e.target.value)}>
              <option value="openai">OpenAI</option>
              <option value="ollama">Ollama</option>
            </select>
          </div>

          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Embedding Model</label>
            {memProvider === 'openai' ? (
              <select style={selectStyle} value={memModel} onChange={e => setMemModel(e.target.value)}>
                <option value="text-embedding-3-small">text-embedding-3-small</option>
                <option value="text-embedding-3-large">text-embedding-3-large</option>
                <option value="text-embedding-ada-002">text-embedding-ada-002</option>
              </select>
            ) : (
              <input
                type="text"
                style={inputStyle}
                value={memModel}
                onChange={e => setMemModel(e.target.value)}
                placeholder="e.g. nomic-embed-text"
              />
            )}
          </div>

          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>API Key (optional, falls back to LLM key)</label>
            <input
              type="password"
              style={inputStyle}
              value={memAPIKey}
              onChange={e => setMemAPIKey(e.target.value)}
              placeholder="Leave empty to use LLM provider API key"
            />
          </div>

          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Base URL (optional)</label>
            <input
              type="text"
              style={inputStyle}
              value={memBaseURL}
              onChange={e => setMemBaseURL(e.target.value)}
              placeholder="https://api.openai.com/v1"
            />
          </div>

          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Top-K Results</label>
            <input
              type="number"
              style={inputStyle}
              value={memTopK}
              onChange={e => setMemTopK(e.target.value)}
              placeholder="5"
            />
          </div>

          <div style={{ marginBottom: '14px' }}>
            <label style={labelStyle}>Minimum Similarity Score (0-1)</label>
            <input
              type="number"
              step="0.1"
              min="0"
              max="1"
              style={inputStyle}
              value={memMinScore}
              onChange={e => setMemMinScore(e.target.value)}
              placeholder="0.6"
            />
          </div>

          <div style={{ marginBottom: '18px' }}>
            <label style={labelStyle}>Max Retrieval Tokens</label>
            <input
              type="number"
              style={inputStyle}
              value={memMaxTokens}
              onChange={e => setMemMaxTokens(e.target.value)}
              placeholder="2000"
            />
          </div>
        </>
      )}
    </>
  );
}

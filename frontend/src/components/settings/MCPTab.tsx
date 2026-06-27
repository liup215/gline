import { useState, useEffect } from 'react';
import { THEME } from '../../theme';
import { MCPServerStatus } from '../../../bindings/github.com/liup215/gline/internal/gui';
import { GetMCPStatus } from '../../../bindings/github.com/liup215/gline/internal/gui/chatservice';

interface MCPServer {
  name: string;
  transport_type?: 'stdio' | 'http' | 'sse';
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  url?: string;
  headers?: Record<string, string>;
  disabled?: boolean;
}

interface MCPTabProps {
  servers: MCPServer[];
  onSave: (servers: MCPServer[]) => void;
}

export function MCPTab({ servers, onSave }: MCPTabProps) {
  const [localServers, setLocalServers] = useState<MCPServer[]>(servers || []);
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [serverStatuses, setServerStatuses] = useState<Record<string, MCPServerStatus>>({});
  const [expandedServers, setExpandedServers] = useState<Record<string, boolean>>({});
  
  // Form state
  const [formName, setFormName] = useState('');
  const [formType, setFormType] = useState<'stdio' | 'http' | 'sse'>('stdio');
  const [formCommand, setFormCommand] = useState('');
  const [formArgs, setFormArgs] = useState('');
  const [formURL, setFormURL] = useState('');
  const [formEnv, setFormEnv] = useState('');

  // Fetch MCP status on mount and when servers change
  useEffect(() => {
    fetchMCPStatus();
  }, [servers]);

  const fetchMCPStatus = async () => {
    try {
      const statuses = await GetMCPStatus();
      const statusMap: Record<string, MCPServerStatus> = {};
      statuses.forEach((s: MCPServerStatus) => {
        statusMap[s.name] = s;
      });
      setServerStatuses(statusMap);
    } catch (err) {
      // Ignore errors - MCP manager might not be initialized yet
    }
  };

  const resetForm = () => {
    setFormName('');
    setFormType('stdio');
    setFormCommand('');
    setFormArgs('');
    setFormURL('');
    setFormEnv('');
    setEditingIndex(null);
    setShowAddForm(false);
  };

  const handleAdd = () => {
    if (!formName.trim()) return;

    const newServer: MCPServer = {
      name: formName.trim(),
      transport_type: formType,
      disabled: false,
    };

    if (formType === 'stdio') {
      newServer.command = formCommand.trim();
      if (formArgs.trim()) {
        newServer.args = formArgs.split(/\s+/).filter(Boolean);
      }
    } else {
      newServer.url = formURL.trim();
    }

    if (formEnv.trim()) {
      newServer.env = {};
      formEnv.split('\n').forEach(line => {
        const [key, ...valueParts] = line.split('=');
        if (key && valueParts.length > 0) {
          newServer.env![key.trim()] = valueParts.join('=').trim();
        }
      });
    }

    if (editingIndex !== null) {
      const updated = [...localServers];
      updated[editingIndex] = newServer;
      setLocalServers(updated);
    } else {
      setLocalServers([...localServers, newServer]);
    }

    onSave(editingIndex !== null ? [...localServers.slice(0, editingIndex), newServer, ...localServers.slice(editingIndex + 1)] : [...localServers, newServer]);
    resetForm();
  };

  const handleEdit = (index: number) => {
    const server = localServers[index];
    setFormName(server.name);
    // Determine transport type: explicit transport_type, or infer from url/command
    let transport: 'stdio' | 'http' | 'sse' = 'stdio';
    if (server.transport_type) {
      transport = server.transport_type;
    } else if (server.url) {
      transport = 'http'; // Default to HTTP for URL-based servers
    }
    setFormType(transport);
    setFormCommand(server.command || '');
    setFormArgs(server.args?.join(' ') || '');
    setFormURL(server.url || '');
    setFormEnv(server.env ? Object.entries(server.env).map(([k, v]) => `${k}=${v}`).join('\n') : '');
    setEditingIndex(index);
    setShowAddForm(true);
  };

  const handleDelete = (index: number) => {
    const updated = localServers.filter((_, i) => i !== index);
    setLocalServers(updated);
    onSave(updated);
  };

  const handleToggle = (index: number) => {
    const updated = localServers.map((s, i) => 
      i === index ? { ...s, disabled: !s.disabled } : s
    );
    setLocalServers(updated);
    onSave(updated);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* Header */}
      <div style={{ 
        padding: '12px 16px', 
        background: THEME.assistantBg,
        borderRadius: '8px',
        border: `1px solid ${THEME.border}`,
      }}>
        <div style={{ fontSize: '0.9rem', fontWeight: 500, marginBottom: '4px' }}>
          🤖 Model Context Protocol (MCP)
        </div>
        <div style={{ fontSize: '0.8rem', color: THEME.textMuted }}>
          Connect external tools and data sources via MCP servers. 
          <a href="https://modelcontextprotocol.io" target="_blank" rel="noopener" style={{ color: THEME.accent, marginLeft: '4px' }}>Learn more →</a>
        </div>
      </div>

      {/* Server List */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {localServers.length === 0 ? (
          <div style={{ 
            padding: '32px', 
            textAlign: 'center', 
            color: THEME.textMuted,
            border: `1px dashed ${THEME.border}`,
            borderRadius: '8px',
          }}>
            <div style={{ fontSize: '1.5rem', marginBottom: '8px' }}>📭</div>
            <div>No MCP servers configured</div>
            <div style={{ fontSize: '0.8rem', marginTop: '4px' }}>Add a server to connect external tools</div>
          </div>
        ) : (
          localServers.map((server, index) => {
            const status = serverStatuses[server.name];
            const isExpanded = expandedServers[server.name];
            const isConnected = status?.connected && status?.initialized;
            const hasError = status?.lastError;
            
            return (
              <div key={server.name}>
                <div
                  style={{
                    padding: '12px 16px',
                    background: server.disabled ? 'transparent' : THEME.inputBg,
                    border: `1px solid ${THEME.border}`,
                    borderRadius: isExpanded ? '8px 8px 0 0' : '8px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px',
                    opacity: server.disabled ? 0.6 : 1,
                  }}
                >
                  {/* Status indicator */}
                  <div style={{
                    width: '8px',
                    height: '8px',
                    borderRadius: '50%',
                    background: server.disabled 
                      ? THEME.textDim 
                      : isConnected 
                        ? '#22c55e' 
                        : hasError 
                          ? '#ef4444' 
                          : '#f59e0b',
                    flexShrink: 0,
                  }} />

                  {/* Server info */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ 
                      fontWeight: 500, 
                      fontSize: '0.9rem',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                    }}>
                      {server.name}
                      {server.disabled && (
                        <span style={{ 
                          fontSize: '0.7rem', 
                          color: THEME.textDim,
                          background: THEME.border,
                          padding: '2px 6px',
                          borderRadius: '4px',
                        }}>disabled</span>
                      )}
                      {!server.disabled && status && (
                        <span style={{ 
                          fontSize: '0.7rem', 
                          color: isConnected ? '#22c55e' : hasError ? '#ef4444' : '#f59e0b',
                          background: isConnected ? 'rgba(34, 197, 94, 0.1)' : hasError ? 'rgba(239, 68, 68, 0.1)' : 'rgba(245, 158, 11, 0.1)',
                          padding: '2px 6px',
                          borderRadius: '4px',
                        }}>
                          {isConnected ? `✓ ${status.tools} tools` : hasError ? '✗ Error' : '⏳ Connecting'}
                        </span>
                      )}
                    </div>
                    <div style={{ 
                      fontSize: '0.8rem', 
                      color: THEME.textMuted,
                      marginTop: '2px',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}>
                      {server.transport_type === 'stdio' || (!server.transport_type && server.command)
                        ? `stdio: ${server.command} ${server.args?.join(' ') || ''}`
                        : server.transport_type === 'sse'
                          ? `SSE: ${server.url}`
                          : `HTTP: ${server.url}`}
                    </div>
                  </div>

                  {/* Actions */}
                  <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
                    {status && status.toolNames && status.toolNames.length > 0 && (
                      <button
                        onClick={() => setExpandedServers(prev => ({ ...prev, [server.name]: !prev[server.name] }))}
                        style={{
                          padding: '6px 10px',
                          fontSize: '0.75rem',
                          background: 'transparent',
                          border: `1px solid ${THEME.border}`,
                          borderRadius: '4px',
                          color: THEME.textMuted,
                          cursor: 'pointer',
                        }}
                      >
                        {isExpanded ? 'Hide Tools' : 'Show Tools'}
                      </button>
                    )}
                    <button
                      onClick={() => handleToggle(index)}
                      style={{
                        padding: '6px 10px',
                        fontSize: '0.75rem',
                        background: 'transparent',
                        border: `1px solid ${THEME.border}`,
                        borderRadius: '4px',
                        color: THEME.textMuted,
                        cursor: 'pointer',
                      }}
                    >
                      {server.disabled ? 'Enable' : 'Disable'}
                    </button>
                    <button
                      onClick={() => handleEdit(index)}
                      style={{
                        padding: '6px 10px',
                        fontSize: '0.75rem',
                        background: 'transparent',
                        border: `1px solid ${THEME.border}`,
                        borderRadius: '4px',
                        color: THEME.textMuted,
                        cursor: 'pointer',
                      }}
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDelete(index)}
                      style={{
                        padding: '6px 10px',
                        fontSize: '0.75rem',
                        background: 'transparent',
                        border: `1px solid ${THEME.toastError}`,
                        borderRadius: '4px',
                        color: THEME.toastError,
                        cursor: 'pointer',
                      }}
                    >
                      Delete
                    </button>
                  </div>
                </div>

                {/* Tools list */}
                {isExpanded && status && status.toolNames && status.toolNames.length > 0 && (
                  <div style={{
                    padding: '12px 16px',
                    background: THEME.assistantBg,
                    border: `1px solid ${THEME.border}`,
                    borderTop: 'none',
                    borderRadius: '0 0 8px 8px',
                  }}>
                    <div style={{ 
                      fontSize: '0.75rem', 
                      color: THEME.textMuted,
                      marginBottom: '8px',
                      fontWeight: 500,
                    }}>
                      Available Tools ({status.toolNames.length}):
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                      {status.toolNames.map((toolName, i) => (
                        <div key={i} style={{
                          fontSize: '0.8rem',
                          color: THEME.text,
                          padding: '4px 8px',
                          background: THEME.inputBg,
                          borderRadius: '4px',
                          fontFamily: 'monospace',
                        }}>
                          • {toolName}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Error message */}
                {hasError && (
                  <div style={{
                    padding: '10px 16px',
                    background: 'rgba(239, 68, 68, 0.1)',
                    border: `1px solid rgba(239, 68, 68, 0.3)`,
                    borderTop: 'none',
                    borderRadius: '0 0 8px 8px',
                    fontSize: '0.8rem',
                    color: '#ef4444',
                  }}>
                    Error: {status.lastError}
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>

      {/* Add Server Button */}
      {!showAddForm && (
        <button
          onClick={() => setShowAddForm(true)}
          style={{
            padding: '12px',
            background: THEME.accent,
            border: 'none',
            borderRadius: '8px',
            color: 'white',
            fontSize: '0.9rem',
            fontWeight: 500,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '8px',
          }}
        >
          <span>+</span>
          <span>Add MCP Server</span>
        </button>
      )}

      {/* Add/Edit Form */}
      {showAddForm && (
        <div style={{
          padding: '16px',
          background: THEME.assistantBg,
          border: `1px solid ${THEME.border}`,
          borderRadius: '8px',
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
        }}>
          <div style={{ fontWeight: 600, fontSize: '0.95rem' }}>
            {editingIndex !== null ? 'Edit MCP Server' : 'Add MCP Server'}
          </div>

          {/* Server Name */}
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '4px' }}>
              Server Name *
            </label>
            <input
              type="text"
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder="e.g., filesystem, github, fetch"
              style={{
                width: '100%',
                padding: '8px 12px',
                background: THEME.inputBg,
                border: `1px solid ${THEME.border}`,
                borderRadius: '6px',
                color: THEME.text,
                fontSize: '0.9rem',
              }}
            />
          </div>

          {/* Transport Type */}
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '4px' }}>
              Transport Type
            </label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                onClick={() => setFormType('stdio')}
                style={{
                  flex: 1,
                  padding: '8px',
                  background: formType === 'stdio' ? THEME.accent : THEME.inputBg,
                  border: `1px solid ${THEME.border}`,
                  borderRadius: '6px',
                  color: formType === 'stdio' ? 'white' : THEME.text,
                  cursor: 'pointer',
                }}
              >
                stdio (Local)
              </button>
              <button
                onClick={() => setFormType('http')}
                style={{
                  flex: 1,
                  padding: '8px',
                  background: formType === 'http' ? THEME.accent : THEME.inputBg,
                  border: `1px solid ${THEME.border}`,
                  borderRadius: '6px',
                  color: formType === 'http' ? 'white' : THEME.text,
                  cursor: 'pointer',
                }}
              >
                HTTP (MCP 2025)
              </button>
              <button
                onClick={() => setFormType('sse')}
                style={{
                  flex: 1,
                  padding: '8px',
                  background: formType === 'sse' ? THEME.accent : THEME.inputBg,
                  border: `1px solid ${THEME.border}`,
                  borderRadius: '6px',
                  color: formType === 'sse' ? 'white' : THEME.text,
                  cursor: 'pointer',
                }}
              >
                SSE (Legacy)
              </button>
            </div>
          </div>

          {/* stdio fields */}
          {formType === 'stdio' && (
            <>
              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '4px' }}>
                  Command *
                </label>
                <input
                  type="text"
                  value={formCommand}
                  onChange={(e) => setFormCommand(e.target.value)}
                  placeholder="e.g., npx, uvx, python"
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    background: THEME.inputBg,
                    border: `1px solid ${THEME.border}`,
                    borderRadius: '6px',
                    color: THEME.text,
                    fontSize: '0.9rem',
                  }}
                />
              </div>
              <div>
                <label style={{ display: 'block', fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '4px' }}>
                  Arguments (space-separated)
                </label>
                <input
                  type="text"
                  value={formArgs}
                  onChange={(e) => setFormArgs(e.target.value)}
                  placeholder="e.g., -y @modelcontextprotocol/server-filesystem /path/to/files"
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    background: THEME.inputBg,
                    border: `1px solid ${THEME.border}`,
                    borderRadius: '6px',
                    color: THEME.text,
                    fontSize: '0.9rem',
                  }}
                />
              </div>
            </>
          )}

          {/* HTTP/SSE fields */}
          {(formType === 'http' || formType === 'sse') && (
            <div>
              <label style={{ display: 'block', fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '4px' }}>
                URL *
              </label>
              <input
                type="text"
                value={formURL}
                onChange={(e) => setFormURL(e.target.value)}
                placeholder={formType === 'http' ? "e.g., https://example.com/mcp" : "e.g., https://mcp.example.com/sse"}
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  background: THEME.inputBg,
                  border: `1px solid ${THEME.border}`,
                  borderRadius: '6px',
                  color: THEME.text,
                  fontSize: '0.9rem',
                }}
              />
              <div style={{ fontSize: '0.75rem', color: THEME.textMuted, marginTop: '4px' }}>
                {formType === 'http' 
                  ? 'MCP Streamable HTTP endpoint (MCP 2025-11-25 spec)'
                  : 'Legacy SSE endpoint (for older MCP servers)'}
              </div>
            </div>
          )}

          {/* Environment Variables */}
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '4px' }}>
              Environment Variables (one per line, KEY=value)
            </label>
            <textarea
              value={formEnv}
              onChange={(e) => setFormEnv(e.target.value)}
              placeholder="GITHUB_TOKEN=your_token_here&#10;API_KEY=${ENV_VAR}"
              rows={3}
              style={{
                width: '100%',
                padding: '8px 12px',
                background: THEME.inputBg,
                border: `1px solid ${THEME.border}`,
                borderRadius: '6px',
                color: THEME.text,
                fontSize: '0.9rem',
                resize: 'vertical',
              }}
            />
          </div>

          {/* Action buttons */}
          <div style={{ display: 'flex', gap: '8px', marginTop: '4px' }}>
            <button
              onClick={handleAdd}
              disabled={!formName.trim() || (formType === 'stdio' ? !formCommand.trim() : !formURL.trim())}
              style={{
                flex: 1,
                padding: '10px',
                background: THEME.accent,
                border: 'none',
                borderRadius: '6px',
                color: 'white',
                fontSize: '0.9rem',
                fontWeight: 500,
                cursor: 'pointer',
                opacity: (!formName.trim() || (formType === 'stdio' ? !formCommand.trim() : !formURL.trim())) ? 0.5 : 1,
              }}
            >
              {editingIndex !== null ? 'Update Server' : 'Add Server'}
            </button>
            <button
              onClick={resetForm}
              style={{
                flex: 1,
                padding: '10px',
                background: 'transparent',
                border: `1px solid ${THEME.border}`,
                borderRadius: '6px',
                color: THEME.text,
                fontSize: '0.9rem',
                cursor: 'pointer',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Quick Examples */}
      <div style={{
        padding: '12px 16px',
        background: 'transparent',
        border: `1px dashed ${THEME.border}`,
        borderRadius: '8px',
      }}>
        <div style={{ fontSize: '0.8rem', color: THEME.textMuted, marginBottom: '8px' }}>
          💡 Quick Examples:
        </div>
        <div style={{ fontSize: '0.75rem', color: THEME.textDim, display: 'flex', flexDirection: 'column', gap: '4px' }}>
          <code style={{ background: THEME.codeBg, padding: '2px 6px', borderRadius: '4px' }}>
            npx -y @modelcontextprotocol/server-filesystem /path/to/files
          </code>
          <code style={{ background: THEME.codeBg, padding: '2px 6px', borderRadius: '4px' }}>
            npx -y @modelcontextprotocol/server-github
          </code>
          <code style={{ background: THEME.codeBg, padding: '2px 6px', borderRadius: '4px' }}>
            uvx mcp-server-fetch
          </code>
        </div>
      </div>
    </div>
  );
}

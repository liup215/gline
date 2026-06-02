import { useState, useEffect, useRef, FormEvent } from 'react'
import hljs from 'highlight.js';
import 'highlight.js/styles/github-dark.css';
import { marked } from 'marked';
import katex from 'katex';
import 'katex/dist/katex.min.css';
import { Events, WML } from "@wailsio/runtime";
import { ChatService } from "../bindings/github.com/liup215/gline/gui";
import { TaskRecord, MessageRecord } from "../bindings/github.com/liup215/gline/internal/storage/models";
import { useSlashCommands, SlashMenu } from "./slash";

interface Message {
  role: 'user' | 'assistant' | 'tool' | 'system';
  content: string;
  id?: string;
  toolName?: string;
  toolInput?: string;
  toolResult?: string;
  streaming?: boolean;
}

const THEME = {
  bg: '#0b1220',
  bgSidebar: '#0f172a',
  bgChat: '#0b1220',
  border: '#1e293b',
  accent: '#3b82f6',
  accentHover: '#2563eb',
  text: '#e2e8f0',
  textMuted: '#94a3b8',
  textDim: '#64748b',
  assistantBg: '#111827',
  userBg: '#1e40af',
  toolBg: '#111827',
  codeBg: '#1e1e2e',
};

function useHighlightCode() {
  useEffect(() => {
    // Small timeout to let marked render complete before highlighting
    requestAnimationFrame(() => {
      document.querySelectorAll('pre code.hljs').forEach((block) => {
        try {
          hljs.highlightElement(block as HTMLElement);
        } catch (e) {
          // Some blocks may have no matching grammar
        }
      });
    });
  });
}

function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const chatInputRef = useRef<HTMLInputElement>(null);

  const [history, setHistory] = useState<TaskRecord[]>([]);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [activeTaskID, setActiveTaskID] = useState<string | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [configData, setConfigData] = useState<any>(null);
  const [saveMessage, setSaveMessage] = useState('');

  const [followup, setFollowup] = useState<{ question: string; options: string[] } | null>(null);
  const [mode, setMode] = useState<'plan' | 'act'>('act');
  const [status, setStatus] = useState<{ provider: string; model: string; cwd: string; currentTokens: string; maxTokens: string }>({ provider: '', model: '', cwd: '', currentTokens: '0', maxTokens: '0' });

  const { menuState, handleInputChange, handleKeyDown, selectCommand, closeMenu } = useSlashCommands();

  useHighlightCode();

  useEffect(() => {
    loadHistory();
    loadMode();
    loadStatus();
  }, []);

  async function loadHistory() {
    try {
      const tasks = await ChatService.ListTasks(50, 0);
      setHistory(tasks || []);
    } catch (err) {
      console.error('Failed to load history:', err);
    }
  }

  async function loadMode() {
    try {
      const currentMode = await ChatService.GetMode();
      setMode(currentMode === 'plan' ? 'plan' : 'act');
    } catch (err) {
      console.error('Failed to load mode:', err);
    }
  }

  async function loadStatus() {
    try {
      const s = await ChatService.GetStatus();
      setStatus({
        provider: s.provider || '',
        model: s.model || '',
        cwd: s.cwd || '',
        currentTokens: s.currentTokens || '0',
        maxTokens: s.maxTokens || '0',
      });
    } catch (err) {
      console.error('Failed to load status:', err);
    }
  }

  useEffect(() => {
    Events.On('chat:streamStart', () => {
      setIsLoading(true);
      setMessages(prev => [...prev, { role: 'assistant', content: '', streaming: true }]);
    });

    Events.On('chat:content', (data: any) => {
      const delta = data?.data ?? '';
      setMessages(prev => {
        const last = prev[prev.length - 1];
        if (last && last.role === 'assistant' && last.streaming) {
          return [...prev.slice(0, -1), { ...last, content: last.content + delta }];
        }
        return prev;
      });
    });

    Events.On('chat:toolStart', (data: any) => {
      const { id, name, input: toolInput } = data?.data ?? {};
      setMessages(prev => {
        // Drop the preceding empty assistant bubble if it has no content
        const last = prev[prev.length - 1];
        if (last && last.role === 'assistant' && last.content.trim() === '' && last.streaming) {
          return [...prev.slice(0, -1), { role: 'tool', id, toolName: name, toolInput, content: '' }];
        }
        return [...prev, { role: 'tool', id, toolName: name, toolInput, content: '' }];
      });
    });

    Events.On('chat:toolComplete', (data: any) => {
      const { id, result } = data?.data ?? {};
      setMessages(prev => prev.map(m => (m.id === id ? { ...m, toolResult: result } : m)));
      loadStatus();
    });

    Events.On('chat:error', (data: any) => {
      const err = data?.data ?? 'Unknown error';
      setIsLoading(false);
      setMessages(prev => [...prev, { role: 'system', content: `Error: ${err}` }]);
    });

    Events.On('chat:complete', () => {
      setIsLoading(false);
      setMessages(prev => {
        const last = prev[prev.length - 1];
        if (last && last.role === 'assistant' && last.streaming) {
          return [...prev.slice(0, -1), { ...last, streaming: false }];
        }
        return prev;
      });
      loadHistory();
      loadStatus();
    });

    Events.On('chat:taskCreated', () => {
      loadHistory();
    });

    Events.On('chat:followupQuestion', (data: any) => {
      const q = data?.data?.question ?? '';
      const opts = (data?.data?.options as string[]) || [];
      setFollowup({ question: q, options: opts });
    });

    WML.Reload();
  }, []);

  // Global keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey) {
        if (e.key === 'n' || e.key === 'N') {
          e.preventDefault();
          handleNewConversation();
        }
        if (e.key === 'k' || e.key === 'K') {
          e.preventDefault();
          const inputRef = document.querySelector('input[placeholder*="Ask gline"]') as HTMLInputElement;
          inputRef?.focus();
        }
        if (e.key === 'b' || e.key === 'B') {
          e.preventDefault();
          setSidebarOpen(prev => !prev);
        }
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const executeSlashCommand = async (name: string, args: string) => {
    try {
      const result: any = await ChatService.ExecuteSlashCommand(name, args);
      const action = result?.action || 'none';
      const msg = result?.message || '';

      switch (action) {
        case 'clear': {
          ChatService.NewConversation();
          setMessages([]);
          setInput('');
          setIsLoading(false);
          setActiveTaskID(null);
          ChatService.SetMode('act').then(() => setMode('act')).catch(() => {});
          if (msg) {
            setMessages(prev => [...prev, { role: 'system', content: msg }]);
          }
          break;
        }
        case 'newtask': {
          handleNewConversation();
          if (msg) {
            setMessages(prev => [...prev, { role: 'system', content: msg }]);
          }
          break;
        }
        case 'compact': {
          const compacted = await ChatService.CompactConversation();
          if (compacted) {
            setMessages(prev => [...prev, { role: 'system', content: msg || 'Conversation compacted' }]);
          }
          loadStatus();
          break;
        }
        case 'help': {
          const helpText = await ChatService.BuildHelpText();
          setMessages(prev => [...prev, { role: 'system', content: helpText || msg || 'Help available' }]);
          break;
        }
        case 'history': {
          setSidebarOpen(true);
          if (msg) {
            setMessages(prev => [...prev, { role: 'system', content: msg }]);
          }
          break;
        }
        case 'quit': {
          // Signal backend quit; frontend can close window if Wails exposes it
          if (msg) {
            setMessages(prev => [...prev, { role: 'system', content: msg }]);
          }
          break;
        }
        default: {
          if (msg) {
            setMessages(prev => [...prev, { role: 'system', content: msg }]);
          }
          break;
        }
      }
    } catch (err: any) {
      setMessages(prev => [...prev, { role: 'system', content: `Slash command error: ${err}` }]);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const prompt = input.trim();

    // Check if this is a standalone slash command
    const isSlash = await ChatService.IsSlashCommand(prompt);
    if (isSlash) {
      setInput('');
      const [name, args] = await ChatService.ParseSlashCommand(prompt);
      if (name) {
        await executeSlashCommand(name, args);
        return;
      }
    }

    setInput('');
    setMessages(prev => [...prev, { role: 'user', content: prompt }]);

    try {
      await ChatService.SendMessage(prompt);
    } catch (err: any) {
      setMessages(prev => [...prev, { role: 'system', content: `Error: ${err}` }]);
      setIsLoading(false);
    }
  };

  const handleNewConversation = () => {
    ChatService.NewConversation();
    setMessages([]);
    setInput('');
    setIsLoading(false);
    setActiveTaskID(null);
    // Reset mode to act for new conversations
    ChatService.SetMode('act').then(() => setMode('act')).catch(() => {});
  };

  const handleSelectTask = async (taskID: string) => {
    try {
      const [task, msgs] = await ChatService.GetTaskSummary(taskID);
      if (!task) return;
      await ChatService.LoadTask(taskID);
      setActiveTaskID(taskID);
      // Step 1: collect tool metadata from assistant messages
      const toolInfoMap: Record<string, { name: string; input: string }> = {};
      (msgs || []).forEach((m: MessageRecord) => {
        if (m.Role === 'assistant' && m.ToolCalls && m.ToolCalls.trim() !== '') {
          try {
            const tcs = JSON.parse(m.ToolCalls);
            tcs.forEach((tc: any) => {
              const id = tc.ID || tc.id;
              if (!id) return;
              const name = tc.Name || tc.name || '';
              const inputRaw = tc.Input || tc.input || (tc.arguments ? JSON.stringify(tc.arguments) : '');
              toolInfoMap[id] = {
                name,
                input: typeof inputRaw === 'string' ? inputRaw : JSON.stringify(inputRaw),
              };
            });
          } catch (e) { /* ignore */ }
        }
      });
      // Step 2: map messages, enriching tool results with name/input
      const displayMessages: Message[] = (msgs || []).map((m: MessageRecord) => {
        const base: Message = { role: m.Role as any, content: m.Content };
        if (m.Role === 'tool' && m.ToolCallID && toolInfoMap[m.ToolCallID]) {
          const info = toolInfoMap[m.ToolCallID];
          base.id = m.ToolCallID;
          base.toolName = info.name;
          base.toolInput = info.input;
          base.toolResult = m.Content;
        }
        return base;
      });
      setMessages(displayMessages);
    } catch (err) {
      console.error('Failed to load task:', err);
    }
  };

  const handleDeleteTask = async (e: React.MouseEvent, taskID: string) => {
    e.stopPropagation();
    if (!confirm('Delete this conversation?')) return;
    try {
      await ChatService.DeleteTask(taskID);
      if (activeTaskID === taskID) {
        setMessages([]);
        setActiveTaskID(null);
      }
      loadHistory();
    } catch (err) {
      console.error('Failed to delete task:', err);
    }
  };

  const handleFollowupAnswer = async (answer: string) => {
    setFollowup(null);
    try {
      await ChatService.AnswerFollowupQuestion(answer);
    } catch (e) {
      console.error('Failed to send followup answer:', e);
    }
  };

  async function loadConfig() {
    try {
      const cfgJson = await ChatService.GetConfig();
      setConfigData(JSON.parse(cfgJson));
    } catch (err) {
      console.error('Failed to load config:', err);
    }
  }

  const renderMessage = (msg: Message, idx: number) => {
    if (msg.role === 'user') {
      return (
        <div key={idx} style={{ display: 'flex', justifyContent: 'flex-end', padding: '0 24px', marginBottom: '16px' }}>
          <div style={{ maxWidth: '70%', background: THEME.userBg, color: '#fff', padding: '12px 18px', borderRadius: '18px 18px 4px 18px', lineHeight: 1.5, fontSize: '0.95rem', boxShadow: '0 2px 8px rgba(0,0,0,0.2)' }}>
            {msg.content}
          </div>
        </div>
      );
    }
    if (msg.role === 'assistant') {
      // Only show cursor on the last assistant message that's still streaming
      const isLast = idx === messages.length - 1;
      return (
        <div key={idx} style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '16px' }}>
          <div style={{ maxWidth: '85%', background: THEME.assistantBg, color: THEME.text, padding: '14px 20px', borderRadius: '18px 18px 18px 4px', lineHeight: 1.6, fontSize: '0.95rem', border: `1px solid ${THEME.border}`, boxShadow: '0 2px 8px rgba(0,0,0,0.15)' }}>
            <div className="md-rendered" dangerouslySetInnerHTML={{ __html: formatContent(msg.content) }} />
            {msg.streaming && isLast && <span style={{ color: THEME.accent, animation: 'blink 1s infinite' }}>▌</span>}
          </div>
        </div>
      );
    }
    if (msg.role === 'tool') {
      const done = !!msg.toolResult;
      const hint = getToolHint(msg.toolName || '', msg.toolInput);
      const isQuestion = msg.toolName === 'ask_followup_question' && hint.startsWith('💬');
      if (isQuestion) {
        return (
          <div key={idx} style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '12px' }}>
            <div style={{ maxWidth: '75%', background: 'rgba(59,130,246,0.10)', border: `1px solid rgba(59,130,246,0.30)`, borderRadius: '14px 14px 14px 4px', padding: '12px 16px', lineHeight: 1.5, fontSize: '0.92rem', color: THEME.text }}>
              <div style={{ fontSize: '0.72rem', color: '#60a5fa', marginBottom: '6px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>Question</div>
              <div>{hint.slice(2).trim()}</div>
              {(() => {
                const input = parseToolInput(msg.toolInput);
                if (input.options && input.options.length > 0) {
                  return (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', marginTop: '10px' }}>
                      {input.options.map((opt: string, oi: number) => (
                        <span key={oi} style={{ padding: '4px 10px', borderRadius: '6px', background: 'rgba(59,130,246,0.15)', color: '#93c5fd', fontSize: '0.82rem', border: '1px solid rgba(59,130,246,0.25)' }}>{opt}</span>
                      ))}
                    </div>
                  );
                }
                return null;
              })()}
            </div>
          </div>
        );
      }
      return (
        <div key={idx} style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '8px' }}>
          <div
            title={msg.toolInput || ''}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '8px',
              padding: '4px 12px',
              borderRadius: '20px',
              background: done ? 'rgba(74, 222, 128, 0.08)' : 'rgba(251, 191, 36, 0.08)',
              border: `1px solid ${done ? 'rgba(74, 222, 128, 0.25)' : 'rgba(251, 191, 36, 0.25)'}`,
              color: done ? '#4ade80' : '#fbbf24',
              fontSize: '0.78rem',
              fontFamily: '"SFMono-Regular", Consolas, monospace',
              lineHeight: 1.4,
              maxWidth: '70%',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            <span style={{ fontSize: '0.85em', flexShrink: 0 }}>{done ? '✓' : '◌'}</span>
            <span>{hint}</span>
          </div>
        </div>
      );
    }
    if (msg.role === 'system') {
      return (
        <div key={idx} style={{ display: 'flex', justifyContent: 'center', padding: '0 24px', marginBottom: '12px' }}>
          <div style={{ maxWidth: '80%', background: '#451a03', color: '#fbbf24', padding: '10px 16px', borderRadius: '8px', fontSize: '0.85rem', textAlign: 'center' }}>
            {msg.content}
          </div>
        </div>
      );
    }
    return null;
  };

  return (
    <div style={{ display: 'flex', height: '100vh', width: '100vw', background: THEME.bg, color: THEME.text, fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif', overflow: 'hidden' }}>
      {/* Sidebar */}
      <div style={{
        width: sidebarOpen ? '280px' : '0px',
        minWidth: sidebarOpen ? '280px' : '0px',
        background: THEME.bgSidebar,
        borderRight: `1px solid ${THEME.border}`,
        display: 'flex',
        flexDirection: 'column',
        transition: 'width 0.25s ease, min-width 0.25s ease',
        overflow: 'hidden',
      }}>
        <div style={{ padding: '16px 20px', borderBottom: `1px solid ${THEME.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <h1 style={{ margin: 0, fontSize: '1.1rem', fontWeight: 700, background: 'linear-gradient(90deg, #60a5fa, #a78bfa)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>gline</h1>
          <button
            style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.3rem', padding: '2px 6px', borderRadius: '6px' }}
            onClick={handleNewConversation}
            title="New conversation"
          >
            +
          </button>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: '12px 10px' }}>
          <div style={{ fontSize: '0.7rem', color: THEME.textDim, textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '8px', padding: '0 10px' }}>
            Recent Conversations
          </div>
          {history.length === 0 && (
            <div style={{ padding: '20px', color: THEME.textDim, fontSize: '0.85rem', textAlign: 'center' }}>
              No conversations yet
            </div>
          )}
          {history.map((task) => (
            <div
              key={task.ID}
              onClick={() => task.ID && handleSelectTask(task.ID)}
              style={{
                padding: '10px 14px',
                borderRadius: '8px',
                marginBottom: '4px',
                cursor: 'pointer',
                background: activeTaskID === task.ID ? 'rgba(59, 130, 246, 0.15)' : 'transparent',
                color: activeTaskID === task.ID ? THEME.text : THEME.textMuted,
                fontSize: '0.85rem',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                transition: 'background 0.15s',
              }}
              onMouseEnter={e => { if (activeTaskID !== task.ID) e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; }}
              onMouseLeave={e => { if (activeTaskID !== task.ID) e.currentTarget.style.background = 'transparent'; }}
            >
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                {task.Title || 'Untitled conversation'}
              </span>
              <button
                onClick={(e) => task.ID && handleDeleteTask(e, task.ID)}
                style={{ background: 'transparent', border: 'none', color: THEME.textDim, cursor: 'pointer', fontSize: '0.75rem', padding: '2px 4px', opacity: 0, transition: 'opacity 0.15s' }}
                onMouseEnter={e => e.currentTarget.style.opacity = '1'}
              >
                🗑
              </button>
            </div>
          ))}
        </div>

        {/* Sidebar Footer */}
        <div style={{ padding: '12px 20px', borderTop: `1px solid ${THEME.border}` }}>
          <button
            onClick={() => { setShowSettings(true); loadConfig(); }}
            style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.85rem', display: 'flex', alignItems: 'center', gap: '8px', transition: 'background 0.15s' }}
            onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.04)'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
          >
            ⚙️ Settings
          </button>
        </div>
      </div>

      {/* Main Chat Area */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 24px', borderBottom: `1px solid ${THEME.border}`, background: THEME.bg }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              style={{ background: 'transparent', border: 'none', color: THEME.textMuted, cursor: 'pointer', fontSize: '1.2rem', padding: '4px', display: 'flex', alignItems: 'center' }}
              title="Toggle sidebar"
            >
              ☰
            </button>
            <span style={{ fontSize: '0.9rem', color: THEME.textMuted }}>
              {activeTaskID ? 'Conversation' : 'New Chat'}
            </span>
            <span
              title={`Working directory: ${status.cwd}`}
              style={{ fontSize: '0.75rem', color: THEME.textDim, maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
            >
              📁 {status.cwd}
            </span>
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            {isLoading && (
              <button
                onClick={() => ChatService.StopMessage()}
                style={{ padding: '6px 14px', borderRadius: '6px', border: '1px solid #ef4444', background: 'transparent', color: '#ef4444', cursor: 'pointer', fontSize: '0.8rem' }}
              >
                ⏹ Stop
              </button>
            )}
          </div>
        </div>

        {/* Messages */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 0', display: 'flex', flexDirection: 'column' }}>
          {messages.length === 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', flex: 1, color: THEME.textDim }}>
              <h2 style={{ margin: '0 0 8px', fontWeight: 400, fontSize: '1.5rem' }}>Welcome to gline</h2>
              <p style={{ margin: 0, fontSize: '0.9rem' }}>AI Programming Assistant powered by Go</p>
            </div>
          )}
          {messages.map((msg, idx) => renderMessage(msg, idx))}
          <div ref={messagesEndRef} />
        </div>

        {/* Input */}
        <div style={{ padding: '14px 24px 20px', borderTop: `1px solid ${THEME.border}`, background: THEME.bg, position: 'relative' }}>
          <form style={{ display: 'flex', gap: '12px' }} onSubmit={handleSubmit}>
            <div style={{ flex: 1, position: 'relative' }}>
              <SlashMenu
                active={menuState.active}
                filtered={menuState.filtered}
                selectedIndex={menuState.selectedIndex}
                onSelect={(cmd) => selectCommand(cmd, setInput, chatInputRef.current)}
                inputRef={chatInputRef}
              />
              <input
                ref={chatInputRef}
                style={{ width: '100%', padding: '12px 16px', borderRadius: '10px', border: `1px solid ${THEME.border}`, background: '#1e293b', color: THEME.text, fontSize: '0.95rem', outline: 'none', transition: 'border-color 0.2s', boxSizing: 'border-box' }}
                value={input}
                onChange={e => {
                  setInput(e.target.value);
                  handleInputChange(e.target.value, e.target.selectionStart || 0, setInput, e.target);
                }}
                onKeyDown={e => {
                  const { handled } = handleKeyDown(e, setInput);
                  if (!handled && e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    handleSubmit(e as any);
                  }
                }}
                onClick={e => {
                  handleInputChange(input, e.currentTarget.selectionStart || 0, setInput, e.currentTarget);
                }}
                placeholder={isLoading ? 'AI is thinking...' : 'Ask gline anything... Type / for commands (Ctrl+N new chat, Ctrl+K focus, Ctrl+B toggle sidebar)'}
                disabled={isLoading}
                onFocus={e => e.currentTarget.style.borderColor = THEME.accent}
                onBlur={e => {
                  e.currentTarget.style.borderColor = THEME.border;
                  // Small delay to allow menu clicks to register before closing
                  setTimeout(() => closeMenu(), 200);
                }}
              />
            </div>
            <button
              type="submit"
              style={{ padding: '12px 20px', borderRadius: '10px', border: 'none', background: isLoading || !input.trim() ? '#334155' : THEME.accent, color: '#fff', cursor: isLoading || !input.trim() ? 'not-allowed' : 'pointer', fontSize: '0.95rem', fontWeight: 500, transition: 'background 0.2s' }}
              disabled={isLoading || !input.trim()}
            >
              {isLoading ? '⏳' : 'Send'}
            </button>
          </form>
          {/* Status Bar */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: '10px', padding: '0 2px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', fontSize: '0.75rem', color: THEME.textDim }}>
              <span title="Model">🤖 {status.model || status.provider || 'unknown'}</span>
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <span>Context</span>
                <div style={{ width: '80px', height: '6px', borderRadius: '3px', background: '#1e293b', overflow: 'hidden' }}>
                  <div style={{
                    width: `${Math.min(100, (parseInt(status.currentTokens || '0') / Math.max(1, parseInt(status.maxTokens || '1'))) * 100)}%`,
                    height: '100%',
                    borderRadius: '3px',
                    background: parseInt(status.currentTokens || '0') > parseInt(status.maxTokens || '0') * 0.8 ? '#ef4444' : '#3b82f6',
                    transition: 'width 0.3s',
                  }} />
                </div>
                <span>{status.currentTokens || '0'}/{status.maxTokens || '0'}</span>
              </div>
            </div>
            <button
              onClick={async () => {
                const newMode = mode === 'act' ? 'plan' : 'act';
                try {
                  await ChatService.SetMode(newMode);
                  setMode(newMode);
                } catch (err) {
                  console.error('Failed to set mode:', err);
                }
              }}
              title={`Mode: ${mode === 'act' ? 'Act (can modify files)' : 'Plan (read-only exploration)'}`}
              style={{
                padding: '3px 10px',
                borderRadius: '6px',
                border: `1px solid ${mode === 'act' ? 'rgba(59,130,246,0.4)' : 'rgba(168,85,247,0.4)'}`,
                background: mode === 'act' ? 'rgba(59,130,246,0.1)' : 'rgba(168,85,247,0.1)',
                color: mode === 'act' ? '#60a5fa' : '#c084fc',
                cursor: 'pointer',
                fontSize: '0.72rem',
                fontWeight: 600,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                transition: 'all 0.2s',
              }}
            >
              {mode === 'act' ? '⚡ Act' : '📋 Plan'}
            </button>
          </div>
        </div>
      </div>

      {/* Settings Modal Overlay */}
      {showSettings && (
        <SettingsPanel
          config={configData}
          onClose={() => { setShowSettings(false); setSaveMessage(''); }}
          onSave={async (updates) => {
            try {
              for (const [key, value] of Object.entries(updates)) {
                await ChatService.UpdateConfig(key, value as string);
              }
              setSaveMessage('Settings saved successfully!');
              setTimeout(() => setSaveMessage(''), 3000);
              loadConfig();
            } catch (err) {
              setSaveMessage('Failed to save settings');
            }
          }}
          saveMessage={saveMessage}
        />
      )}

      {/* Followup Question Modal */}
      {followup && (
        <FollowupModal question={followup.question} options={followup.options} onAnswer={handleFollowupAnswer} />
      )}
    </div>
  );
}

interface ToolInput {
  path?: string;
  command?: string;
  question?: string;
  options?: string[];
  content?: string;
  search?: string;
  replace?: string;
  regex?: string;
  file_pattern?: string;
  [key: string]: any;
}

function parseToolInput(raw: string | undefined): ToolInput {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

function getToolHint(name: string, rawInput: string | undefined): string {
  const input = parseToolInput(rawInput);
  switch (name) {
    case 'read_file':
      return input.path ? `View: ${input.path}` : name;
    case 'write_to_file':
      return input.path ? `Write: ${input.path}` : name;
    case 'replace_in_file':
      return input.path ? `Edit: ${input.path}` : name;
    case 'list_files':
      return input.path ? `List: ${input.path}` : name;
    case 'search_files':
      return input.regex ? `Search "${input.regex}"${input.path ? ` in ${input.path}` : ''}` : name;
    case 'list_code_definition_names':
      return input.path ? `Definitions in ${input.path}` : name;
    case 'execute_command':
      return input.command ? `Run: ${input.command}` : name;
    case 'ask_followup_question':
      return input.question ? `💬 ${input.question}` : name;
    case 'attempt_completion':
      return 'Complete';
    default:
      return name;
  }
}

function detectLanguage(code: string): string | null {
  const trimmed = code.trim().toLowerCase();
  if (trimmed.startsWith('<!doctype html') || trimmed.startsWith('<html')) return 'xml';
  if (/^(import|export|const|let|var|function|class|interface|type)\b/.test(trimmed)) return 'typescript';
  if (/^(package|import|func|type|struct|interface|var|const)\b/.test(trimmed)) return 'go';
  if (/^(def|class|import|from|print|if __name__)/.test(trimmed)) return 'python';
  if (trimmed.includes('#include') || trimmed.includes('int main(')) return 'cpp';
  if (trimmed.startsWith('{') || trimmed.includes('"') && trimmed.includes(':')) return 'json';
  if (trimmed.includes('dockerfile') || /^(from|run|cmd|entrypoint|copy|add)\b/.test(trimmed)) return 'dockerfile';
  if (/^(select|insert|update|delete|create table|drop table|alter table)\b/.test(trimmed)) return 'sql';
  return null;
}

// Marked options
marked.setOptions({
  gfm: true,
  breaks: true,
} as any);

// --- Math rendering helpers ---
const MATH_PLACEHOLDER_BLOCK = '§§§BLOCKMATH';
const MATH_PLACEHOLDER_INLINE = '§§§INLINEMATH';

function _renderKatex(tex: string, displayMode: boolean): string {
  try {
    return katex.renderToString(tex, {
      displayMode,
      throwOnError: false,
      strict: false,
    });
  } catch (e) {
    return `<span style="color:#ef4444;font-family:monospace;">${displayMode ? '$$' : '$'}${tex}${displayMode ? '$$' : '$'}</span>`;
  }
}

function _processFootnotes(content: string): string {
  // Support extended footnote syntax: [^1]: note text
  // Collect definitions
  const defs: Record<string, string> = {};
  let text = content.replace(/\n\[\^(\d+|[a-zA-Z-]+)\]:[ \t]*(.*(?:\n(?![a-zA-Z0-9]).*)*)/g, (_match, label, noteText) => {
    defs[label] = noteText.trim().replace(/\n[ \t]+/g, ' ');
    return '\n';
  });

  // Replace references [^1] with superscript links if def exists
  text = text.replace(/\[\^(\d+|[a-zA-Z-]+)\]/g, (_match, label) => {
    if (defs[label]) {
      return `<sup class="md-footnote-ref" data-fn="${label}" title="${escapeHtml(defs[label])}">${label}</sup>`;
    }
    return `<sup>[${label}]</sup>`;
  });

  // Append footnotes section if any defs were used
  const usedLabels = Object.keys(defs).filter(l => text.includes(`data-fn="${l}"`));
  if (usedLabels.length > 0) {
    let footnotesHtml = '\n\n<div class="md-footnotes"><hr style="border:none;border-top:1px solid #1e293b;margin:16px 1px 8px 0;"/><h4 style="font-size:0.9rem;color:#94a3b8;margin:0 0 8px;">Footnotes</h4><ol style="padding-left:1.2em;margin:0;font-size:0.85rem;color:#94a3b8;">';
    usedLabels.forEach(label => {
      footnotesHtml += `<li id="fn-${label}"><span style="color:#cbd5e1;">${escapeHtml(defs[label])}</span></li>`;
    });
    footnotesHtml += '</ol></div>';
    text = text + footnotesHtml;
  }

  return text;
}

function formatContent(content: string): string {
  // 1. Extract and protect math expressions using string scanning
  const blockMath: string[] = [];
  const inlineMath: string[] = [];
  let text = content;

  // Block math: \[ ... \]  and $$ ... $$
  function extractBlockDelim(start: string, end: string): void {
    let idx = text.indexOf(start);
    while (idx !== -1) {
      const endIdx = text.indexOf(end, idx + start.length);
      if (endIdx === -1) break;
      const tex = text.slice(idx + start.length, endIdx).trim();
      blockMath.push(tex);
      const placeholder = `\n${MATH_PLACEHOLDER_BLOCK}${blockMath.length - 1}\n`;
      text = text.slice(0, idx) + placeholder + text.slice(endIdx + end.length);
      idx = text.indexOf(start);
    }
  }
  extractBlockDelim('\\[', '\\]');
  extractBlockDelim('$$', '$$');

  // Inline math: \( ... \)  and $ ... $
  function extractInlineDelim(start: string, end: string): void {
    // For $ delimiters, be careful not to match $$ again
    const isDollar = start === '$';
    let searchFrom = 0;
    while (true) {
      let idx = text.indexOf(start, searchFrom);
      if (idx === -1) break;
      if (isDollar) {
        // Skip if it's part of $$ (already processed) or a placeholder
        if (text[idx + 1] === '$') { searchFrom = idx + 2; continue; }
        if ((text[idx - 1] || '') === '$') { searchFrom = idx + 1; continue; }
        if (text.slice(idx, idx + MATH_PLACEHOLDER_BLOCK.length) === MATH_PLACEHOLDER_BLOCK) { searchFrom = idx + 1; continue; }
      }
      const endIdx = text.indexOf(end, idx + start.length);
      if (endIdx === -1) break;
      if (isDollar && endIdx - idx < 2) { searchFrom = idx + 1; continue; } // empty
      const tex = text.slice(idx + start.length, endIdx).trim();
      // For $: skip if it looks like currency (digits only, no math symbols)
      if (isDollar && /^\d/.test(tex) && !/[\\=<>^_&{}]/.test(tex)) {
        searchFrom = endIdx + 1; continue;
      }
      inlineMath.push(tex);
      const placeholder = `${MATH_PLACEHOLDER_INLINE}${inlineMath.length - 1}`;
      text = text.slice(0, idx) + placeholder + text.slice(endIdx + end.length);
      // re-search from same position since text got shorter
      searchFrom = idx;
    }
  }
  extractInlineDelim('\\(', '\\)');
  extractInlineDelim('$', '$');

  // 2. Process footnotes
  text = _processFootnotes(text);

  // 3. Parse markdown
  let html = marked.parse(text, { async: false }) as string;

  // 4. Transform fenced code blocks: add copy button + language detection
  html = html.replace(/<pre><code class="language-([^"]*)">([\s\S]*?)<\/code><\/pre>/g, (match, lang, code) => {
    const cleanLang = detectLanguage((lang || '').toLowerCase()) || lang || 'plaintext';
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = code;
    const rawCode = tempDiv.textContent || '';
    const b64 = btoa(unescape(encodeURIComponent(rawCode)));
    return `<pre style="position:relative;background:${THEME.codeBg};padding:14px;border-radius:8px;overflow-x:auto;margin:10px 0;border:1px solid ${THEME.border};">
<span class="hljs-lang-label" style="position:absolute;top:0;left:0;padding:2px 8px;border-radius:8px 0 6px 0;background:rgba(255,255,255,0.06);color:#94a3b8;font-size:0.7rem;font-family:monospace;text-transform:uppercase;">${cleanLang}</span>
<button class="hljs-copy-btn" onclick="window.__copyCode(this)" data-clipboard="${b64}" title="Copy code" style="position:absolute;top:8px;right:8px;padding:4px 10px;border-radius:6px;border:none;background:rgba(255,255,255,0.08);color:#94a3b8;font-size:0.75rem;cursor:pointer;z-index:10;">📋 Copy</button>
<code class="hljs language-${cleanLang}">${code}</code>
</pre>`;
  });

  // Inline code styling
  html = html.replace(/<code>/g, `<code style="background:#2d2d3a;padding:2px 6px;border-radius:4px;font-size:0.9em;font-family:monospace;color:#cdd6f4;">`);

  // Dark theme overrides
  html = html.replace(/<ul>/g, `<ul style="padding-left:1.5em;margin:8px 0;color:#cbd5e1;list-style-type:disc;">`);
  html = html.replace(/<ol>/g, `<ol style="padding-left:1.5em;margin:8px 0;color:#cbd5e1;list-style-type:decimal;">`);
  html = html.replace(/<blockquote>/g, `<blockquote style="border-left:3px solid #3b82f6;margin:8px 0;padding-left:14px;color:#94a3b8;font-style:italic;">`);
  html = html.replace(/<hr\s*\/?>/g, `<hr style="border:none;border-top:1px solid #1e293b;margin:12px 0;"/>`);

  html = html.replace(/<table>/g, `<table style="width:100%;border-collapse:collapse;margin:10px 0;font-size:0.9em;">`);
  html = html.replace(/<thead>/g, `<thead style="background:#1e293b;">`);
  html = html.replace(/<th>/g, `<th style="text-align:left;padding:8px 12px;border-bottom:1px solid #334155;color:#e2e8f0;font-weight:600;">`);
  html = html.replace(/<td>/g, `<td style="padding:8px 12px;border-bottom:1px solid #1e293b;color:#cbd5e1;">`);

  html = html.replace(/<a /g, `<a style="color:#60a5fa;text-decoration:underline;" `);
  html = html.replace(/<code style="background:#2d2d3a;padding:2px 6px;border-radius:4px;font-size:0.9em;font-family:monospace;color:#cdd6f4;">\s*\n/g, '<code>\n');

  // 5. Restore and render math
  html = html.replace(new RegExp(`<p>${MATH_PLACEHOLDER_BLOCK}(\\d+)<\\/p>`, 'g'), (_match, idx) => {
    return _renderKatex(blockMath[parseInt(idx)], true);
  });
  html = html.replace(new RegExp(`${MATH_PLACEHOLDER_BLOCK}(\\d+)`, 'g'), (_match, idx) => {
    return _renderKatex(blockMath[parseInt(idx)], true);
  });
  html = html.replace(new RegExp(`${MATH_PLACEHOLDER_INLINE}(\\d+)`, 'g'), (_match, idx) => {
    return _renderKatex(inlineMath[parseInt(idx)], false);
  });

  return html;
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

// Attach global copy handler once
if (typeof window !== 'undefined' && !(window as any).__copyCode) {
  (window as any).__copyCode = async function(btn: HTMLButtonElement) {
    try {
      const b64 = btn.getAttribute('data-clipboard');
      if (!b64) return;
      const text = decodeURIComponent(escape(atob(b64)));
      await navigator.clipboard.writeText(text);
      btn.textContent = '✓ Copied!';
      btn.classList.add('copied');
      setTimeout(() => {
        btn.textContent = '📋 Copy';
        btn.classList.remove('copied');
      }, 2000);
    } catch (err) {
      console.error('Copy failed:', err);
    }
  };
}

function FollowupModal({ question, options, onAnswer }: { question: string; options: string[]; onAnswer: (ans: string) => void }) {
  const [customMode, setCustomMode] = useState(false);
  const [customValue, setCustomValue] = useState('');
  const overlayStyle: React.CSSProperties = {
    position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
    background: 'rgba(0, 0, 0, 0.6)', backdropFilter: 'blur(4px)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    zIndex: 1000,
  };
  const panelStyle: React.CSSProperties = {
    width: '440px', maxWidth: '90vw',
    background: THEME.bgSidebar,
    border: `1px solid ${THEME.border}`,
    borderRadius: '14px',
    padding: '24px 28px',
    color: THEME.text,
    boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
  };
  const optionBtnStyle: React.CSSProperties = {
    width: '100%', padding: '10px 14px', borderRadius: '8px',
    border: `1px solid ${THEME.border}`,
    background: 'rgba(59, 130, 246, 0.1)', color: THEME.text,
    cursor: 'pointer', fontSize: '0.9rem', textAlign: 'left',
    transition: 'background 0.15s',
  };
  return (
    <div style={overlayStyle}>
      <div style={panelStyle}>
        <div style={{ fontSize: '0.72rem', color: '#60a5fa', marginBottom: '10px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>Question</div>
        <div style={{ fontSize: '1rem', lineHeight: 1.5, marginBottom: '20px' }}>{question}</div>
        {customMode ? (
          <form onSubmit={(e) => { e.preventDefault(); onAnswer(customValue); }} style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <textarea
              autoFocus
              rows={3}
              value={customValue}
              onChange={e => setCustomValue(e.target.value)}
              placeholder="Type your own answer..."
              style={{ padding: '10px 12px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: '#1e293b', color: THEME.text, fontSize: '0.9rem', resize: 'vertical', fontFamily: 'inherit' }}
            />
            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button type="button" onClick={() => setCustomMode(false)} style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: 'transparent', color: THEME.textMuted, cursor: 'pointer', fontSize: '0.85rem' }}>Back</button>
              <button type="submit" style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', background: '#3b82f6', color: '#fff', cursor: 'pointer', fontSize: '0.85rem' }}>Send</button>
            </div>
          </form>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {options.map((opt, i) => (
              <button
                key={i}
                onClick={() => onAnswer(opt)}
                style={optionBtnStyle}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(59, 130, 246, 0.2)'; }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(59, 130, 246, 0.1)'; }}
              >
                {opt}
              </button>
            ))}
            {options.length > 0 && (
              <button
                onClick={() => setCustomMode(true)}
                style={optionBtnStyle}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(59, 130, 246, 0.2)'; }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(59, 130, 246, 0.1)'; }}
              >
                ✎ 其他（自行输入）
              </button>
            )}
            {options.length === 0 && (
              <form onSubmit={(e) => { e.preventDefault(); const el = (e.target as HTMLFormElement).elements.namedItem('answer') as HTMLInputElement; onAnswer(el.value); }} style={{ display: 'flex', gap: '8px' }}>
                <input name="answer" autoFocus placeholder="Type your answer..." style={{ flex: 1, padding: '10px 12px', borderRadius: '8px', border: `1px solid ${THEME.border}`, background: '#1e293b', color: THEME.text, fontSize: '0.9rem' }} />
                <button type="submit" style={{ padding: '10px 18px', borderRadius: '8px', border: 'none', background: '#3b82f6', color: '#fff', cursor: 'pointer', fontSize: '0.9rem' }}>Send</button>
              </form>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function SettingsPanel({ config, onClose, onSave, saveMessage }: { config: any; onClose: () => void; onSave: (u: any) => void; saveMessage: string }) {
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
    position: 'fixed',
    top: 0, left: 0, right: 0, bottom: 0,
    background: 'rgba(0, 0, 0, 0.6)',
    backdropFilter: 'blur(4px)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
  };

  const panelStyle: React.CSSProperties = {
    width: '520px',
    maxHeight: '85vh',
    overflowY: 'auto',
    background: '#111827',
    border: `1px solid ${THEME.border}`,
    borderRadius: '14px',
    padding: '24px 28px',
    color: THEME.text,
    boxShadow: '0 20px 60px rgba(0,0,0,0.5)',
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '10px 14px',
    borderRadius: '8px',
    border: `1px solid ${THEME.border}`,
    background: '#1e293b',
    color: THEME.text,
    fontSize: '0.9rem',
    outline: 'none',
    boxSizing: 'border-box',
  };

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontSize: '0.85rem',
    color: THEME.textMuted,
    marginBottom: '6px',
    fontWeight: 500,
  };

  const selectStyle: React.CSSProperties = {
    width: '100%',
    padding: '10px 14px',
    borderRadius: '8px',
    border: `1px solid ${THEME.border}`,
    background: '#1e293b',
    color: THEME.text,
    fontSize: '0.9rem',
    outline: 'none',
    cursor: 'pointer',
    boxSizing: 'border-box',
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

export default App

import { useState, useCallback, useRef } from 'react';
import { Events, WML } from '@wailsio/runtime';
import { ChatService } from '../../bindings/github.com/liup215/gline/internal/gui';
import { Message, FileRef } from '../types';

export function useChat(onLoadHistory: () => void, onLoadStatus: () => void, getFileRefs?: () => FileRef[], clearFileRefs?: () => void) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  // Armed by setupEventListeners; re-anchors the ordered dispatcher before each
  // send so a new run's events are never swallowed as stale (see below).
  const resetStreamOrderRef = useRef<(() => void) | null>(null);

  const [followup, setFollowup] = useState<{ question: string; options: string[] } | null>(null);

  const executeSlashCommand = useCallback(async (name: string, args: string) => {
    try {
      const result: any = await ChatService.ExecuteSlashCommand(name, args);
      const action = result?.action || 'none';
      const msg = result?.message || '';

      switch (action) {
        case 'clear': {
          ChatService.ClearConversation();
          setMessages([]);
          setInput('');
          setIsLoading(false);
          ChatService.SetMode('act').catch(() => {});
          onLoadStatus();
          if (msg) setMessages(prev => [...prev, { role: 'system', content: msg }]);
          break;
        }
        case 'newtask': {
          ChatService.StartNewConversation();
          setMessages([]);
          setInput('');
          setIsLoading(false);
          onLoadStatus();
          if (msg) setMessages(prev => [...prev, { role: 'system', content: msg }]);
          break;
        }
        case 'compact': {
          const compacted = await ChatService.CompactConversation();
          if (compacted) {
            setMessages(prev => [...prev, { role: 'system', content: msg || 'Conversation compacted' }]);
          }
          onLoadStatus();
          break;
        }
        case 'help': {
          const helpText = await ChatService.BuildHelpText();
          setMessages(prev => [...prev, { role: 'system', content: helpText || msg || 'Help available' }]);
          break;
        }
        case 'history': {
          if (msg) setMessages(prev => [...prev, { role: 'system', content: msg }]);
          break;
        }
        case 'reload': {
          if (msg) setMessages(prev => [...prev, { role: 'system', content: msg }]);
          break;
        }
        case 'quit': {
          if (msg) setMessages(prev => [...prev, { role: 'system', content: msg }]);
          break;
        }
        default: {
          if (msg) setMessages(prev => [...prev, { role: 'system', content: msg }]);
          break;
        }
      }
    } catch (err: any) {
      setMessages(prev => [...prev, { role: 'system', content: `Slash command error: ${err}` }]);
    }
  }, [onLoadStatus]);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    const prompt = input.trim();
    if (!prompt || isLoading) return;

    const isSlash = await ChatService.IsSlashCommand(prompt);
    if (isSlash) {
      setInput('');
      const [name, args] = await ChatService.ParseSlashCommand(prompt);
      if (name) {
        await executeSlashCommand(name, args);
      }
      return;
    }

    setInput('');

    // Show STOP button immediately — don't wait for the LLM to start streaming.
    setIsLoading(true);
    // Re-anchor the seq dispatcher: each SendMessage starts a fresh stream, and
    // events from the previous run must not wedge this one's ordering.
    resetStreamOrderRef.current?.();

    // Build display message with file indicators
    const fileRefs = getFileRefs?.() || [];
    const displayPrefix = fileRefs.length > 0
      ? fileRefs.map(f => `@${f.name}`).join(' ') + '\n'
      : '';

    setMessages(prev => [...prev, { role: 'user', content: displayPrefix + prompt }]);

    try {
      if (fileRefs.length > 0) {
        // Send with file context
        const pathsJSON = JSON.stringify(fileRefs.map(f => f.path));
        await ChatService.SendMessageWithContext(prompt, pathsJSON);
        clearFileRefs?.();
      } else {
        await ChatService.SendMessage(prompt);
      }
    } catch (err: any) {
      setMessages(prev => [...prev, { role: 'system', content: `Error: ${err}` }]);
      setIsLoading(false);
    }
  }, [input, isLoading, executeSlashCommand, getFileRefs, clearFileRefs]);

  const handleNewChat = useCallback(async () => {
    ChatService.StartNewConversation();
    setMessages([]);
    setInput('');
    setIsLoading(false);
    ChatService.SetMode('act').catch(() => {});
    onLoadStatus();
  }, [onLoadStatus]);

  const selectProjectDir = useCallback(async () => {
    const dir = await ChatService.SelectProjectDir();
    if (dir) {
      onLoadStatus();
    }
    return dir;
  }, [onLoadStatus]);

  const handleFollowupAnswer = useCallback(async (answer: string) => {
    setFollowup(null);
    try {
      await ChatService.AnswerFollowupQuestion(answer);
    } catch (e) {
      console.error('Failed to send followup answer:', e);
    }
  }, []);

  const stopMessage = useCallback(() => {
    ChatService.StopMessage();
  }, []);

  const setupEventListeners = useCallback(() => {
    // Wails v3 dispatches each Go Emit on its own goroutine, so stream
    // events (content deltas especially) can arrive out of order — the
    // symptom was scrambled message text ("语序错乱") while the persisted
    // transcript stayed clean. Every stream event carries a monotonic seq
    // (guiStreamCallback.emit); this dispatcher applies handlers strictly
    // in seq order, stashing early arrivals until their predecessors land.
    // It also swallows stale duplicates (double listener registration).
    let expected = -1;
    const stash = new Map<number, () => void>();
    const ordered = (seq: number, fn: () => void) => {
      if (expected === -1) expected = seq; // first event after (re)load anchors
      if (seq === expected) {
        fn();
        expected++;
        while (stash.has(expected)) {
          const f = stash.get(expected)!;
          stash.delete(expected);
          f();
          expected++;
        }
      } else if (seq > expected) {
        stash.set(seq, fn);
      }
    };
    // Re-anchor before each user send: the Go side starts a fresh stream and
    // (before the ChatService-level seq counter fix) reused low seq numbers,
    // which this dispatcher would treat as stale duplicates and swallow —
    // the UI then froze on "AI is thinking..." while the run completed and
    // persisted fine. Clearing `expected` lets the first event of the new
    // run re-anchor, and clearing the stash drops any stale leftovers.
    resetStreamOrderRef.current = () => {
      expected = -1;
      stash.clear();
    };

    Events.On('chat:streamStart', (ev: any) => {
      ordered(ev?.data?.seq ?? 0, () => {
        setIsLoading(true);
        setMessages(prev => {
          // Reasoning deltas usually arrive before any text content and have
          // already opened the streaming assistant bubble below — don't add
          // a second, empty one.
          const last = prev[prev.length - 1];
          if (last && last.role === 'assistant' && last.streaming) {
            return prev;
          }
          return [...prev, { role: 'assistant', content: '', streaming: true }];
        });
      });
    });

    Events.On('chat:content', (ev: any) => {
      const seq = ev?.data?.seq ?? 0;
      const delta = ev?.data?.delta ?? '';
      ordered(seq, () => {
        setMessages(prev => {
          const last = prev[prev.length - 1];
          if (last && last.role === 'assistant' && last.streaming) {
            return [...prev.slice(0, -1), { ...last, content: last.content + delta }];
          }
          return prev;
        });
      });
    });

    Events.On('chat:reasoning', (ev: any) => {
      const seq = ev?.data?.seq ?? 0;
      const delta = ev?.data?.delta ?? '';
      ordered(seq, () => {
        setMessages(prev => {
          const last = prev[prev.length - 1];
          if (last && last.role === 'assistant' && last.streaming) {
            return [...prev.slice(0, -1), { ...last, thinking: (last.thinking || '') + delta }];
          }
          // Reasoning arrives before the bridge opens the text slot (streamStart
          // only fires on model-authored text) — open the streaming assistant
          // bubble here so thinking is visible while the model reasons.
          return [...prev, { role: 'assistant', content: '', thinking: delta, streaming: true }];
        });
      });
    });

    Events.On('chat:toolStart', (ev: any) => {
      const { seq, id, name, input: toolInput } = ev?.data ?? {};
      ordered(seq ?? 0, () => {
        setMessages(prev => {
          const last = prev[prev.length - 1];
          // A thinking bubble is worth keeping visible next to the tool row;
          // only an empty text-less bubble gets replaced by the tool row.
          const hasThinking = !!(last && last.role === 'assistant' && last.streaming && last.thinking && last.thinking.trim());
          if (hasThinking) {
            return [...prev, { role: 'tool', id, toolName: name, toolInput, content: '' }];
          }
          if (last && last.role === 'assistant' && last.content.trim() === '' && last.streaming) {
            return [...prev.slice(0, -1), { role: 'tool', id, toolName: name, toolInput, content: '' }];
          }
          return [...prev, { role: 'tool', id, toolName: name, toolInput, content: '' }];
        });
      });
    });

    Events.On('chat:toolComplete', (ev: any) => {
      const { seq, id, name, result } = ev?.data ?? {};
      ordered(seq ?? 0, () => {
        setMessages(prev => {
          const updated = prev.map(m => (m.id === id ? { ...m, toolResult: result } : m));
          // For attempt_completion, also insert the result as an assistant message
          // so the user can see the final summary without expanding tool details.
          if (name === 'attempt_completion' && result) {
            return [...updated, { role: 'assistant', content: result }];
          }
          return updated;
        });
        if (name === 'attempt_completion') {
          setIsLoading(false);
        }
        onLoadStatus();
      });
    });

    // streamEnd closes the current assistant text slot (emitted by the bridge
    // after each non-partial model turn).  The frontend treats this as the
    // signal to stop showing the streaming cursor on the current message.
    Events.On('chat:streamEnd', (ev: any) => {
      const seq = ev?.data?.seq ?? 0;
      ordered(seq, () => {
        setMessages(prev => {
          // At most one slot should be open; closing every streaming assistant
          // message is the safe behaviour and heals any earlier turn that
          // missed its streamEnd.
          let changed = false;
          const updated = prev.map(m => {
            if (m.role === 'assistant' && m.streaming) {
              changed = true;
              return { ...m, streaming: false };
            }
            return m;
          });
          return changed ? updated : prev;
        });
      });
    });

    Events.On('chat:error', (data: any) => {
      const err = data?.data ?? 'Unknown error';
      setIsLoading(false);
      setMessages(prev => {
        // Finalize any in-progress streaming assistant message
        const updated = prev.map(m =>
          m.role === 'assistant' && m.streaming ? { ...m, streaming: false } : m
        );
        return [...updated, { role: 'system', content: `Error: ${err}` }];
      });
    });

    Events.On('chat:complete', (ev: any) => {
      ordered(ev?.data?.seq ?? 0, () => {
        setIsLoading(false);
        setMessages(prev => {
          // Finalize every streaming assistant message — earlier turns that
          // ended in a tool call never get their own streamEnd-driven close
          // and would otherwise leave the thinking indicator pulsing forever.
          const updated = prev.map(m =>
            m.role === 'assistant' && m.streaming ? { ...m, streaming: false } : m
          );
          const last = updated[updated.length - 1];
          // Drop a trailing bubble that carries nothing at all; keep it when
          // it has thinking (that is the visible record of the turn).
          if (
            last &&
            last.role === 'assistant' &&
            !last.content.trim() &&
            !(last.thinking && last.thinking.trim())
          ) {
            return updated.slice(0, -1);
          }
          return updated;
        });
        onLoadHistory();
        onLoadStatus();
      });
    });

    Events.On('chat:taskCreated', () => {
      onLoadHistory();
    });

    Events.On('chat:systemMessage', (ev: any) => {
      const seq = ev?.data?.seq ?? 0;
      const content = ev?.data?.content ?? ev?.data ?? '';
      ordered(seq, () => {
        if (content) {
          setMessages(prev => [...prev, { role: 'system', content } as Message]);
        }
      });
    });

    Events.On('chat:followupQuestion', (ev: any) => {
      const seq = ev?.data?.seq ?? 0;
      const q = ev?.data?.question ?? '';
      const opts = (ev?.data?.options as string[]) || [];
      ordered(seq, () => {
        setFollowup({ question: q, options: opts });
      });
    });

    WML.Reload();
  }, [onLoadHistory, onLoadStatus]);

  return {
    messages,
    setMessages,
    input,
    setInput,
    isLoading,
    messagesEndRef,
    followup,
    handleSubmit,
    handleNewChat,
    selectProjectDir,
    handleFollowupAnswer,
    stopMessage,
    setupEventListeners,
  };
}

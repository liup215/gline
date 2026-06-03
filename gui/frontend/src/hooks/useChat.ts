import { useState, useCallback, useRef } from 'react';
import { Events, WML } from '@wailsio/runtime';
import { ChatService } from '../../bindings/github.com/liup215/gline/gui';
import { Message } from '../types';

export function useChat(onLoadHistory: () => void, onLoadStatus: () => void) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const [followup, setFollowup] = useState<{ question: string; options: string[] } | null>(null);

  const executeSlashCommand = useCallback(async (name: string, args: string) => {
    try {
      const result: any = await ChatService.ExecuteSlashCommand(name, args);
      const action = result?.action || 'none';
      const msg = result?.message || '';

      switch (action) {
        case 'clear': {
          ChatService.StartNewConversation();
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
    setMessages(prev => [...prev, { role: 'user', content: prompt }]);

    try {
      await ChatService.SendMessage(prompt);
    } catch (err: any) {
      setMessages(prev => [...prev, { role: 'system', content: `Error: ${err}` }]);
      setIsLoading(false);
    }
  }, [input, isLoading, executeSlashCommand]);

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
        const last = prev[prev.length - 1];
        if (last && last.role === 'assistant' && last.content.trim() === '' && last.streaming) {
          return [...prev.slice(0, -1), { role: 'tool', id, toolName: name, toolInput, content: '' }];
        }
        return [...prev, { role: 'tool', id, toolName: name, toolInput, content: '' }];
      });
    });

    Events.On('chat:toolComplete', (data: any) => {
      const { id, name, result } = data?.data ?? {};
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
          if (!last.content.trim()) {
            return prev.slice(0, -1);
          }
          return [...prev.slice(0, -1), { ...last, streaming: false }];
        }
        return prev;
      });
      onLoadHistory();
      onLoadStatus();
    });

    Events.On('chat:taskCreated', () => {
      onLoadHistory();
    });

    Events.On('chat:systemMessage', (data: any) => {
      const content = data?.data?.content ?? data?.data ?? '';
      if (content) {
        setMessages(prev => [...prev, { role: 'system', content } as Message]);
      }
    });

    Events.On('chat:followupQuestion', (data: any) => {
      const q = data?.data?.question ?? '';
      const opts = (data?.data?.options as string[]) || [];
      setFollowup({ question: q, options: opts });
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

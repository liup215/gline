import { useEffect, useRef } from 'react';
import { Message } from '../types';
import { UserMessage } from './UserMessage';
import { AssistantMessage } from './AssistantMessage';
import { ToolMessage } from './ToolMessage';
import { SystemMessage } from './SystemMessage';
import { useHighlightCode } from '../utils/format';

interface MessageListProps {
  messages: Message[];
  showSelectDir?: boolean;
  onSelectProjectDir?: () => void;
}

export function MessageList({ messages, showSelectDir, onSelectProjectDir }: MessageListProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useHighlightCode();

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const renderMessage = (msg: Message, idx: number) => {
    // Skip assistant messages that have no content and are pure tool-call prompts
    if (msg.role === 'assistant' && msg.content.trim() === '') {
      return null;
    }
    if (msg.role === 'user') {
      return <UserMessage key={idx} content={msg.content} />;
    }
    if (msg.role === 'assistant') {
      return <AssistantMessage key={idx} content={msg.content} streaming={msg.streaming} isLast={idx === messages.length - 1} />;
    }
    if (msg.role === 'tool') {
      return <ToolMessage key={idx} toolName={msg.toolName} toolInput={msg.toolInput} toolResult={msg.toolResult} />;
    }
    if (msg.role === 'system') {
      return <SystemMessage key={idx} content={msg.content} />;
    }
    return null;
  };

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '20px 0', display: 'flex', flexDirection: 'column' }}>
      {messages.length === 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', flex: 1, color: '#64748b', gap: '18px' }}>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '6px' }}>
            <h2 style={{ margin: '0 0 8px', fontWeight: 400, fontSize: '1.5rem', color: '#94a3b8' }}>Welcome to gline</h2>
            <p style={{ margin: 0, fontSize: '0.9rem' }}>AI Programming Assistant powered by Go</p>
          </div>
          {showSelectDir && onSelectProjectDir && (
            <button
              onClick={onSelectProjectDir}
              style={{
                padding: '12px 28px',
                borderRadius: '10px',
                border: 'none',
                background: '#3b82f6',
                color: '#fff',
                fontSize: '0.95rem',
                fontWeight: 600,
                cursor: 'pointer',
                transition: 'background 0.2s',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = '#2563eb')}
              onMouseLeave={(e) => (e.currentTarget.style.background = '#3b82f6')}
            >
              <span style={{ fontSize: '1.1rem' }}>📁</span> Select Project Directory
            </button>
          )}
        </div>
      )}
      {messages.map((msg, idx) => renderMessage(msg, idx))}
      <div ref={messagesEndRef} />
    </div>
  );
}

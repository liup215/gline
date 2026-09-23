import { useState, useEffect, useRef } from 'react';
import { THEME } from '../theme';
import { formatContent } from '../utils/format';

interface AssistantMessageProps {
  content: string;
  thinking?: string;
  streaming: boolean | undefined;
  isLast: boolean;
}

export function AssistantMessage({ content, thinking, streaming, isLast }: AssistantMessageProps) {
  const hasThinking = !!(thinking && thinking.trim());
  const hasContent = !!content.trim();
  // While the model is still thinking (streaming, no answer text yet), show
  // the reasoning live with a pulsing indicator; once answer text starts,
  // collapse it back behind the toggle.
  const liveThinking = !!streaming && hasThinking && !hasContent;
  const [showThinking, setShowThinking] = useState(false);
  const thinkingBoxRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (liveThinking && thinkingBoxRef.current) {
      thinkingBoxRef.current.scrollTop = thinkingBoxRef.current.scrollHeight;
    }
  }, [thinking, liveThinking]);

  const expanded = showThinking || liveThinking;

  return (
    <div style={{ display: 'flex', justifyContent: 'flex-start', padding: '0 24px', marginBottom: '16px' }}>
      <div style={{ maxWidth: '85%', background: THEME.assistantBg, color: THEME.text, padding: '14px 20px', borderRadius: '18px 18px 18px 4px', lineHeight: 1.6, fontSize: '0.95rem', border: `1px solid ${THEME.border}`, boxShadow: '0 2px 8px rgba(0,0,0,0.15)' }}>
        {hasThinking && (
          <div style={{ marginBottom: hasContent ? '10px' : 0 }}>
            <button
              onClick={() => setShowThinking(!showThinking)}
              style={{
                background: 'none', border: 'none', color: THEME.textDim, cursor: 'pointer',
                fontSize: '0.8rem', padding: '2px 0', display: 'flex', alignItems: 'center', gap: '4px',
              }}
            >
              <span style={{ transition: 'transform 0.2s', display: 'inline-block', transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)' }}>▶</span>
              Thinking
              {liveThinking && (
                <span style={{ display: 'inline-flex', gap: '3px', marginLeft: '2px', alignItems: 'center' }}>
                  {[0, 1, 2].map(i => (
                    <span key={i} style={{
                      width: 4, height: 4, borderRadius: '50%', background: THEME.accent,
                      animation: `blink 1.2s ${i * 0.2}s infinite`,
                    }} />
                  ))}
                </span>
              )}
            </button>
            {expanded && (
              <div
                ref={thinkingBoxRef}
                style={{
                  marginTop: '6px', padding: '10px 14px', background: 'rgba(128,128,128,0.08)', borderRadius: '8px',
                  fontSize: '0.85rem', color: THEME.textDim, lineHeight: 1.5, whiteSpace: 'pre-wrap',
                  maxHeight: '300px', overflowY: 'auto',
                }}
              >
                {thinking}
              </div>
            )}
          </div>
        )}
        <div className="md-rendered" dangerouslySetInnerHTML={{ __html: formatContent(content) }} />
        {streaming && isLast && <span style={{ color: THEME.accent, animation: 'blink 1s infinite' }}>▌</span>}
      </div>
    </div>
  );
}

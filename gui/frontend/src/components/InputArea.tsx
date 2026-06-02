import { THEME } from '../theme';
import { AppStatus } from '../types';
import { SlashMenu, SlashMenuState } from '../slash';

interface InputAreaProps {
  input: string;
  setInput: (v: string) => void;
  isLoading: boolean;
  onSubmit: (e: React.FormEvent) => void;
  menuState: SlashMenuState;
  handleInputChange: (text: string, cursorPos: number, setInputValue: (v: string) => void, inputEl: HTMLInputElement | null) => void;
  handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>, setInputValue: (v: string) => void) => { handled: boolean };
  selectCommand: (cmd: any, setInputValue: (v: string) => void, inputEl: HTMLInputElement | null) => void;
  closeMenu: () => void;
  status: AppStatus;
  mode: 'plan' | 'act';
  onToggleMode: () => void;
  chatInputRef: React.MutableRefObject<HTMLInputElement | null>;
}

export function InputArea({
  input, setInput, isLoading, onSubmit,
  menuState, handleInputChange, handleKeyDown, selectCommand, closeMenu,
  status, mode, onToggleMode, chatInputRef,
}: InputAreaProps) {
  return (
    <div style={{ padding: '14px 24px 20px', borderTop: `1px solid ${THEME.border}`, background: THEME.bg, position: 'relative' }}>
      <form style={{ display: 'flex', gap: '12px' }} onSubmit={onSubmit}>
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
                onSubmit(e as any);
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

      {/* Hint Row */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '6px',
        marginTop: '8px',
        fontSize: '0.72rem',
        color: '#636e7b',
      }}>
        Type <span style={{ color: '#8B5CF6', fontWeight: 600 }}>/</span> for slash commands · Use <span style={{ color: '#8B5CF6', fontWeight: 600 }}>@</span> to add files
      </div>

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
          onClick={onToggleMode}
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
  );
}

import { THEME } from '../theme';
import { AppStatus, FileRef } from '../types';
import { SlashMenu, SlashMenuState } from '../slash';
import { FilePicker } from './FilePicker';
import type { FileEntry, FilePickerState } from '../hooks/useFileReference';

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
  canChat?: boolean;
  // File reference props
  selectedFiles: FileRef[];
  onRemoveFile: (path: string) => void;
  filePickerState: FilePickerState;
  onFileSelect: (entry: FileEntry) => void;
  onFilePickerKeyDown: (e: React.KeyboardEvent) => { handled: boolean };
  onFilePickerQueryChange: (query: string) => void;
  onOpenFilePicker: () => void;
  onCloseFilePicker: () => void;
}

export function InputArea({
  input, setInput, isLoading, onSubmit,
  menuState, handleInputChange, handleKeyDown, selectCommand, closeMenu,
  status, mode, onToggleMode, chatInputRef, canChat = true,
  selectedFiles, onRemoveFile,
  filePickerState, onFileSelect, onFilePickerKeyDown, onFilePickerQueryChange,
  onOpenFilePicker, onCloseFilePicker,
}: InputAreaProps) {
  // Detect @ trigger in input change
  const handleInputChangeWithAt = (text: string, cursorPos: number, setInputValue: (v: string) => void, inputEl: HTMLInputElement | null) => {
    // First, let slash menu handle its logic
    handleInputChange(text, cursorPos, setInputValue, inputEl);

    // Detect @ character just typed
    if (cursorPos > 0 && text[cursorPos - 1] === '@') {
      const beforeAt = text.slice(0, cursorPos - 1);
      // Only trigger if @ is at start or after whitespace
      if (beforeAt === '' || beforeAt.endsWith(' ') || beforeAt.endsWith('\n')) {
        // Don't trigger if it's a slash command context
        if (!beforeAt.includes('/') || beforeAt.lastIndexOf('/') < beforeAt.lastIndexOf(' ')) {
          // Remove the trailing @ and open file picker
          const cleaned = text.slice(0, cursorPos - 1) + text.slice(cursorPos);
          setInputValue(cleaned);
          onOpenFilePicker();
        }
      }
    }
  };

  const handleKeyDownWithPicker = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // If file picker is active, handle its key events first
    if (filePickerState.active) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === 'Escape') {
        const { handled } = onFilePickerKeyDown(e);
        if (handled) return;
      }
      if (e.key === 'Backspace' && input === '') {
        const { handled } = onFilePickerKeyDown(e);
        if (handled) return;
      }
    }

    // Slash menu handling
    const { handled } = handleKeyDown(e, setInput);
    if (handled) return;

    if (e.key === 'Tab' && !e.shiftKey) {
      e.preventDefault();
      onToggleMode();
      return;
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSubmit(e as any);
    }
  };

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
          <FilePicker
            active={filePickerState.active}
            entries={filePickerState.entries}
            currentPath={filePickerState.currentPath}
            loading={filePickerState.loading}
            selectedIndex={filePickerState.selectedIndex}
            query={filePickerState.query}
            onSelect={onFileSelect}
            onNavigate={(path) => {
              // navigateTo equivalent - this is handled through selectEntry for dirs
              // but we expose a dedicated prop for consistency
              // Currently navigation is done via selectEntry detecting isDir
            }}
            onQueryChange={onFilePickerQueryChange}
            onClose={onCloseFilePicker}
            onKeyDown={onFilePickerKeyDown}
          />

          {/* File tags above input */}
          {selectedFiles.length > 0 && (
            <div style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: '6px',
              padding: '8px 12px 0',
            }}>
              {selectedFiles.map(file => {
                const displayName = file.path.includes('/') ? file.path : file.name;
                const shortName = displayName.split('/').slice(-2).join('/');
                return (
                <span
                  key={file.path}
                  title={file.path}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '4px',
                    padding: '3px 8px',
                    borderRadius: '6px',
                    background: 'rgba(59,130,246,0.15)',
                    border: '1px solid rgba(59,130,246,0.3)',
                    color: '#93c5fd',
                    fontSize: '0.78rem',
                    whiteSpace: 'nowrap',
                    maxWidth: '240px',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  📄 {shortName}
                  <button
                    type="button"
                    onClick={() => onRemoveFile(file.path)}
                    style={{
                      background: 'transparent',
                      border: 'none',
                      color: '#93c5fd',
                      cursor: 'pointer',
                      fontSize: '0.85rem',
                      padding: '0 2px',
                      lineHeight: 1,
                    }}
                  >
                    ✕
                  </button>
                </span>
                );
              })}
            </div>
          )}

          <input
            ref={chatInputRef}
            style={{ width: '100%', padding: '12px 16px', borderRadius: '10px', border: `1px solid ${THEME.border}`, background: '#1e293b', color: THEME.text, fontSize: '0.95rem', outline: 'none', transition: 'border-color 0.2s', boxSizing: 'border-box' }}
            value={input}
            onChange={e => {
              setInput(e.target.value);
              handleInputChangeWithAt(e.target.value, e.target.selectionStart || 0, setInput, e.target);
            }}
            onKeyDown={handleKeyDownWithPicker}
            onClick={e => {
              handleInputChange(input, e.currentTarget.selectionStart || 0, setInput, e.currentTarget);
            }}
            placeholder={!canChat ? 'Please select a project directory first' : isLoading ? 'AI is thinking...' : 'Ask gline anything... Type / for commands, @ for files'}
            disabled={isLoading || !canChat}
            onFocus={e => e.currentTarget.style.borderColor = THEME.accent}
            onBlur={e => {
              e.currentTarget.style.borderColor = THEME.border;
              setTimeout(() => closeMenu(), 200);
            }}
          />
        </div>
        <button
          type="submit"
          style={{ padding: '12px 20px', borderRadius: '10px', border: 'none', background: isLoading || !input.trim() || !canChat ? '#334155' : THEME.accent, color: '#fff', cursor: isLoading || !input.trim() || !canChat ? 'not-allowed' : 'pointer', fontSize: '0.95rem', fontWeight: 500, transition: 'background 0.2s' }}
          disabled={isLoading || !input.trim() || !canChat}
        >
          {isLoading ? '⏳' : (!canChat ? 'Select Folder' : 'Send')}
        </button>
      </form>

      {/* Hint Row */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-start',
        gap: '6px',
        marginTop: '8px',
        fontSize: '0.72rem',
        color: '#636e7b',
      }}>
        Type <span style={{ color: '#8B5CF6', fontWeight: 600 }}>/</span> for slash commands · <span style={{ color: '#3b82f6', fontWeight: 600 }}>@</span> for files
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

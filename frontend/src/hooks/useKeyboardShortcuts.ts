import { useEffect } from 'react';

interface ShortcutHandlers {
  onNewChat: () => void;
  onFocusInput: () => void;
  onToggleSidebar: () => void;
}

export function useKeyboardShortcuts({ onNewChat, onFocusInput, onToggleSidebar }: ShortcutHandlers) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey) {
        if (e.key === 'n' || e.key === 'N') {
          e.preventDefault();
          onNewChat();
        }
        if (e.key === 'k' || e.key === 'K') {
          e.preventDefault();
          onFocusInput();
        }
        if (e.key === 'b' || e.key === 'B') {
          e.preventDefault();
          onToggleSidebar();
        }
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onNewChat, onFocusInput, onToggleSidebar]);
}

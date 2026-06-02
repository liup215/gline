import { useState, useEffect, useCallback, useRef } from 'react';
import { ChatService } from '../../bindings/github.com/liup215/gline/gui';

export interface SlashCommand {
  name: string;
  description: string;
  section: string;
}

export interface SlashMenuState {
  active: boolean;
  query: string;
  filtered: SlashCommand[];
  selectedIndex: number;
  allCommands: SlashCommand[];
}

export function useSlashCommands() {
  const [menuState, setMenuState] = useState<SlashMenuState>({
    active: false,
    query: '',
    filtered: [],
    selectedIndex: 0,
    allCommands: [],
  });

  const inputRef = useRef<HTMLInputElement | null>(null);

  // Load all slash commands on mount
  useEffect(() => {
    ChatService.GetSlashCommands()
      .then((cmds: any[]) => {
        const mapped = cmds.map((c: any) => ({
          name: c.name || '',
          description: c.description || '',
          section: c.section || 'default',
        }));
        setMenuState(prev => ({ ...prev, allCommands: mapped }));
      })
      .catch(() => {
        // silently fail; slash commands just won't be available
      });
  }, []);

  const isSlashPrefix = useCallback((text: string, cursorPos: number): boolean => {
    if (cursorPos < 1) return false;
    const before = text.slice(0, cursorPos);
    const lastSlash = before.lastIndexOf('/');
    if (lastSlash < 0) return false;
    const afterSlash = before.slice(lastSlash + 1);
    if (/\s/.test(afterSlash)) return false;
    if (lastSlash > 0) {
      const prev = before[lastSlash - 1];
      if (prev !== ' ' && prev !== '\t' && prev !== '\n') return false;
    }
    return true;
  }, []);

  const filterCommands = useCallback((query: string, all: SlashCommand[]): SlashCommand[] => {
    const q = query.toLowerCase();
    const filtered = all.filter(cmd => cmd.name.toLowerCase().startsWith(q));
    // Sort: custom section first, then by name
    filtered.sort((a, b) => {
      if (a.section !== b.section) {
        return a.section === 'custom' ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });
    return filtered;
  }, []);

  const handleInputChange = useCallback((
    text: string,
    cursorPos: number,
    setInputValue: (v: string) => void,
    inputEl: HTMLInputElement | null
  ) => {
    inputRef.current = inputEl;

    if (!isSlashPrefix(text, cursorPos)) {
      setMenuState(prev => {
        if (!prev.active) return prev;
        return { ...prev, active: false, filtered: [], query: '' };
      });
      return;
    }

    const before = text.slice(0, cursorPos);
    const lastSlash = before.lastIndexOf('/');
    const query = before.slice(lastSlash + 1);

    setMenuState(prev => {
      const filtered = filterCommands(query, prev.allCommands);
      return {
        ...prev,
        active: true,
        query,
        filtered,
        selectedIndex: 0,
      };
    });
  }, [isSlashPrefix, filterCommands]);

  const handleKeyDown = useCallback((
    e: React.KeyboardEvent<HTMLInputElement>,
    setInputValue: (v: string) => void
  ): { handled: boolean; selectedCommand?: SlashCommand } => {
    if (!menuState.active || menuState.filtered.length === 0) {
      return { handled: false };
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setMenuState(prev => ({
        ...prev,
        selectedIndex: (prev.selectedIndex + 1) % prev.filtered.length,
      }));
      return { handled: true };
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault();
      setMenuState(prev => ({
        ...prev,
        selectedIndex: (prev.selectedIndex - 1 + prev.filtered.length) % prev.filtered.length,
      }));
      return { handled: true };
    }

    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault();
      const cmd = menuState.filtered[menuState.selectedIndex];
      if (cmd) {
        setMenuState(prev => ({ ...prev, active: false, filtered: [], query: '' }));
        // Replace input: everything before the last slash + command name + space
        const inputEl = inputRef.current;
        if (inputEl) {
          const text = inputEl.value;
          const beforeSlash = text.slice(0, text.lastIndexOf('/'));
          // Only if at very start of input, replace whole input; otherwise append after
          const newVal = beforeSlash === '' ? `/${cmd.name} ` : `${beforeSlash}/${cmd.name} `;
          setInputValue(newVal);
          // Refocus and set cursor at end
          setTimeout(() => {
            inputEl.focus();
            inputEl.setSelectionRange(newVal.length, newVal.length);
          }, 0);
        }
        return { handled: true, selectedCommand: cmd };
      }
    }

    if (e.key === 'Escape') {
      e.preventDefault();
      setMenuState(prev => ({ ...prev, active: false, filtered: [], query: '' }));
      return { handled: true };
    }

    return { handled: false };
  }, [menuState.active, menuState.filtered, menuState.selectedIndex]);

  const selectCommand = useCallback((
    cmd: SlashCommand,
    setInputValue: (v: string) => void,
    inputEl: HTMLInputElement | null
  ) => {
    setMenuState(prev => ({ ...prev, active: false, filtered: [], query: '' }));
    if (inputEl) {
      const text = inputEl.value;
      const lastSlash = text.lastIndexOf('/');
      const beforeSlash = text.slice(0, lastSlash);
      const newVal = beforeSlash === '' ? `/${cmd.name} ` : `${beforeSlash}/${cmd.name} `;
      setInputValue(newVal);
      setTimeout(() => {
        inputEl.focus();
        inputEl.setSelectionRange(newVal.length, newVal.length);
      }, 0);
    }
  }, []);

  const closeMenu = useCallback(() => {
    setMenuState(prev => ({ ...prev, active: false, filtered: [], query: '' }));
  }, []);

  return {
    menuState,
    handleInputChange,
    handleKeyDown,
    selectCommand,
    closeMenu,
  };
}

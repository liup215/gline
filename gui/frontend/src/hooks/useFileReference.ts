import { useState, useCallback } from 'react';
import { ChatService } from '../../bindings/github.com/liup215/gline/gui';
import type { FileRef } from '../types';

export interface FileEntry {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  modTime: number;
}

export interface FilePickerState {
  active: boolean;
  entries: FileEntry[];
  currentPath: string;
  loading: boolean;
  query: string;
  selectedIndex: number;
}

export function useFileReference() {
  const [pickerState, setPickerState] = useState<FilePickerState>({
    active: false,
    entries: [],
    currentPath: '',
    loading: false,
    query: '',
    selectedIndex: 0,
  });

  const [selectedFiles, setSelectedFiles] = useState<FileRef[]>([]);

  const loadEntries = useCallback(async (dirPath: string) => {
    setPickerState(prev => ({ ...prev, loading: true }));
    try {
      const entries = await ChatService.ListDirEntries(dirPath);
      setPickerState(prev => ({
        ...prev,
        entries: entries || [],
        currentPath: dirPath,
        loading: false,
        query: '',
        selectedIndex: 0,
      }));
    } catch (err) {
      console.error('Failed to list directory:', err);
      setPickerState(prev => ({
        ...prev,
        entries: [],
        loading: false,
        query: '',
        selectedIndex: 0,
      }));
    }
  }, []);

  const openPicker = useCallback(async () => {
    setPickerState(prev => ({ ...prev, active: true }));
    await loadEntries('');
  }, [loadEntries]);

  const closePicker = useCallback(() => {
    setPickerState(prev => ({ ...prev, active: false, entries: [], query: '', selectedIndex: 0 }));
  }, []);

  const navigateTo = useCallback(async (dirPath: string) => {
    await loadEntries(dirPath);
  }, [loadEntries]);

  const navigateUp = useCallback(async () => {
    const current = pickerState.currentPath;
    if (!current) {
      closePicker();
      return;
    }
    // Go up one level
    const parts = current.split('/');
    parts.pop();
    const parentPath = parts.join('/');
    await loadEntries(parentPath);
  }, [pickerState.currentPath, loadEntries, closePicker]);

  const selectEntry = useCallback((entry: FileEntry) => {
    if (entry.isDir) {
      navigateTo(entry.path);
    } else {
      // Add file to selected list (deduplicate)
      setSelectedFiles(prev => {
        if (prev.some(f => f.path === entry.path)) return prev;
        return [...prev, { path: entry.path, name: entry.name, isDir: false }];
      });
    }
  }, [navigateTo]);

  const addFile = useCallback((file: FileRef) => {
    setSelectedFiles(prev => {
      if (prev.some(f => f.path === file.path)) return prev;
      return [...prev, file];
    });
  }, []);

  const removeFile = useCallback((path: string) => {
    setSelectedFiles(prev => prev.filter(f => f.path !== path));
  }, []);

  const clearFiles = useCallback(() => {
    setSelectedFiles([]);
  }, []);

  const setPickerQuery = useCallback((query: string) => {
    setPickerState(prev => {
        // Compute the visible list for selectedIndex clamping
      const maxIdx = Math.max(0, getFilteredEntries(prev.entries, query).length - 1);
      return {
        ...prev,
        query,
        selectedIndex: prev.selectedIndex > maxIdx ? 0 : prev.selectedIndex,
      };
    });
  }, []);

  const handlePickerKeyDown = useCallback((
    e: React.KeyboardEvent,
    onSelectEntry?: (entry: FileEntry) => void,
  ): { handled: boolean } => {
    if (!pickerState.active) return { handled: false };

    const filtered = getFilteredEntries(pickerState.entries, pickerState.query);
    if (filtered.length === 0) {
      if (e.key === 'Escape') {
        e.preventDefault();
        closePicker();
        return { handled: true };
      }
      return { handled: false };
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setPickerState(prev => ({
        ...prev,
        selectedIndex: (prev.selectedIndex + 1) % Math.max(1, filtered.length),
      }));
      return { handled: true };
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault();
      setPickerState(prev => ({
        ...prev,
        selectedIndex: (prev.selectedIndex - 1 + Math.max(1, filtered.length)) % Math.max(1, filtered.length),
      }));
      return { handled: true };
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      if (filtered.length > 0) {
        const entry = filtered[pickerState.selectedIndex];
        if (entry) {
          selectEntry(entry);
          onSelectEntry?.(entry);
          if (!entry.isDir) {
            closePicker();
          }
        }
      }
      return { handled: true };
    }

    if (e.key === 'Escape') {
      e.preventDefault();
      closePicker();
      return { handled: true };
    }

    if (e.key === 'Backspace' && pickerState.query === '') {
      e.preventDefault();
      navigateUp();
      return { handled: true };
    }

    return { handled: false };
  }, [pickerState.active, pickerState.entries, pickerState.query, pickerState.selectedIndex, selectEntry, closePicker, navigateUp]);

  // Get filtered entries based on current query
  const getFiltered = useCallback((): FileEntry[] => {
    return getFilteredEntries(pickerState.entries, pickerState.query);
  }, [pickerState.entries, pickerState.query]);

  return {
    pickerState,
    selectedFiles,
    openPicker,
    closePicker,
    navigateTo,
    navigateUp,
    selectEntry,
    addFile,
    removeFile,
    clearFiles,
    setPickerQuery,
    handlePickerKeyDown,
    getFiltered,
  };
}

function getFilteredEntries(entries: FileEntry[], query: string): FileEntry[] {
  if (!query) return entries;
  const q = query.toLowerCase();
  return entries.filter(e => e.name.toLowerCase().includes(q));
}

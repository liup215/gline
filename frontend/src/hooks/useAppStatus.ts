import { useState, useCallback } from 'react';
import { ChatService } from '../../bindings/github.com/liup215/gline/internal/gui';
import { AppStatus } from '../types';

export function useAppStatus() {
  const [mode, setMode] = useState<'plan' | 'act'>('act');
  const [status, setStatus] = useState<AppStatus>({ provider: '', model: '', cwd: '', currentTokens: '0', maxTokens: '0' });

  const loadMode = useCallback(async () => {
    try {
      const currentMode = await ChatService.GetMode();
      setMode(currentMode === 'plan' ? 'plan' : 'act');
    } catch (err) {
      console.error('Failed to load mode:', err);
    }
  }, []);

  const loadStatus = useCallback(async () => {
    try {
      const s = await ChatService.GetStatus();
      const status = {
        provider: s.provider || '',
        model: s.model || '',
        cwd: s.cwd || '',
        currentTokens: s.currentTokens || '0',
        maxTokens: s.maxTokens || '0',
      };
      setStatus(status);
      return status;
    } catch (err) {
      console.error('Failed to load status:', err);
      return null;
    }
  }, []);

  const toggleMode = useCallback(async () => {
    const newMode = mode === 'act' ? 'plan' : 'act';
    try {
      await ChatService.SetMode(newMode);
      setMode(newMode);
    } catch (err) {
      console.error('Failed to set mode:', err);
    }
  }, [mode]);

  return {
    mode,
    status,
    loadMode,
    loadStatus,
    toggleMode,
  };
}

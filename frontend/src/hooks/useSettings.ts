import { useState, useCallback } from 'react';
import { ChatService } from '../../bindings/github.com/liup215/gline/internal/gui';

export interface RuleInfo {
  name: string;
  source: string;
  size: number;
  modTime: number;
}

export function useSettings() {
  const [showSettings, setShowSettings] = useState(false);
  const [configData, setConfigData] = useState<any>(null);
  const [saveMessage, setSaveMessage] = useState('');
  const [rules, setRules] = useState<RuleInfo[]>([]);
  const [rulesMessage, setRulesMessage] = useState('');
  const [loadingRules, setLoadingRules] = useState(false);

  const loadRules = useCallback(async () => {
    try {
      setLoadingRules(true);
      const infos = await ChatService.GetRulesInfo();
      setRules(infos || []);
    } catch (err) {
      console.error('Failed to load rules:', err);
      setRules([]);
    } finally {
      setLoadingRules(false);
    }
  }, []);

  const loadConfig = useCallback(async () => {
    try {
      const cfgJson = await ChatService.GetConfig();
      setConfigData(JSON.parse(cfgJson));
      // Auto-load rules info when settings opens
      loadRules();
    } catch (err) {
      console.error('Failed to load config:', err);
    }
  }, [loadRules]);

  const saveSettings = useCallback(async (updates: Record<string, string>) => {
    try {
      for (const [key, value] of Object.entries(updates)) {
        await ChatService.UpdateConfig(key, value as string);
      }
      setSaveMessage('Settings saved successfully!');
      setTimeout(() => setSaveMessage(''), 3000);
      loadConfig();
    } catch (err) {
      setSaveMessage('Failed to save settings');
    }
  }, [loadConfig]);

  const reloadRules = useCallback(async () => {
    try {
      setLoadingRules(true);
      setRulesMessage('');
      const [count] = await ChatService.ReloadRules();
      setRulesMessage(`✅ Reloaded ${count} rule file(s). Rules will apply to the next message.`);
      const infos = await ChatService.GetRulesInfo();
      setRules(infos || []);
      setTimeout(() => setRulesMessage(''), 4000);
    } catch (err: any) {
      setRulesMessage(`❌ Error: ${err?.message || 'Failed to reload rules'}`);
      console.error('Failed to reload rules:', err);
    } finally {
      setLoadingRules(false);
    }
  }, []);

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  };

  const formatModTime = (ts: number): string => {
    const d = new Date(ts * 1000);
    return d.toLocaleString();
  };

  return {
    showSettings,
    setShowSettings,
    configData,
    setConfigData,
    saveMessage,
    setSaveMessage,
    loadConfig,
    saveSettings,
    rules,
    rulesMessage,
    loadingRules,
    loadRules,
    reloadRules,
    formatFileSize,
    formatModTime,
  };
}

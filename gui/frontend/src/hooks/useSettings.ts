import { useState, useCallback } from 'react';
import { ChatService } from '../../bindings/github.com/liup215/gline/gui';

export function useSettings() {
  const [showSettings, setShowSettings] = useState(false);
  const [configData, setConfigData] = useState<any>(null);
  const [saveMessage, setSaveMessage] = useState('');

  const loadConfig = useCallback(async () => {
    try {
      const cfgJson = await ChatService.GetConfig();
      setConfigData(JSON.parse(cfgJson));
    } catch (err) {
      console.error('Failed to load config:', err);
    }
  }, []);

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

  return {
    showSettings,
    setShowSettings,
    configData,
    setConfigData,
    saveMessage,
    setSaveMessage,
    loadConfig,
    saveSettings,
  };
}

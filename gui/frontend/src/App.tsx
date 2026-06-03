import { useState, useRef, useEffect } from 'react';
import { THEME } from './theme';
import { useChat } from './hooks/useChat';
import { useTaskHistory } from './hooks/useTaskHistory';
import { useAppStatus } from './hooks/useAppStatus';
import { useSettings } from './hooks/useSettings';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useSlashCommands } from './slash';
import { Sidebar } from './components/Sidebar';
import { ChatArea } from './components/ChatArea';
import { SettingsPanel } from './components/SettingsPanel';
import { FollowupModal } from './components/FollowupModal';

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const chatInputRef = useRef<HTMLInputElement | null>(null);

  const appStatus = useAppStatus();
  const settings = useSettings();
  const tasks = useTaskHistory();
  const chat = useChat(tasks.loadHistory, appStatus.loadStatus);
  const slash = useSlashCommands();

  // Can chat when a project directory has been selected or there is message history
  const canChat = (appStatus.status.cwd || '') !== '' || chat.messages.length > 0;

  useEffect(() => {
    tasks.loadHistory();
    appStatus.loadMode();
    appStatus.loadStatus();
  }, []);

  useEffect(() => {
    chat.setupEventListeners();
  }, [chat.setupEventListeners]);

  const handleSelectTask = async (taskID: string) => {
    const result = await tasks.handleSelectTask(taskID);
    if (!result) return;
    chat.setMessages(result.messages);
    // Always refresh status so cwd reflects the loaded task's workingDir
    await appStatus.loadStatus();
  };

  const handleNewChat = () => {
    chat.handleNewChat();
    tasks.setActiveTaskID(null);
  };

  const handleSelectProjectDir = async () => {
    await chat.selectProjectDir();
    await appStatus.loadStatus();
  };

  const handleDeleteTask = async (e: React.MouseEvent, taskID: string) => {
    e.stopPropagation();
    if (!confirm('Delete this conversation?')) return;
    try {
      await tasks.handleDeleteTask(e, taskID);
      if (tasks.activeTaskID === taskID) {
        chat.setMessages([]);
        tasks.setActiveTaskID(null);
      }
    } catch (err) {
      console.error('Failed to delete task:', err);
    }
  };

  useKeyboardShortcuts({
    onNewChat: handleNewChat,
    onFocusInput: () => chatInputRef.current?.focus(),
    onToggleSidebar: () => setSidebarOpen(prev => !prev),
  });

  return (
    <div
      style={{
        display: 'flex',
        height: '100vh',
        width: '100vw',
        background: THEME.bg,
        color: THEME.text,
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
        overflow: 'hidden',
      }}
    >
      <Sidebar
        sidebarOpen={sidebarOpen}
        history={tasks.history}
        activeTaskID={tasks.activeTaskID}
        onNewChat={handleNewChat}
        onSelectTask={handleSelectTask}
        onDeleteTask={handleDeleteTask}
        onOpenSettings={() => { settings.setShowSettings(true); settings.loadConfig(); }}
      />
      <ChatArea
        sidebarOpen={sidebarOpen}
        setSidebarOpen={setSidebarOpen}
        activeTaskID={tasks.activeTaskID}
        status={appStatus.status}
        isLoading={chat.isLoading}
        onStop={chat.stopMessage}
        messages={chat.messages}
        input={chat.input}
        setInput={chat.setInput}
        onSubmit={chat.handleSubmit}
        menuState={slash.menuState}
        handleInputChange={slash.handleInputChange}
        handleKeyDown={slash.handleKeyDown}
        selectCommand={slash.selectCommand}
        closeMenu={slash.closeMenu}
        mode={appStatus.mode}
        onToggleMode={appStatus.toggleMode}
        chatInputRef={chatInputRef}
        onSelectProjectDir={handleSelectProjectDir}
        canChat={canChat}
        showSelectDir={(appStatus.status.cwd || '') === ''}
      />
      {settings.showSettings && (
        <SettingsPanel
          config={settings.configData}
          onClose={() => { settings.setShowSettings(false); settings.setSaveMessage(''); }}
          onSave={settings.saveSettings}
          saveMessage={settings.saveMessage}
          rules={settings.rules}
          rulesMessage={settings.rulesMessage}
          loadingRules={settings.loadingRules}
          onLoadRules={settings.loadRules}
          onReloadRules={settings.reloadRules}
          formatFileSize={settings.formatFileSize}
          formatModTime={settings.formatModTime}
        />
      )}
      {chat.followup && (
        <FollowupModal
          question={chat.followup.question}
          options={chat.followup.options}
          onAnswer={chat.handleFollowupAnswer}
        />
      )}
    </div>
  );
}

export default App

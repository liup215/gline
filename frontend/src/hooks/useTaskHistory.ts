import { useState, useCallback } from 'react';
import { ChatService } from '../../bindings/github.com/liup215/gline/internal/gui';
import { MessageRecord } from '../../bindings/github.com/liup215/gline/internal/storage/models';
import { Message } from '../types';

export function useTaskHistory() {
  const [history, setHistory] = useState<any[]>([]);
  const [activeTaskID, setActiveTaskID] = useState<string | null>(null);

  const loadHistory = useCallback(async () => {
    try {
      const tasks = await ChatService.ListTasks(50, 0);
      setHistory(tasks || []);
    } catch (err) {
      console.error('Failed to load history:', err);
    }
  }, []);

  const handleSelectTask = useCallback(async (taskID: string) => {
    try {
      const [task, msgs] = await ChatService.GetTaskSummary(taskID);
      if (!task) return;
      await ChatService.LoadTask(taskID);
      setActiveTaskID(taskID);

      // Step 1: collect tool metadata from assistant messages
      const toolInfoMap: Record<string, { name: string; input: string }> = {};
      (msgs || []).forEach((m: MessageRecord) => {
        if (m.Role === 'assistant' && m.ToolCalls && m.ToolCalls.trim() !== '') {
          try {
            const tcs = JSON.parse(m.ToolCalls);
            tcs.forEach((tc: any) => {
              const id = tc.ID || tc.id;
              if (!id) return;
              const name = tc.Name || tc.name || '';
              const inputRaw = tc.Input || tc.input || (tc.arguments ? JSON.stringify(tc.arguments) : '');
              toolInfoMap[id] = {
                name,
                input: typeof inputRaw === 'string' ? inputRaw : JSON.stringify(inputRaw),
              };
            });
          } catch (e) { /* ignore */ }
        }
      });

      // Step 2: map messages, enriching tool results with name/input
      const displayMessages: Message[] = (msgs || [])
        .filter((m: MessageRecord) => {
          // Skip assistant messages with empty content that are pure tool-call prompts
          if (m.Role === 'assistant' && m.Content === '' && m.ToolCalls && m.ToolCalls.trim() !== '') {
            return false;
          }
          return true;
        })
        .map((m: MessageRecord) => {
          const base: Message = { role: m.Role as any, content: m.Content };
          if (m.Role === 'tool' && m.ToolCallID && toolInfoMap[m.ToolCallID]) {
            const info = toolInfoMap[m.ToolCallID];
            base.id = m.ToolCallID;
            base.toolName = info.name;
            base.toolInput = info.input;
            base.toolResult = m.Content;
          }
          return base;
        });

      return { messages: displayMessages, workingDir: (task as any).WorkingDir || '' };
    } catch (err) {
      console.error('Failed to load task:', err);
      return null;
    }
  }, []);

  const handleDeleteTask = useCallback(async (e: React.MouseEvent, taskID: string) => {
    e.stopPropagation();
    if (!confirm('Delete this conversation?')) return;
    try {
      await ChatService.DeleteTask(taskID);
      if (activeTaskID === taskID) {
        setActiveTaskID(null);
      }
      loadHistory();
    } catch (err) {
      console.error('Failed to delete task:', err);
    }
  }, [activeTaskID, loadHistory]);

  const handleNewConversation = useCallback((onReset: () => void) => {
    ChatService.StartNewConversation();
    setActiveTaskID(null);
    onReset();
  }, []);

  return {
    history,
    setHistory,
    activeTaskID,
    setActiveTaskID,
    loadHistory,
    handleSelectTask,
    handleDeleteTask,
    handleNewConversation,
  };
}

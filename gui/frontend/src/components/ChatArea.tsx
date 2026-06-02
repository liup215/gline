import { Message } from '../types';
import { AppStatus } from '../types';
import { Header } from './Header';
import { MessageList } from './MessageList';
import { InputArea } from './InputArea';
import { SlashMenuState } from '../slash';

interface ChatAreaProps {
  sidebarOpen: boolean;
  setSidebarOpen: (v: boolean) => void;
  activeTaskID: string | null;
  status: AppStatus;
  isLoading: boolean;
  onStop: () => void;
  messages: Message[];
  input: string;
  setInput: (v: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  menuState: SlashMenuState;
  handleInputChange: (text: string, cursorPos: number, setInputValue: (v: string) => void, inputEl: HTMLInputElement | null) => void;
  handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>, setInputValue: (v: string) => void) => { handled: boolean };
  selectCommand: (cmd: any, setInputValue: (v: string) => void, inputEl: HTMLInputElement | null) => void;
  closeMenu: () => void;
  mode: 'plan' | 'act';
  onToggleMode: () => void;
  chatInputRef: React.MutableRefObject<HTMLInputElement | null>;
  onSelectProjectDir?: () => void;
  canChat?: boolean;
  showSelectDir?: boolean;
}

export function ChatArea(props: ChatAreaProps) {
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
      <Header
        sidebarOpen={props.sidebarOpen}
        setSidebarOpen={props.setSidebarOpen}
        activeTaskID={props.activeTaskID}
        status={props.status}
        isLoading={props.isLoading}
        onStop={props.onStop}
      />
      <MessageList messages={props.messages} onSelectProjectDir={props.onSelectProjectDir} showSelectDir={props.showSelectDir} />
      <InputArea
        input={props.input}
        setInput={props.setInput}
        isLoading={props.isLoading}
        onSubmit={props.onSubmit}
        menuState={props.menuState}
        handleInputChange={props.handleInputChange}
        handleKeyDown={props.handleKeyDown}
        selectCommand={props.selectCommand}
        closeMenu={props.closeMenu}
        status={props.status}
        mode={props.mode}
        onToggleMode={props.onToggleMode}
        chatInputRef={props.chatInputRef}
        canChat={props.canChat}
      />
    </div>
  );
}

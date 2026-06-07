export interface Message {
  role: 'user' | 'assistant' | 'tool' | 'system';
  content: string;
  id?: string;
  toolName?: string;
  toolInput?: string;
  toolResult?: string;
  streaming?: boolean;
}

export interface FileRef {
  path: string;
  name: string;
  isDir: boolean;
}

export interface AppStatus {
  provider: string;
  model: string;
  cwd: string;
  currentTokens: string;
  maxTokens: string;
}

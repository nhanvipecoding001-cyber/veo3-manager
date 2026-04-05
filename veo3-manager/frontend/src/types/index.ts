// Plain interfaces matching Wails-generated types (without class methods)
export interface Task {
  id: string;
  prompt: string;
  status: string;
  aspectRatio: string;
  model: string;
  outputCount: number;
  mediaIds: string[];
  videoPaths: string[];
  errorMessage: string;
  seed: string;
  createdAt: any;
  updatedAt: any;
  completedAt: any;
}

export interface TaskStats {
  total: number;
  pending: number;
  processing: number;
  completed: number;
  failed: number;
}

export interface BrowserInfo {
  status: string;
  chromePath: string;
  profilePath: string;
  debugPort: number;
  webSocketURL: string;
  version: string;
  stealth: boolean;
  stealthMods: string[];
}

export type QueueState = 'idle' | 'running' | 'paused' | 'stopping';
export type BrowserStatus = 'disconnected' | 'connecting' | 'connected' | 'error';
export type Page = 'dashboard' | 'queue' | 'history' | 'settings';

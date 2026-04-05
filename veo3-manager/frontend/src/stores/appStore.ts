import { create } from 'zustand';
import type { BrowserStatus, Page } from '../types';

interface AppState {
  currentPage: Page;
  browserStatus: BrowserStatus;
  setPage: (page: Page) => void;
  setBrowserStatus: (status: BrowserStatus) => void;
}

export const useAppStore = create<AppState>((set) => ({
  currentPage: 'queue',
  browserStatus: 'disconnected',
  setPage: (page) => set({ currentPage: page }),
  setBrowserStatus: (status) => set({ browserStatus: status }),
}));

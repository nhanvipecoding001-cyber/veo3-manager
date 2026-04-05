import { create } from 'zustand';
import type { Task, TaskStats, QueueState } from '../types';

interface QueueStoreState {
  tasks: Task[];
  queueState: QueueState;
  currentTaskId: string | null;
  stats: TaskStats | null;
  setTasks: (tasks: Task[]) => void;
  addTasks: (tasks: Task[]) => void;
  updateTask: (id: string, updates: Partial<Task>) => void;
  removeTask: (id: string) => void;
  setQueueState: (state: QueueState) => void;
  setCurrentTaskId: (id: string | null) => void;
  setStats: (stats: TaskStats) => void;
}

export const useQueueStore = create<QueueStoreState>((set) => ({
  tasks: [],
  queueState: 'idle',
  currentTaskId: null,
  stats: null,
  setTasks: (tasks) => set({ tasks }),
  addTasks: (newTasks) => set((s) => ({ tasks: [...newTasks, ...s.tasks] })),
  updateTask: (id, updates) =>
    set((s) => ({
      tasks: s.tasks.map((t) => (t.id === id ? { ...t, ...updates } : t)),
    })),
  removeTask: (id) =>
    set((s) => ({ tasks: s.tasks.filter((t) => t.id !== id) })),
  setQueueState: (queueState) => set({ queueState }),
  setCurrentTaskId: (currentTaskId) => set({ currentTaskId }),
  setStats: (stats) => set({ stats }),
}));

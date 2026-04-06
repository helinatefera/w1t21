import { create } from 'zustand';
import type { User } from '../types';
import { me } from '../api/auth';

interface AuthState {
  user: User | null;
  roles: string[];
  isAuthenticated: boolean;
  isBootstrapping: boolean;
  setAuth: (user: User, roles: string[]) => void;
  clearAuth: () => void;
  hasRole: (role: string) => boolean;
  bootstrap: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  roles: [],
  isAuthenticated: false,
  isBootstrapping: true,
  setAuth: (user, roles) => set({ user, roles, isAuthenticated: true, isBootstrapping: false }),
  clearAuth: () => set({ user: null, roles: [], isAuthenticated: false, isBootstrapping: false }),
  hasRole: (role) => get().roles.includes(role),
  bootstrap: async () => {
    try {
      const { data } = await me();
      set({ user: data.user, roles: data.roles, isAuthenticated: true, isBootstrapping: false });
    } catch {
      set({ user: null, roles: [], isAuthenticated: false, isBootstrapping: false });
    }
  },
}));

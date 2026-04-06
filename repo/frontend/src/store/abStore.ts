import { create } from 'zustand';
import type { ABTestAssignment } from '../types';

interface ABState {
  assignments: ABTestAssignment[];
  setAssignments: (a: ABTestAssignment[]) => void;
  getVariant: (testName: string) => string | null;
}

export const useABStore = create<ABState>((set, get) => ({
  assignments: [],
  setAssignments: (assignments) => set({ assignments }),
  getVariant: (testName) => {
    const a = get().assignments.find((x) => x.test_name === testName);
    return a?.variant ?? null;
  },
}));

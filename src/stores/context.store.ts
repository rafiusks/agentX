import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { DEFAULT_CONTEXT_CONFIG, type ContextConfig } from '../config/context';

interface ContextState {
  config: ContextConfig;
  setStrategy: (strategy: ContextConfig['strategy']) => void;
  setUseImportanceScoring: (use: boolean) => void;
  updateConfig: (updates: Partial<ContextConfig>) => void;
  resetToDefaults: () => void;
}

export const useContextStore = create<ContextState>()(
  persist(
    (set) => ({
      config: DEFAULT_CONTEXT_CONFIG,
      
      setStrategy: (strategy) => set((state) => ({
        config: {
          ...state.config,
          strategy,
          // Enable importance scoring for smart mode
          useImportanceScoring: strategy === 'smart',
        }
      })),
      
      setUseImportanceScoring: (use) => set((state) => ({
        config: {
          ...state.config,
          useImportanceScoring: use,
        }
      })),
      
      updateConfig: (updates) => set((state) => ({
        config: {
          ...state.config,
          ...updates,
        }
      })),
      
      resetToDefaults: () => set({
        config: DEFAULT_CONTEXT_CONFIG,
      }),
    }),
    {
      name: 'context-settings',
    }
  )
);
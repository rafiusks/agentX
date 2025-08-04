import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type UIMode = 'simple' | 'mission-control' | 'pro'

interface UIState {
  mode: UIMode
  showAdvancedOptions: boolean
  showDebugInfo: boolean
  
  // Actions
  setMode: (mode: UIMode) => void
  toggleAdvancedOptions: () => void
  toggleDebugInfo: () => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      mode: 'simple',
      showAdvancedOptions: false,
      showDebugInfo: false,
      
      setMode: (mode) => set({ 
        mode,
        showAdvancedOptions: mode !== 'simple',
        showDebugInfo: mode === 'pro'
      }),
      
      toggleAdvancedOptions: () => set((state) => ({ 
        showAdvancedOptions: !state.showAdvancedOptions 
      })),
      
      toggleDebugInfo: () => set((state) => ({ 
        showDebugInfo: !state.showDebugInfo 
      })),
    }),
    {
      name: 'agentx-ui-settings'
    }
  )
)
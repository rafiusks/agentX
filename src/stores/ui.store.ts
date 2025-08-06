import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * UI Store - Only handles client-side UI state
 * Server state is managed by TanStack Query
 */

export type UIMode = 'simple' | 'mission-control' | 'pro'
export type Theme = 'light' | 'dark' | 'system'

interface UIState {
  // Mode & Display
  mode: UIMode
  showAdvancedOptions: boolean
  showDebugInfo: boolean
  theme: Theme
  
  // Sidebar & Navigation
  sidebarOpen: boolean
  activeTab: 'chat' | 'agents' | 'mcp' | 'settings'
  
  // Command Palette
  commandPaletteOpen: boolean
  
  // Modals
  activeModal: string | null
  
  // Actions - Mode
  setMode: (mode: UIMode) => void
  toggleAdvancedOptions: () => void
  toggleDebugInfo: () => void
  
  // Actions - Theme
  setTheme: (theme: Theme) => void
  
  // Actions - Sidebar
  toggleSidebar: () => void
  setSidebarOpen: (open: boolean) => void
  setActiveTab: (tab: 'chat' | 'agents' | 'mcp' | 'settings') => void
  
  // Actions - Command Palette
  setCommandPaletteOpen: (open: boolean) => void
  
  // Actions - Modals
  openModal: (modal: string) => void
  closeModal: () => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      // State
      mode: 'simple',
      showAdvancedOptions: false,
      showDebugInfo: false,
      theme: 'system',
      sidebarOpen: true,
      activeTab: 'chat',
      commandPaletteOpen: false,
      activeModal: null,
      
      // Actions - Mode
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
      
      // Actions - Theme
      setTheme: (theme) => set({ theme }),
      
      // Actions - Sidebar
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setActiveTab: (tab) => set({ activeTab: tab }),
      
      // Actions - Command Palette
      setCommandPaletteOpen: (open) => set({ commandPaletteOpen: open }),
      
      // Actions - Modals
      openModal: (modal) => set({ activeModal: modal }),
      closeModal: () => set({ activeModal: null }),
    }),
    {
      name: 'agentx-ui-settings',
      partialize: (state) => ({
        // Only persist user preferences
        mode: state.mode,
        theme: state.theme,
        sidebarOpen: state.sidebarOpen,
        activeTab: state.activeTab,
      }),
    }
  )
)
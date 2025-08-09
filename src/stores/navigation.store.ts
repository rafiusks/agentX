import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface NavigationStore {
  isSidebarCollapsed: boolean;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
}

export const useNavigationStore = create<NavigationStore>()(
  persist(
    (set) => ({
      isSidebarCollapsed: false,
      
      toggleSidebar: () => set((state) => ({
        isSidebarCollapsed: !state.isSidebarCollapsed
      })),
      
      setSidebarCollapsed: (collapsed: boolean) => set({
        isSidebarCollapsed: collapsed
      }),
    }),
    {
      name: 'agentx-navigation',
      partialize: (state) => ({
        isSidebarCollapsed: state.isSidebarCollapsed
      })
    }
  )
);
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type CanvasType = 'code' | 'document' | 'diagram' | 'data';
export type CanvasMode = 'edit' | 'view' | 'diff';

export interface CanvasArtifact {
  id: string;
  sessionId?: string;
  type: CanvasType;
  title: string;
  content: string;
  language?: string; // For code artifacts
  version: number;
  parentVersion?: string;
  isDirty: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CanvasHistory {
  artifactId: string;
  versions: CanvasArtifact[];
  currentVersion: number;
}

interface CanvasState {
  // Canvas visibility
  isCanvasOpen: boolean;
  canvasWidth: number; // Percentage of screen width
  
  // Current artifact
  currentArtifact: CanvasArtifact | null;
  artifactHistory: Map<string, CanvasHistory>;
  
  // Canvas settings
  canvasType: CanvasType;
  canvasMode: CanvasMode;
  showLineNumbers: boolean;
  wordWrap: boolean;
  fontSize: number;
  theme: 'light' | 'dark' | 'auto';
  
  // Actions
  openCanvas: (type?: CanvasType) => void;
  closeCanvas: () => void;
  toggleCanvas: () => void;
  setCanvasWidth: (width: number) => void;
  
  // Artifact management
  createArtifact: (type: CanvasType, title: string, content: string, language?: string) => void;
  updateArtifact: (content: string) => void;
  saveArtifact: () => void;
  loadArtifact: (artifactId: string) => void;
  clearArtifact: () => void;
  
  // Version control
  createVersion: () => void;
  loadVersion: (version: number) => void;
  compareVersions: (v1: number, v2: number) => void;
  
  // Settings
  setCanvasMode: (mode: CanvasMode) => void;
  setCanvasType: (type: CanvasType) => void;
  toggleLineNumbers: () => void;
  toggleWordWrap: () => void;
  setFontSize: (size: number) => void;
  setTheme: (theme: 'light' | 'dark' | 'auto') => void;
}

export const useCanvasStore = create<CanvasState>()(
  persist(
    (set, get) => ({
      // Initial state
      isCanvasOpen: false,
      canvasWidth: 50,
      currentArtifact: null,
      artifactHistory: new Map(),
      canvasType: 'code',
      canvasMode: 'edit',
      showLineNumbers: true,
      wordWrap: false,
      fontSize: 14,
      theme: 'auto',

      // Canvas control
      openCanvas: (type = 'code') => {
        set({ isCanvasOpen: true, canvasType: type });
      },

      closeCanvas: () => {
        const { currentArtifact } = get();
        if (currentArtifact?.isDirty) {
          // Prompt to save changes
          const shouldSave = window.confirm('You have unsaved changes. Do you want to save them?');
          if (shouldSave) {
            get().saveArtifact();
          }
        }
        set({ isCanvasOpen: false });
      },

      toggleCanvas: () => {
        const { isCanvasOpen } = get();
        if (isCanvasOpen) {
          get().closeCanvas();
        } else {
          get().openCanvas();
        }
      },

      setCanvasWidth: (width) => {
        set({ canvasWidth: Math.min(80, Math.max(30, width)) });
      },

      // Artifact management
      createArtifact: (type, title, content, language) => {
        const artifact: CanvasArtifact = {
          id: `artifact-${Date.now()}`,
          type,
          title,
          content,
          language,
          version: 1,
          isDirty: false,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };

        const history: CanvasHistory = {
          artifactId: artifact.id,
          versions: [artifact],
          currentVersion: 1,
        };

        set((state) => {
          const newHistory = new Map(state.artifactHistory);
          newHistory.set(artifact.id, history);
          return {
            currentArtifact: artifact,
            artifactHistory: newHistory,
            isCanvasOpen: true,
            canvasType: type,
          };
        });
      },

      updateArtifact: (content) => {
        set((state) => {
          if (!state.currentArtifact) return state;
          
          return {
            currentArtifact: {
              ...state.currentArtifact,
              content,
              isDirty: true,
              updatedAt: new Date().toISOString(),
            },
          };
        });
      },

      saveArtifact: () => {
        const { currentArtifact } = get();
        if (!currentArtifact || !currentArtifact.isDirty) return;

        set((state) => {
          if (!state.currentArtifact) return state;
          
          const savedArtifact = {
            ...state.currentArtifact,
            isDirty: false,
            updatedAt: new Date().toISOString(),
          };

          // Update version history
          const history = state.artifactHistory.get(savedArtifact.id);
          if (history) {
            // Check if content actually changed
            const lastVersion = history.versions[history.versions.length - 1];
            if (lastVersion.content !== savedArtifact.content) {
              // Create new version
              const newVersion = {
                ...savedArtifact,
                version: history.currentVersion + 1,
                parentVersion: lastVersion.id,
              };
              
              history.versions.push(newVersion);
              history.currentVersion = newVersion.version;
            }
          }

          return {
            currentArtifact: savedArtifact,
          };
        });
      },

      loadArtifact: (artifactId) => {
        const { artifactHistory } = get();
        const history = artifactHistory.get(artifactId);
        
        if (history) {
          const currentVersion = history.versions[history.currentVersion - 1];
          set({
            currentArtifact: currentVersion,
            isCanvasOpen: true,
            canvasType: currentVersion.type,
          });
        }
      },

      clearArtifact: () => {
        set({ currentArtifact: null });
      },

      // Version control
      createVersion: () => {
        get().saveArtifact();
      },

      loadVersion: (version) => {
        const { currentArtifact, artifactHistory } = get();
        if (!currentArtifact) return;

        const history = artifactHistory.get(currentArtifact.id);
        if (history && version > 0 && version <= history.versions.length) {
          const versionArtifact = history.versions[version - 1];
          set({
            currentArtifact: versionArtifact,
          });
        }
      },

      compareVersions: (_v1, _v2) => {
        // TODO: Implement version comparison
        set({ canvasMode: 'diff' });
      },

      // Settings
      setCanvasMode: (mode) => set({ canvasMode: mode }),
      setCanvasType: (type) => set({ canvasType: type }),
      toggleLineNumbers: () => set((state) => ({ showLineNumbers: !state.showLineNumbers })),
      toggleWordWrap: () => set((state) => ({ wordWrap: !state.wordWrap })),
      setFontSize: (size) => set({ fontSize: Math.min(24, Math.max(10, size)) }),
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: 'canvas-storage',
      partialize: (state) => ({
        canvasWidth: state.canvasWidth,
        showLineNumbers: state.showLineNumbers,
        wordWrap: state.wordWrap,
        fontSize: state.fontSize,
        theme: state.theme,
      }),
    }
  )
);
import { useEffect } from 'react';
import { usePreferencesStore, type ResponseStyle } from '../stores/preferences.store';

export function useKeyboardShortcuts() {
  const { responseStyle, setResponseStyle } = usePreferencesStore();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + Shift + S to cycle through response styles
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 's') {
        e.preventDefault();
        
        const styles: ResponseStyle[] = ['ultra-concise', 'concise', 'balanced', 'detailed'];
        const currentIndex = styles.indexOf(responseStyle);
        const nextIndex = (currentIndex + 1) % styles.length;
        setResponseStyle(styles[nextIndex]);
        
        // Show a toast or some visual feedback
        console.log(`Response style changed to: ${styles[nextIndex]}`);
      }
      
      // Cmd/Ctrl + Shift + 1-4 for direct style selection
      if ((e.metaKey || e.ctrlKey) && e.shiftKey) {
        const styleMap: Record<string, ResponseStyle> = {
          '1': 'ultra-concise',
          '2': 'concise',
          '3': 'balanced',
          '4': 'detailed',
        };
        
        if (styleMap[e.key]) {
          e.preventDefault();
          setResponseStyle(styleMap[e.key]);
          console.log(`Response style changed to: ${styleMap[e.key]}`);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [responseStyle, setResponseStyle]);
}
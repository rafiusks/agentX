import { useEffect, useCallback, useRef } from 'react';
import { useChatStore } from '../stores/chat.store';
import { useStreamingStore } from '../stores/streaming.store';
import { useUIStore } from '../stores/ui.store';

interface ShortcutConfig {
  key: string;
  ctrl?: boolean;
  cmd?: boolean;
  shift?: boolean;
  alt?: boolean;
  action: () => void;
  description: string;
}

export function useEnhancedKeyboardShortcuts() {
  const { currentChatId } = useChatStore();
  const { isStreaming, abortStream } = useStreamingStore();
  const { showCommandPalette, setShowCommandPalette } = useUIStore();
  
  // Track focused message index for navigation
  const focusedMessageIndex = useRef<number>(-1);
  
  // Navigate between messages
  const navigateMessages = useCallback((direction: 'up' | 'down') => {
    const messages = document.querySelectorAll('[data-message-id]');
    if (messages.length === 0) return;
    
    if (direction === 'up') {
      focusedMessageIndex.current = Math.max(0, focusedMessageIndex.current - 1);
    } else {
      focusedMessageIndex.current = Math.min(messages.length - 1, focusedMessageIndex.current + 1);
    }
    
    const targetMessage = messages[focusedMessageIndex.current] as HTMLElement;
    targetMessage?.focus();
    targetMessage?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }, []);
  
  // Copy focused message
  const copyFocusedMessage = useCallback(() => {
    const focusedElement = document.activeElement;
    const messageContent = focusedElement?.querySelector('[data-message-content]')?.textContent;
    
    if (messageContent) {
      navigator.clipboard.writeText(messageContent);
      
      // Show visual feedback
      const copyButton = focusedElement?.querySelector('[data-copy-button]') as HTMLElement;
      if (copyButton) {
        copyButton.click();
      }
    }
  }, []);
  
  // Regenerate last assistant message
  const regenerateLastMessage = useCallback(() => {
    const regenerateButtons = document.querySelectorAll('[data-regenerate-button]');
    const lastButton = regenerateButtons[regenerateButtons.length - 1] as HTMLElement;
    lastButton?.click();
  }, []);
  
  // Focus message input
  const focusMessageInput = useCallback(() => {
    const input = document.querySelector('textarea[placeholder*="Ask"]') as HTMLTextAreaElement;
    input?.focus();
  }, []);
  
  // Expand/collapse code blocks
  const toggleCodeBlock = useCallback(() => {
    const focusedElement = document.activeElement;
    const codeBlock = focusedElement?.querySelector('[data-code-block]') as HTMLElement;
    const expandButton = codeBlock?.querySelector('[data-expand-button]') as HTMLElement;
    expandButton?.click();
  }, []);
  
  // Define all shortcuts
  const shortcuts: ShortcutConfig[] = [
    // Navigation
    {
      key: 'ArrowUp',
      alt: true,
      action: () => navigateMessages('up'),
      description: 'Navigate to previous message'
    },
    {
      key: 'ArrowDown',
      alt: true,
      action: () => navigateMessages('down'),
      description: 'Navigate to next message'
    },
    {
      key: 'Tab',
      action: () => {
        // Default tab behavior for accessibility
        return;
      },
      description: 'Navigate between interactive elements'
    },
    
    // Actions
    {
      key: 'c',
      cmd: true,
      action: copyFocusedMessage,
      description: 'Copy focused message'
    },
    // Removed cmd+r shortcut as it hijacks browser refresh
    {
      key: '/',
      cmd: true,
      action: focusMessageInput,
      description: 'Focus message input'
    },
    {
      key: 'k',
      cmd: true,
      action: () => setShowCommandPalette(true),
      description: 'Open command palette'
    },
    {
      key: 'Escape',
      action: () => {
        if (isStreaming) {
          abortStream();
        } else if (showCommandPalette) {
          setShowCommandPalette(false);
        }
      },
      description: 'Stop streaming / Close dialogs'
    },
    {
      key: 'Enter',
      action: toggleCodeBlock,
      description: 'Expand/collapse code block (when focused)'
    },
    
    // Quick actions
    {
      key: '1',
      cmd: true,
      action: () => {
        // Switch to ultra-concise mode
        const styleSelect = document.querySelector('select[title="Response style"]') as HTMLSelectElement;
        if (styleSelect) {
          styleSelect.value = 'ultra-concise';
          styleSelect.dispatchEvent(new Event('change', { bubbles: true }));
        }
      },
      description: 'Switch to ultra-concise responses'
    },
    {
      key: '2',
      cmd: true,
      action: () => {
        // Switch to concise mode
        const styleSelect = document.querySelector('select[title="Response style"]') as HTMLSelectElement;
        if (styleSelect) {
          styleSelect.value = 'concise';
          styleSelect.dispatchEvent(new Event('change', { bubbles: true }));
        }
      },
      description: 'Switch to concise responses'
    },
    {
      key: '3',
      cmd: true,
      action: () => {
        // Switch to balanced mode
        const styleSelect = document.querySelector('select[title="Response style"]') as HTMLSelectElement;
        if (styleSelect) {
          styleSelect.value = 'balanced';
          styleSelect.dispatchEvent(new Event('change', { bubbles: true }));
        }
      },
      description: 'Switch to balanced responses'
    },
    {
      key: '4',
      cmd: true,
      action: () => {
        // Switch to detailed mode
        const styleSelect = document.querySelector('select[title="Response style"]') as HTMLSelectElement;
        if (styleSelect) {
          styleSelect.value = 'detailed';
          styleSelect.dispatchEvent(new Event('change', { bubbles: true }));
        }
      },
      description: 'Switch to detailed responses'
    },
  ];
  
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't handle shortcuts when typing in inputs (except for Escape)
      const isTyping = ['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement).tagName);
      if (isTyping && e.key !== 'Escape') return;
      
      // Check for matching shortcut
      const shortcut = shortcuts.find(s => {
        const keyMatch = s.key === e.key;
        const ctrlMatch = s.ctrl ? (e.ctrlKey || e.metaKey) : !s.cmd || true;
        const cmdMatch = s.cmd ? (e.metaKey || e.ctrlKey) : true;
        const shiftMatch = s.shift ? e.shiftKey : !e.shiftKey;
        const altMatch = s.alt ? e.altKey : !e.altKey;
        
        return keyMatch && ctrlMatch && cmdMatch && shiftMatch && altMatch;
      });
      
      if (shortcut) {
        e.preventDefault();
        shortcut.action();
      }
    };
    
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [shortcuts, isStreaming, showCommandPalette]);
  
  return { shortcuts };
}
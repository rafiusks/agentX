import React, { useEffect, useState } from 'react';
import { Sparkles, X } from 'lucide-react';
import { conversationIntelligence } from '@/services/conversationIntelligence';
import { useChatStore } from '@/stores/chat.store';

interface SmartSuggestionsProps {
  onSuggestionClick?: (suggestion: string) => void;
}

export const SmartSuggestions: React.FC<SmartSuggestionsProps> = ({ onSuggestionClick }) => {
  const { currentChatId } = useChatStore();
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());
  const [showSuggestions, setShowSuggestions] = useState(false);
  
  useEffect(() => {
    if (!currentChatId) {
      setSuggestions([]);
      return;
    }
    
    // Get suggestions based on conversation context
    const updateSuggestions = () => {
      const newSuggestions = conversationIntelligence.getSuggestions(currentChatId);
      setSuggestions(newSuggestions.filter(s => !dismissed.has(s)));
    };
    
    // Update suggestions less frequently
    updateSuggestions();
    const interval = setInterval(updateSuggestions, 60000); // Update every 60 seconds
    
    return () => clearInterval(interval);
  }, [currentChatId, dismissed]);
  
  useEffect(() => {
    // Show suggestions only after a long delay and only if really useful
    if (suggestions.length > 0) {
      // Only show if we have high-confidence suggestions
      const timer = setTimeout(() => setShowSuggestions(true), 10000); // 10 seconds delay
      return () => clearTimeout(timer);
    } else {
      setShowSuggestions(false);
    }
  }, [suggestions]);
  
  const handleDismiss = (suggestion: string) => {
    setDismissed(prev => new Set(prev).add(suggestion));
    setSuggestions(prev => prev.filter(s => s !== suggestion));
  };
  
  const handleSuggestionClick = (suggestion: string) => {
    onSuggestionClick?.(suggestion);
    handleDismiss(suggestion);
  };
  
  if (!showSuggestions || suggestions.length === 0) {
    return null;
  }
  
  return (
    <div className="absolute bottom-20 right-4 max-w-sm z-40">
      <div className="bg-background-secondary/90 border border-border-subtle/50 rounded-lg shadow-sm p-2 backdrop-blur-sm">
        <div className="flex items-center gap-2 mb-1">
          <Sparkles size={12} className="text-accent-blue/60" />
          <span className="text-xs text-foreground-tertiary">
            Suggestion
          </span>
        </div>
        
        {suggestions.map((suggestion, index) => (
          <div 
            key={index}
            className="flex items-start gap-2 group"
          >
            <button
              onClick={() => handleSuggestionClick(suggestion)}
              className="flex-1 text-left text-xs text-foreground-secondary hover:text-foreground-primary 
                         hover:bg-background-tertiary rounded px-2 py-1 transition-colors"
            >
              {suggestion}
            </button>
            <button
              onClick={() => handleDismiss(suggestion)}
              className="opacity-0 group-hover:opacity-100 transition-opacity p-1"
              title="Dismiss"
            >
              <X size={12} className="text-foreground-tertiary hover:text-foreground-secondary" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
import { Bot } from 'lucide-react';

interface TypingIndicatorProps {
  modelName?: string;
}

export const TypingIndicator = ({ modelName = 'AI' }: TypingIndicatorProps) => {
  return (
    <div className="flex gap-3 message-animate-in">
      {/* Avatar */}
      <div className="message-avatar bg-gradient-to-br from-accent-blue to-accent-green">
        <Bot size={16} className="text-white" />
      </div>
      
      {/* Typing Bubble */}
      <div className="message-assistant max-w-[100px]">
        <div className="flex items-center gap-2">
          <span className="text-xs text-foreground-muted">{modelName}</span>
          <div className="flex items-center gap-1">
            <span className="typing-dot"></span>
            <span className="typing-dot" style={{ animationDelay: '0.2s' }}></span>
            <span className="typing-dot" style={{ animationDelay: '0.4s' }}></span>
          </div>
        </div>
      </div>
    </div>
  );
};
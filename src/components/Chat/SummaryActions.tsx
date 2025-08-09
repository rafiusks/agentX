import { useState } from 'react';
import { Sparkles, Loader2 } from 'lucide-react';
import { Button } from '../ui/button';
import { useGenerateSummary } from '../../hooks/queries/useSummaries';
import { toast } from 'sonner';

interface SummaryActionsProps {
  sessionId?: string;
  messageCount: number;
}

export function SummaryActions({ sessionId, messageCount }: SummaryActionsProps) {
  const generateSummaryMutation = useGenerateSummary();
  const [isGenerating, setIsGenerating] = useState(false);

  const canGenerateSummary = sessionId && messageCount >= 10;

  const handleGenerateSummary = async () => {
    if (!sessionId || isGenerating) return;

    setIsGenerating(true);
    try {
      await generateSummaryMutation.mutateAsync({
        sessionId,
        messageCount: Math.min(messageCount, 30), // Summarize up to 30 messages
      });
      toast.success('Summary generated successfully');
    } catch (error) {
      console.error('Failed to generate summary:', error);
      toast.error('Failed to generate summary');
    } finally {
      setIsGenerating(false);
    }
  };

  if (!canGenerateSummary) return null;

  return (
    <div className="flex items-center gap-2">
      <Button
        variant="ghost"
        size="sm"
        onClick={handleGenerateSummary}
        disabled={isGenerating}
        className="text-xs"
        title="Generate a summary of the conversation to save context space"
      >
        {isGenerating ? (
          <Loader2 className="h-3 w-3 mr-1 animate-spin" />
        ) : (
          <Sparkles className="h-3 w-3 mr-1" />
        )}
        Generate Summary
      </Button>
    </div>
  );
}
import type { Message } from '../hooks/queries/useChats';

/**
 * Calculate importance score for a message (0-1)
 * Higher scores mean the message is more important to keep in context
 */
export function calculateMessageImportance(
  message: Message,
  previousMessage?: Message
): { score: number; flags: Message['importanceFlags'] } {
  let score = 0.3; // Base score
  const flags: Message['importanceFlags'] = {};

  const content = message.content.toLowerCase();

  // Check for code blocks (very important)
  if (content.includes('```') || content.includes('    ')) {
    score += 0.3;
    flags.hasCode = true;
  }

  // Check for error indicators
  if (
    content.includes('error') ||
    content.includes('exception') ||
    content.includes('failed') ||
    content.includes('❌') ||
    message.content.startsWith('Error:')
  ) {
    score += 0.25;
    flags.hasError = true;
  }

  // Check for decision-making keywords
  const decisionKeywords = [
    'should we',
    'let\'s',
    'we need to',
    'i think',
    'instead',
    'better to',
    'recommend',
    'suggest',
    'important',
    'critical',
    'must',
    'please',
    'make sure',
  ];
  
  if (decisionKeywords.some(keyword => content.includes(keyword))) {
    score += 0.2;
    flags.hasDecision = true;
  }

  // Check if this is a user correction (follows an assistant message)
  if (
    message.role === 'user' &&
    previousMessage?.role === 'assistant' &&
    (content.includes('no') ||
     content.includes('wrong') ||
     content.includes('actually') ||
     content.includes('instead') ||
     content.includes('fix') ||
     content.includes('change'))
  ) {
    score += 0.25;
    flags.isUserCorrection = true;
  }

  // Recent messages are more important (recency bias)
  // This will be handled by the context selection logic

  // Long messages might contain more context
  if (message.content.length > 500) {
    score += 0.1;
  }

  // Cap the score at 1.0
  score = Math.min(score, 1.0);

  // Estimate tokens (rough approximation)
  flags.tokens = Math.ceil(message.content.length / 4);

  return { score, flags };
}

/**
 * Select messages for context based on importance and constraints
 */
export function selectImportantMessages(
  messages: Message[],
  maxMessages: number,
  maxTokens: number
): Message[] {
  // Calculate importance for all messages
  const scoredMessages = messages.map((msg, index) => ({
    message: msg,
    ...calculateMessageImportance(msg, messages[index - 1])
  }));

  // Sort by importance score (keeping recent messages at the end)
  const recentCount = Math.min(10, messages.length); // Always keep last 10
  const recentMessages = scoredMessages.slice(-recentCount);
  const olderMessages = scoredMessages.slice(0, -recentCount);

  // Sort older messages by importance
  olderMessages.sort((a, b) => b.score - a.score);

  // Select messages within constraints
  const selected: typeof scoredMessages = [];
  let tokenCount = 0;

  // First, add recent messages
  for (const msg of recentMessages) {
    if (selected.length < maxMessages && tokenCount + (msg.flags.tokens || 0) < maxTokens) {
      selected.push(msg);
      tokenCount += msg.flags.tokens || 0;
    }
  }

  // Then add important older messages
  for (const msg of olderMessages) {
    if (
      msg.score > 0.6 && // Only high-importance messages
      selected.length < maxMessages &&
      tokenCount + (msg.flags.tokens || 0) < maxTokens
    ) {
      selected.push(msg);
      tokenCount += msg.flags.tokens || 0;
    }
  }

  // Sort selected messages back to chronological order
  selected.sort((a, b) => {
    const timeA = new Date(a.message.created_at).getTime();
    const timeB = new Date(b.message.created_at).getTime();
    return timeA - timeB;
  });

  return selected.map(s => s.message);
}
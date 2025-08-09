export function formatLLMError(error: any): string {
  // Handle different error formats from various LLM providers
  
  // OpenAI/OpenAI-compatible errors
  if (error?.error?.message) {
    return error.error.message;
  }
  
  // Anthropic errors
  if (error?.error?.type === 'invalid_request_error') {
    return error.error.message || 'Invalid request to the AI model';
  }
  
  // API errors
  if (error?.message) {
    // Clean up common error messages
    if (error.message.includes('429')) {
      return 'Rate limit exceeded. Please wait a moment and try again.';
    }
    if (error.message.includes('401') || error.message.includes('403')) {
      return 'Authentication failed. Please check your API key.';
    }
    if (error.message.includes('404')) {
      return 'Model not found. Please check if the selected model is available.';
    }
    if (error.message.includes('500') || error.message.includes('502') || error.message.includes('503')) {
      return 'The AI service is temporarily unavailable. Please try again later.';
    }
    if (error.message.includes('timeout')) {
      return 'Request timed out. The AI service took too long to respond.';
    }
    if (error.message.includes('network') || error.message.includes('fetch')) {
      return 'Network error. Please check your internet connection.';
    }
    
    return error.message;
  }
  
  // String errors
  if (typeof error === 'string') {
    return error;
  }
  
  // Default fallback
  return 'An unexpected error occurred. Please try again.';
}
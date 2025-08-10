import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type ResponseStyle = 'ultra-concise' | 'concise' | 'balanced' | 'detailed';
export type SearchMode = 'conservative' | 'balanced' | 'aggressive';

interface PreferencesState {
  responseStyle: ResponseStyle;
  maxResponseTokens: number | null;
  includeCodeComments: boolean;
  preferBulletPoints: boolean;
  searchMode: SearchMode;
  forceWebSearch: boolean;
  
  setResponseStyle: (style: ResponseStyle) => void;
  setMaxResponseTokens: (tokens: number | null) => void;
  setIncludeCodeComments: (include: boolean) => void;
  setPreferBulletPoints: (prefer: boolean) => void;
  setSearchMode: (mode: SearchMode) => void;
  setForceWebSearch: (force: boolean) => void;
  
  getSystemPrompt: () => string;
}

const stylePrompts: Record<ResponseStyle, string> = {
  'ultra-concise': `Be extremely brief. Answer in 1-3 sentences maximum. No explanations unless asked. For code, use \`\`\`language blocks.`,
  
  concise: `Be concise and direct. Give me the key information without fluff. A paragraph or two is fine, but get straight to the point. For longer topics, bullet points work well. Always use \`\`\`language for code blocks.`,
  
  balanced: `Be conversational yet informative. Write naturally - mix paragraphs with occasional lists or bullets when they help clarity. Think of this as explaining to a colleague: friendly but focused. Don't over-structure everything into bullet points unless it really helps. For code, use \`\`\`language blocks.`,
  
  detailed: `Take your time to explain things thoroughly. I want comprehensive answers with context, examples, and nuance. Walk me through your thinking. Cover different angles and edge cases. Feel free to be expansive - multiple paragraphs are welcome. For code, include detailed examples with \`\`\`language blocks and explain the important parts.`
};

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set, get) => ({
      responseStyle: 'concise',
      maxResponseTokens: null,
      includeCodeComments: false,
      preferBulletPoints: true,
      searchMode: 'balanced',
      forceWebSearch: false,
      
      setResponseStyle: (style) => set({ responseStyle: style }),
      setMaxResponseTokens: (tokens) => set({ maxResponseTokens: tokens }),
      setIncludeCodeComments: (include) => set({ includeCodeComments: include }),
      setPreferBulletPoints: (prefer) => set({ preferBulletPoints: prefer }),
      setSearchMode: (mode) => set({ searchMode: mode }),
      setForceWebSearch: (force) => set({ forceWebSearch: force }),
      
      getSystemPrompt: () => {
        const state = get();
        console.log('[PreferencesStore] Getting system prompt for style:', state.responseStyle);
        let prompt = stylePrompts[state.responseStyle];
        
        // Put the response style first as highest priority
        if (state.responseStyle === 'ultra-concise') {
          prompt = `⚠️ CRITICAL: ${prompt}`;
        }
        
        // Add token limit awareness
        if (state.maxResponseTokens) {
          const wordEstimate = Math.floor(state.maxResponseTokens * 0.75);
          
          // For balanced/detailed modes, be gentler about limits
          if (state.responseStyle === 'balanced' || state.responseStyle === 'detailed') {
            prompt = `Note: Try to keep your response under ${wordEstimate} words if possible.\n\n${prompt}`;
          } else {
            prompt = `Keep your response under ${state.maxResponseTokens} tokens (roughly ${wordEstimate} words).\n\n${prompt}`;
          }
          
          // Only add restrictive guidance for ultra-concise or with very low limits
          if (state.responseStyle === 'ultra-concise' || state.maxResponseTokens <= 300) {
            prompt += '\n\nBe extremely selective about what to include.';
          }
        }
        
        // Add minimal code formatting reminder
        if (!prompt.includes('```')) {
          prompt = `${prompt}\n\nRemember to use \`\`\`language for code blocks.`;
        }
        
        if (state.preferBulletPoints) {
          // Only add bullet preference for non-balanced modes
          if (state.responseStyle !== 'balanced' && state.responseStyle !== 'detailed') {
            prompt += '\n\nWhen listing multiple items, use bullet points for clarity.';
          }
        }
        
        if (!state.includeCodeComments && state.responseStyle === 'ultra-concise') {
          prompt += ' Skip code comments.';
        }
        
        return prompt;
      }
    }),
    {
      name: 'response-preferences',
    }
  )
);
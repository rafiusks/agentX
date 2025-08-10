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
  'ultra-concise': `You are an AI assistant. Be EXTREMELY brief.
- Maximum 1-3 sentences for any response
- NO explanations whatsoever unless explicitly requested with "explain"
- NO greetings, acknowledgments, or pleasantries
- Answer with the absolute minimum necessary
- For code: ALWAYS wrap code in markdown code blocks with \`\`\`language
- If asked for a list: bullet points only, no intro/outro`,
  
  concise: `You are a helpful AI assistant. Provide concise, direct answers. 
- Get to the point immediately - no introductions
- Maximum 2-3 sentences for explanations
- Skip phrases like "I'd be happy to", "Let me", "Here's", etc.
- For code: ALWAYS use markdown code blocks with \`\`\`language syntax
- Use bullet points instead of paragraphs
- Omit obvious information`,
  
  balanced: `You are a helpful AI assistant. Please provide clear, well-structured answers.
- Balance brevity with clarity
- Include essential context and explanations
- Use formatting to improve readability
- ALWAYS wrap code in markdown code blocks with \`\`\`language syntax
- Provide examples when helpful
- Keep responses focused on the question`,
  
  detailed: `You are a helpful AI assistant. Please provide comprehensive, thorough answers.
- Include detailed explanations and context
- Cover edge cases and alternatives
- ALWAYS wrap code in markdown code blocks with \`\`\`language syntax
- Provide examples and best practices
- Explain the reasoning behind recommendations
- Be thorough but organized`
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
        let prompt = stylePrompts[state.responseStyle];
        
        // ALWAYS add code formatting rules
        prompt = `IMPORTANT CODE FORMATTING RULES:
- ALWAYS wrap code in markdown code blocks using \`\`\`language syntax
- Use proper language identifiers (python, javascript, typescript, go, etc.)
- Include proper line breaks and indentation in code
- Never send code as plain text or inline text
- For single-line code snippets, use inline code with single backticks

${prompt}`;
        
        // Add token limit awareness
        if (state.maxResponseTokens) {
          const wordEstimate = Math.floor(state.maxResponseTokens * 0.75);
          prompt = `TOKEN LIMIT: You have a ${state.maxResponseTokens} token limit (approximately ${wordEstimate} words). You MUST complete your response within this limit. Be concise and prioritize the most important information.\n\n${prompt}`;
          
          // Add specific guidance based on token limit
          if (state.maxResponseTokens <= 500) {
            prompt += '\n- Extremely brief responses only - get to the point immediately';
            prompt += '\n- Skip ALL explanations unless critical';
            prompt += '\n- For code: provide minimal working examples';
          } else if (state.maxResponseTokens <= 1000) {
            prompt += '\n- Keep explanations very brief';
            prompt += '\n- Focus on essential information only';
            prompt += '\n- For code: include only key parts';
          }
        }
        
        if (state.preferBulletPoints) {
          prompt += '\n- Prefer bullet points over paragraphs for multiple items';
        }
        
        if (!state.includeCodeComments && state.responseStyle !== 'detailed') {
          prompt += '\n- Provide code without extensive comments unless specifically requested';
        }
        
        return prompt;
      }
    }),
    {
      name: 'response-preferences',
    }
  )
);
use std::collections::HashMap;

/// Known Ollama models and their function calling capabilities
pub struct OllamaModelInfo {
    pub supports_functions: bool,
    pub context_window: usize,
    pub description: &'static str,
}

lazy_static::lazy_static! {
    pub static ref OLLAMA_MODELS: HashMap<&'static str, OllamaModelInfo> = {
        let mut m = HashMap::new();
        
        // Models that support function calling
        m.insert("mistral", OllamaModelInfo {
            supports_functions: true,
            context_window: 8192,
            description: "Mistral 7B - Good function calling support"
        });
        
        m.insert("mixtral", OllamaModelInfo {
            supports_functions: true,
            context_window: 32768,
            description: "Mixtral 8x7B - Excellent function calling"
        });
        
        m.insert("llama3.1", OllamaModelInfo {
            supports_functions: true,
            context_window: 8192,
            description: "Llama 3.1 - Best function calling support"
        });
        
        m.insert("qwen2.5", OllamaModelInfo {
            supports_functions: true,
            context_window: 32768,
            description: "Qwen 2.5 - Good function support"
        });
        
        m.insert("command-r", OllamaModelInfo {
            supports_functions: true,
            context_window: 128000,
            description: "Command-R - Designed for tool use"
        });
        
        // Models that DO NOT support function calling
        m.insert("llama2", OllamaModelInfo {
            supports_functions: false,
            context_window: 4096,
            description: "Llama 2 - No function calling"
        });
        
        m.insert("codellama", OllamaModelInfo {
            supports_functions: false,
            context_window: 4096,
            description: "Code Llama - No function calling"
        });
        
        m.insert("deepseek-coder", OllamaModelInfo {
            supports_functions: false,
            context_window: 16384,
            description: "DeepSeek Coder - No function calling"
        });
        
        m
    };
}

/// Check if a model supports function calling
pub fn model_supports_functions(model_name: &str) -> bool {
    // Check exact match first
    if let Some(info) = OLLAMA_MODELS.get(model_name) {
        return info.supports_functions;
    }
    
    // Check if it's a variant (e.g., "mistral:latest" or "llama3.1:70b")
    let base_name = model_name.split(':').next().unwrap_or(model_name);
    if let Some(info) = OLLAMA_MODELS.get(base_name) {
        return info.supports_functions;
    }
    
    // For unknown models, check if they contain known function-capable model names
    let lower = model_name.to_lowercase();
    if lower.contains("mistral") || 
       lower.contains("mixtral") || 
       lower.contains("llama3.1") || 
       lower.contains("llama-3.1") ||
       lower.contains("qwen2.5") ||
       lower.contains("qwen-2.5") ||
       lower.contains("command-r") {
        return true;
    }
    
    // Default to false for unknown models
    false
}
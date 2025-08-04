use super::FunctionCall;
use regex::Regex;
use serde_json;

/// Parse Deepseek's custom tool calling format
pub fn parse_deepseek_tool_calls(content: &str) -> Option<Vec<FunctionCall>> {
    // Look for [TOOL_REQUEST]...[END_TOOL_REQUEST] patterns
    let re = Regex::new(r"\[TOOL_REQUEST\](.*?)\[END_TOOL_REQUEST\]").ok()?;
    
    let mut function_calls = Vec::new();
    
    for cap in re.captures_iter(content) {
        if let Some(json_str) = cap.get(1) {
            // Try to parse the JSON inside the tool request
            if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(json_str.as_str().trim()) {
                if let (Some(name), Some(args)) = (
                    parsed.get("name").and_then(|n| n.as_str()),
                    parsed.get("arguments")
                ) {
                    function_calls.push(FunctionCall {
                        name: name.to_string(),
                        arguments: args.to_string(),
                    });
                }
            }
        }
    }
    
    if function_calls.is_empty() {
        None
    } else {
        Some(function_calls)
    }
}

/// Extract content without tool requests for display
pub fn extract_content_without_tools(content: &str) -> String {
    // Remove [TOOL_REQUEST]...[END_TOOL_REQUEST] blocks
    let re = Regex::new(r"\[TOOL_REQUEST\].*?\[END_TOOL_REQUEST\]").unwrap();
    let without_tools = re.replace_all(content, "").trim().to_string();
    
    // Also remove <think>...</think> blocks if present
    let think_re = Regex::new(r"<think>.*?</think>").unwrap();
    think_re.replace_all(&without_tools, "").trim().to_string()
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_parse_deepseek_tool_calls() {
        let content = r#"Let me check that for you.
[TOOL_REQUEST]
{"name": "get_current_time", "arguments": {}}
[END_TOOL_REQUEST]"#;
        
        let calls = parse_deepseek_tool_calls(content).unwrap();
        assert_eq!(calls.len(), 1);
        assert_eq!(calls[0].name, "get_current_time");
        assert_eq!(calls[0].arguments, "{}");
    }
    
    #[test]
    fn test_extract_content_without_tools() {
        let content = r#"<think>
Some thinking
</think>
Let me check that for you.
[TOOL_REQUEST]
{"name": "get_current_time", "arguments": {}}
[END_TOOL_REQUEST]"#;
        
        let clean = extract_content_without_tools(content);
        assert_eq!(clean, "Let me check that for you.");
    }
}
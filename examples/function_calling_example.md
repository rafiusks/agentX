# Function Calling Example for AgentX

## Overview

AgentX now supports OpenAI function calling, allowing the AI to request specific actions be taken. This feature is integrated into the backend and frontend.

## Backend Support

The Rust backend has been updated to support function calling:

1. **Provider Interface** - Added `Function`, `FunctionCall`, and `ToolChoice` types
2. **OpenAI Provider** - Full implementation of function calling in requests and responses
3. **Streaming Support** - Function calls can be streamed with `function_call_delta`

## Frontend Support

The React frontend displays function calls with a dedicated UI component:

1. **FunctionCall Component** - Shows function name and formatted arguments
2. **ChatMessage Updates** - Displays function role messages with special icon
3. **Inline Display** - Function calls appear inline with assistant responses

## Example Usage

To test function calling with OpenAI:

1. Set your OpenAI API key in Settings
2. Use a model that supports functions (e.g., gpt-3.5-turbo, gpt-4)
3. The provider will need to be configured with functions

### Example Function Definition

```rust
let functions = vec![
    Function {
        name: "get_weather".to_string(),
        description: "Get the current weather in a given location".to_string(),
        parameters: serde_json::json!({
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "The city and state, e.g. San Francisco, CA"
                },
                "unit": {
                    "type": "string", 
                    "enum": ["celsius", "fahrenheit"]
                }
            },
            "required": ["location"]
        }),
    }
];
```

### Response Handling

When the AI decides to call a function, you'll see:
- A function call indicator in the chat
- The function name and arguments displayed
- Support for executing the function and returning results

## Next Steps

To fully utilize function calling:

1. Define your functions in the completion request
2. Handle function call responses from the AI
3. Execute the requested functions
4. Send function results back to continue the conversation

This creates a powerful loop where the AI can request actions and process their results.
# Testing Function Calling in AgentX

## Prerequisites

1. **LM Studio** running on http://localhost:1234 (or update the endpoint in Settings)
2. A model that supports function calling loaded in LM Studio:
   - Recommended: Qwen models, Mistral models, or any model that supports OpenAI function calling format
   - The model should be configured to enable function calling in LM Studio settings

## Quick Test

1. **Start AgentX**: 
   ```bash
   npm run tauri:dev
   ```

2. **Open the test page**: 
   - In a browser, open `file:///path/to/agentx/test_functions.html`
   - Or use the browser console with the test script

3. **Verify Settings**:
   - Go to Settings in AgentX
   - Check that "Local Models" shows the correct endpoint (http://localhost:1234)
   - Click "Refresh Models" to discover available models
   - Select a model that supports function calling

## Testing Methods

### Method 1: Test HTML Page
Open `test_functions.html` in a browser while AgentX is running. This provides:
- Visual feedback of function calls
- Stream output display
- Pre-configured test messages

### Method 2: Browser Console
```javascript
// In the AgentX app, open browser console (F12)

// Test function calling
await window.__TAURI__.invoke('test_function_calling', {
    message: "What's the weather in San Francisco?"
});

// Listen for results
await window.__TAURI__.event.listen('test-function-stream', console.log);
```

### Method 3: Direct Chat
1. In the AgentX chat interface, try messages like:
   - "What's the weather in Tokyo?"
   - "Calculate 15 * 27"
   - "Tell me the weather in Paris and New York"

## What to Expect

When function calling works correctly:
1. The model will recognize when to use a function
2. You'll see function call JSON in the logs/output
3. The function will be "executed" (simulated in test mode)
4. The model will use the function result in its response

## Debugging

Enable debug output by checking the terminal where you ran `npm run tauri:dev`:
- Look for `[OpenAI-Compatible]` log lines
- Check for "Functions: X functions included" messages
- Verify the model supports function calling

## Common Issues

1. **No function calls detected**: 
   - Model doesn't support function calling
   - Model needs specific prompting to use functions
   - Function definitions aren't being sent properly

2. **Connection errors**:
   - LM Studio not running
   - Wrong port/endpoint
   - Model not loaded in LM Studio

3. **Parsing errors**:
   - Model using non-standard function call format
   - Response format differs from OpenAI spec
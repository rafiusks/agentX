# Ollama Models with Function Calling Support

As of late 2024, the following Ollama models support function/tool calling:

## Models with Native Function Support:

1. **llama3.1** (8B, 70B, 405B)
   - Best function calling support
   - Use: `ollama pull llama3.1`

2. **mistral** (7B) 
   - Good function support
   - Use: `ollama pull mistral`

3. **mixtral** (8x7B)
   - Excellent function support
   - Use: `ollama pull mixtral`

4. **qwen2.5** (0.5B to 72B)
   - Good function support
   - Use: `ollama pull qwen2.5`

5. **command-r** (35B)
   - Designed for tool use
   - Use: `ollama pull command-r`

## Testing Function Support:

To test if your model supports functions, try this command:

```bash
curl -X POST http://localhost:11434/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1",
    "messages": [{"role": "user", "content": "What time is it?"}],
    "tools": [{
      "type": "function",
      "function": {
        "name": "get_time",
        "description": "Get current time"
      }
    }]
  }'
```

If the model supports functions, you'll see a `tool_calls` field in the response.

## Important Notes:

- **llama2** does NOT support function calling
- **codellama** does NOT support function calling
- Models need to be specifically trained for function calling
- Larger models generally perform better at function calling

## Recommended Setup:

For AgentX with MCP support, use:
```bash
ollama pull llama3.1  # or mistral for smaller size
```
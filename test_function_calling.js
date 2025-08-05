// Test script for function calling
// Run this in the browser console when AgentX is running

async function testFunctionCalling() {
    console.log("Testing function calling with LM Studio...");
    
    // First, let's add a simple MCP server with test functions
    try {
        // Add a demo MCP server (you might need to adjust this based on your setup)
        await window.__TAURI__.invoke('add_mcp_server', {
            name: 'test-functions',
            command: 'node',
            args: ['/path/to/mcp-server.js'] // You'll need to create this
        });
        console.log("✓ MCP server added");
    } catch (e) {
        console.log("Note: MCP server may already be added or not available");
    }
    
    // Test 1: Send a message that should trigger function calling
    console.log("\nTest 1: Weather query (should trigger function call)");
    try {
        const response = await window.__TAURI__.invoke('send_message', {
            message: "What's the weather like in San Francisco?",
            providerId: "local"
        });
        console.log("Response:", response);
    } catch (e) {
        console.error("Error:", e);
    }
    
    // Test 2: Stream a message with function calling
    console.log("\nTest 2: Calculator query (streaming with function call)");
    try {
        // For streaming, we need to listen to events
        const unlisten = await window.__TAURI__.event.listen('stream-chunk', (event) => {
            console.log("Stream chunk:", event.payload);
        });
        
        await window.__TAURI__.invoke('stream_message', {
            message: "Calculate the sum of 42 and 58",
            providerId: "local"
        });
        
        // Clean up listener after a delay
        setTimeout(() => unlisten(), 5000);
    } catch (e) {
        console.error("Error:", e);
    }
    
    // Test 3: Check available functions
    console.log("\nTest 3: List available MCP tools");
    try {
        const tools = await window.__TAURI__.invoke('list_mcp_tools');
        console.log("Available tools:", tools);
    } catch (e) {
        console.error("Error:", e);
    }
}

// Helper to test without MCP - using hardcoded test functions
async function testWithHardcodedFunctions() {
    console.log("Testing with hardcoded functions...");
    
    // This would require modifying the backend to add test functions
    // For now, let's just test basic streaming
    try {
        const unlisten = await window.__TAURI__.event.listen('stream-chunk', (event) => {
            const chunk = event.payload;
            if (chunk.function_call) {
                console.log("Function call detected:", chunk.function_call);
            } else {
                console.log("Content:", chunk.content);
            }
        });
        
        await window.__TAURI__.invoke('stream_message', {
            message: "If I had a get_weather function, what's the weather in Paris?",
            providerId: "local"
        });
        
        setTimeout(() => unlisten(), 5000);
    } catch (e) {
        console.error("Error:", e);
    }
}

// Run the tests
console.log("=== AgentX Function Calling Test ===");
console.log("Make sure LM Studio is running on http://localhost:1234");
console.log("Model should support function calling (e.g., Qwen, Mistral, etc.)");
console.log("\nRun: testFunctionCalling() or testWithHardcodedFunctions()");
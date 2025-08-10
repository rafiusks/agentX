#!/usr/bin/env node

// Simple test for the MCP server
const { spawn } = require('child_process');
const path = require('path');

// Start the MCP server
const serverPath = path.join(__dirname, 'dist', 'index.js');
const server = spawn('node', [serverPath], {
  stdio: ['pipe', 'pipe', 'pipe'],
});

let responseData = '';

server.stdout.on('data', (data) => {
  responseData += data.toString();
  
  // Look for complete JSON responses
  const lines = responseData.split('\n');
  for (const line of lines) {
    if (line.trim()) {
      try {
        const response = JSON.parse(line);
        console.log('Received:', JSON.stringify(response, null, 2));
      } catch (e) {
        // Not JSON, ignore
      }
    }
  }
});

server.stderr.on('data', (data) => {
  console.error('Server stderr:', data.toString());
});

server.on('close', (code) => {
  console.log(`Server process exited with code ${code}`);
});

// Send initialization request
setTimeout(() => {
  const initRequest = {
    jsonrpc: '2.0',
    id: 1,
    method: 'initialize',
    params: {
      protocolVersion: '2024-11-05',
      capabilities: {
        roots: {
          listChanged: true,
        },
        sampling: {},
      },
      clientInfo: {
        name: 'test-client',
        version: '1.0.0',
      },
    },
  };
  
  console.log('Sending init request...');
  server.stdin.write(JSON.stringify(initRequest) + '\n');
}, 1000);

// Send list tools request
setTimeout(() => {
  const toolsRequest = {
    jsonrpc: '2.0',
    id: 2,
    method: 'tools/list',
    params: {},
  };
  
  console.log('Sending tools list request...');
  server.stdin.write(JSON.stringify(toolsRequest) + '\n');
}, 2000);

// Clean shutdown after 5 seconds
setTimeout(() => {
  console.log('Shutting down...');
  server.kill('SIGTERM');
  process.exit(0);
}, 5000);
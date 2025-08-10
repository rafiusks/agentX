#!/bin/bash

# Simple MCP test server - responds to initialize request

while IFS= read -r line; do
    if echo "$line" | grep -q '"method":"initialize"'; then
        echo '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{},"resources":{},"logging":{}},"serverInfo":{"name":"code-rag-mcp","version":"1.0.0"}}}'
    fi
done
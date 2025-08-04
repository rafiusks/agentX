#!/usr/bin/env node

// Minimal test - just echo back whatever we receive
process.stdin.setEncoding('utf8');
process.stdout.setEncoding('utf8');

process.stdin.on('data', function(data) {
  process.stderr.write(`Received: ${data.trim()}\n`);
  process.stdout.write(`Echo: ${data}`);
  process.stdout.flush && process.stdout.flush();
});

process.stderr.write('Minimal test server starting...\n');
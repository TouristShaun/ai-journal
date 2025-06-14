#!/bin/bash

# Test script for real-time entry processing updates

echo "Testing real-time journal entry processing..."
echo ""

# Create a test entry
echo "Creating a new journal entry..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "journal.create",
    "params": {
      "content": "Today I learned about Server-Sent Events (SSE) and implemented real-time updates in the journal app. This is a test entry to verify that the processing status updates are working correctly. Check out https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events for more information."
    },
    "id": 1
  }')

echo "Response: $RESPONSE"
echo ""

# Extract entry ID
ENTRY_ID=$(echo $RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created entry with ID: $ENTRY_ID"
echo ""

echo "Open the frontend in your browser and watch the entry get processed in real-time!"
echo "The entry should show:"
echo "1. 'Processing...' status initially"
echo "2. Update to 'Processed' with AI analysis when complete"
echo ""
echo "You can also monitor SSE events with:"
echo "curl -N http://localhost:8080/api/events"
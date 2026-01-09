#!/bin/bash
# Mock Claude CLI for testing
# Echoes back input and outputs completion marker

echo "Mock Agent Started"
echo "Ready for input..."

while IFS= read -r line; do
    echo "Received: $line"
    sleep 0.5
    echo "Processing..."
    sleep 0.5
    echo "Task complete"
    echo "### TASK_DONE ###"
done

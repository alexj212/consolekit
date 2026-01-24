#!/bin/bash
# Test script to check if prompt appears
# Returns 0 if working (good), 1 if broken (bad)

cd examples/simple

# Run the app and send a newline after 0.5 seconds
# If prompt appears, output will show "> " before the newline
# If prompt doesn't appear, there will be no "> " in output
output=$(timeout 1 bash -c 'sleep 0.5; echo; sleep 0.3' | go run . 2>&1)

# Check if the prompt "> " appeared in the output
if echo "$output" | grep -q "> "; then
    echo "GOOD: Prompt appeared"
    exit 0
else
    echo "BAD: Prompt did not appear"
    echo "Output was: $output"
    exit 1
fi

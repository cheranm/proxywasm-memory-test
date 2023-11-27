#!/bin/bash

# Run the command 5 times with a 20-second pause between executions
for i in {1..5}; do
    echo "Running iteration $i"
    $HEY_PATH -z 12s -c 10 -q 120 http://localhost:8080
    sleep 20
done

echo "Test completed."

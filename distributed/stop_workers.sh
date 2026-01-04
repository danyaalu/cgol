#!/bin/bash

# Script to stop workers on remote nodes by killing tmux sessions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/nodes.json"

# Check if sshpass is installed
if ! command -v sshpass &> /dev/null; then
    echo "Error: sshpass is not installed."
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed."
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file not found: $CONFIG_FILE"
    exit 1
fi

echo "=========================================="
echo "Stopping workers on remote nodes"
echo "=========================================="
echo

# Get number of nodes
NODE_COUNT=$(jq '.nodes | length' "$CONFIG_FILE")

# Loop through each node
for i in $(seq 0 $(($NODE_COUNT - 1))); do
    # Extract node configuration
    HOSTNAME=$(jq -r ".nodes[$i].hostname" "$CONFIG_FILE")
    USERNAME=$(jq -r ".nodes[$i].username" "$CONFIG_FILE")
    PASSWORD=$(jq -r ".nodes[$i].password" "$CONFIG_FILE")
    
    echo "[$((i+1))/$NODE_COUNT] Stopping worker on $USERNAME@$HOSTNAME..."
    
    # SSH command to kill tmux session
    sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$USERNAME@$HOSTNAME" "
        tmux kill-session -t cgol 2>/dev/null && echo 'Worker stopped' || echo 'No worker found'
    " &
    
    PIDS[$i]=$!
done

# Wait for all SSH connections to complete
echo
echo "Waiting for all connections to complete..."
for pid in ${PIDS[@]}; do
    wait $pid
done

echo
echo "=========================================="
echo "All workers stopped!"
echo "=========================================="

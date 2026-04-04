#!/bin/bash

# Script to start workers on remote nodes via SSH in tmux sessions
# This script reads configuration from nodes.json and starts workers on each node

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/nodes.json"

# Check if sshpass is installed
if ! command -v sshpass &> /dev/null; then
    echo "Error: sshpass is not installed. Please install it first:"
    echo "  Ubuntu/Debian: sudo apt-get install sshpass"
    echo "  Fedora/RHEL: sudo dnf install sshpass"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed. Please install it first:"
    echo "  Ubuntu/Debian: sudo apt-get install jq"
    echo "  Fedora/RHEL: sudo dnf install jq"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file not found: $CONFIG_FILE"
    exit 1
fi

# Read coordinator info
COORDINATOR_IP=$(jq -r '.coordinator.ip' "$CONFIG_FILE")
COORDINATOR_PORT=$(jq -r '.coordinator.port' "$CONFIG_FILE")

echo "=========================================="
echo "Starting workers on remote nodes"
echo "Coordinator: http://$COORDINATOR_IP:$COORDINATOR_PORT"
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
    THREADS=$(jq -r ".nodes[$i].threads" "$CONFIG_FILE")
    
    echo "[$((i+1))/$NODE_COUNT] Connecting to $USERNAME@$HOSTNAME..."
    
    # SSH command to create tmux session and start worker
    # Using sshpass for password authentication (insecure but fine for local dev)
    sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$USERNAME@$HOSTNAME" "
        # Kill existing tmux session if it exists
        tmux kill-session -t cgol 2>/dev/null || true
        
        # Create new tmux session and start worker
        tmux new-session -d -s cgol \"cd ~/cgol/distributed && ./bin/worker --server http://$COORDINATOR_IP:$COORDINATOR_PORT --threads $THREADS\"
        
        echo 'Worker started in tmux session: cgol'
    " &
    
    # Store the PID to wait for all SSH connections
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
echo "All workers started successfully!"
echo "=========================================="
echo
echo "To view worker logs on a node, SSH in and run:"
echo "  tmux attach -t cgol"
echo
echo "To stop all workers, run:"
echo "  ./stop_workers.sh"

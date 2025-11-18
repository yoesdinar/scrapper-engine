#!/bin/bash

# Quick start script - runs all three services in tmux

if ! command -v tmux &> /dev/null; then
    echo "tmux is not installed. Please install it first:"
    echo "  macOS: brew install tmux"
    echo "  Ubuntu: apt-get install tmux"
    exit 1
fi

SESSION="config-mgmt"

# Kill existing session if it exists
tmux kill-session -t ${SESSION} 2>/dev/null

# Create new session
tmux new-session -d -s ${SESSION} -n "services"

# Split into 3 panes
tmux split-window -h -t ${SESSION}:0
tmux split-window -v -t ${SESSION}:0.1

# Run controller in first pane
tmux send-keys -t ${SESSION}:0.0 "cd controller && ./controller" C-m

# Run worker in second pane
tmux send-keys -t ${SESSION}:0.1 "cd worker && ./worker" C-m

# Run agent in third pane
tmux send-keys -t ${SESSION}:0.2 "cd agent && ./agent" C-m

# Attach to session
tmux attach-session -t ${SESSION}

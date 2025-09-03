#!/bin/bash

NODES_FILE="scripts/NODES.TXT"
MODELS_FILE="scripts/MODELS.TXT"

# Function to pull models on a single node sequentially
pull_models_on_node() {
    local node=$1
    echo "Starting model pulls for $node..."
    
    # Iterate through each model for the current node
    for model in "${MODELS[@]}"; do
        echo "  -> Pulling model: $model on $node"
        curl --insecure https://$node/api/pull -d '{ "name": "'"$model"'" }'
        echo ""
    done
}

load_node_names() {
  if [[ ! -f "$NODES_FILE" ]]; then
    echo "Error: Required file ${NODES_FILE} not found." >&2
    exit 1
  fi
  
  readarray -t NODES < "$NODES_FILE"
  echo "Loaded ${#NODES[@]} nodes from ${NODES_FILE}."
}

load_model_names() {
  if [[ ! -f "$MODELS_FILE" ]]; then
    echo "Error: Required file ${MODELS_FILE} not found." >&2
    exit 1
  fi
  
  readarray -t MODELS < "$MODELS_FILE"
  echo "Loaded ${#MODELS[@]} models from ${MODELS_FILE}."
}

# Function to list models on a single node
list_models() {
    local node=$1
    echo "--- Models on $node ---"
    if curl --fail --silent --insecure https://$node/api/tags > /dev/null; then
        curl --insecure https://$node/api/tags | jq -r '.models[].name'
    else
        echo "Could not list models: Ollama is not accessible on $node."
    fi
    echo ""
}

load_node_names
load_model_names

# --- Parallel Pull Requests ---
echo "Starting parallel model pulls across all nodes..."
for node in "${NODES[@]}"; do
    # Run the model pull function for each node in the background
    pull_models_on_node "$node" &
done

# Wait for all background processes to complete
wait

echo "----------------------------------------"
echo "All model pull commands have finished. Now listing available models on each node."
echo "========================================"

# List models on each node after all pulls are complete
for node in "${NODES[@]}"; do
    list_models "$node"
done
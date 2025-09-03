#!/bin/bash

# Define file paths
NODES_FILE="scripts/NODES.TXT"
MODELS_FILE="scripts/MODELS.TXT"

# Function to load node names from file
load_node_names() {
    if [[ ! -f "$NODES_FILE" ]]; then
        echo "Error: Required file ${NODES_FILE} not found." >&2
        exit 1
    fi
    readarray -t NODES < "$NODES_FILE"
    echo "Loaded ${#NODES[@]} nodes from ${NODES_FILE}."
}

# Function to load model names from file
load_model_names() {
    if [[ ! -f "$MODELS_FILE" ]]; then
        echo "Error: Required file ${MODELS_FILE} not found." >&2
        exit 1
    fi
    readarray -t MODELS_TO_KEEP < "$MODELS_FILE"
    echo "Loaded ${#MODELS_TO_KEEP[@]} models from ${MODELS_FILE}."
}

# Function to delete models on a single node sequentially
delete_models_on_node() {
    local node=$1
    echo "Starting model cleanup for $node..."
    
    # Get the list of all models currently on the node
    local current_models
    current_models=$(curl --fail --silent --insecure https://$node/api/tags | jq -r '.models[].name')

    # Convert the list of models to keep into a single string for easier searching
    local models_to_keep_string="${MODELS_TO_KEEP[@]}"

    # Iterate through each currently installed model
    while read -r current_model; do
        # Check if the current model is NOT in the list of models to keep
        if [[ ! " ${models_to_keep_string} " =~ " ${current_model} " ]]; then
            echo "  -> Deleting model: $current_model on $node"
            curl --insecure -X DELETE https://$node/api/delete -d "{ \"model\": \"$current_model\" }"
            echo ""
        else
            echo "  -> Keeping model: $current_model on $node"
        fi
    done <<< "$current_models"
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

# Main script execution
load_node_names
load_model_names

# --- Parallel Delete Requests ---
echo "Starting parallel model cleanup across all nodes..."
for node in "${NODES[@]}"; do
    # Run the delete function for each node in the background
    delete_models_on_node "$node" &
done

# Wait for all background processes to complete
wait

echo "----------------------------------------"
echo "All model cleanup commands have finished. Now listing available models on each node."
echo "========================================"

# List models on each node after all deletes are complete
for node in "${NODES[@]}"; do
    list_models "$node"
done
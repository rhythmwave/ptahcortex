#!/bin/bash
# Run OpenCode benchmark with GLM-5.1
# Usage: ./run-opencode.sh [model]

set -e

MODEL="${1:-glm-5.1}"
RESULTS_DIR="$(dirname "$0")/results"
OPENCODE_CONFIG="/tmp/opencode-benchmark-config.json"

mkdir -p "$RESULTS_DIR"

# Create OpenCode config with GLM-5.1
cat > "$OPENCODE_CONFIG" << EOF
{
    "provider": {
        "type": "openai",
        "model": "$MODEL",
        "apiKey": "sk-bzu08x_omiQ_lj5ZL2IHcg",
        "baseUrl": "https://ai.sumopod.com/v1"
    }
}
EOF

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "OpenCode Benchmark - Model: $MODEL"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if opencode is installed
if ! command -v opencode &> /dev/null; then
    echo "OpenCode not installed. Installing..."
    go install github.com/opencode-ai/opencode@latest
fi

# Tasks to run
declare -A TASKS
TASKS[1]="List all Go files in the project"
TASKS[2]="Find all functions named Call or Start in the codebase"
TASKS[3]="Review error handling in the MCP client code"
TASKS[4]="Trace all callers of the CallTool function"
TASKS[5]="Find potential race conditions in concurrent code"

run_opencode_task() {
    local task_num=$1
    local task_desc="${TASKS[$task_num]}"
    local result_file="$RESULTS_DIR/opencode_${MODEL}_task${task_num}_$(date +%Y%m%d_%H%M%S).json"
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "TASK $task_num: $task_desc"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local start_time=$(date +%s%N)
    
    # Run opencode with the task
    # Note: opencode uses interactive mode, so we pipe the task
    local output=$(echo "$task_desc" | timeout 180 opencode --config "$OPENCODE_CONFIG" 2>&1 || true)
    
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Count tokens (approximate from output)
    local token_count=$(echo "$output" | wc -c | awk '{print int($1/4)}')  # rough estimate
    
    # Create result
    cat > "$result_file" << EOF
{
    "agent": "opencode",
    "model": "$MODEL",
    "task_num": $task_num,
    "task": "$task_desc",
    "timestamp": "$(date -Iseconds)",
    "metrics": {
        "estimated_tokens": $token_count,
        "duration_ms": $duration_ms
    },
    "output_preview": "$(echo "$output" | head -20 | tr '\n' ' ' | sed 's/"/\\"/g')"
}
EOF
    
    echo "Duration: ${duration_ms}ms"
    echo "Output preview: $(echo "$output" | head -5)"
    echo "Result saved to: $result_file"
}

# Run tasks
if [ "${2:-all}" = "all" ]; then
    for i in 1 2 3 4 5; do
        run_opencode_task $i
    done
else
    run_opencode_task "$2"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "OpenCode BENCHMARK COMPLETE"
echo "Results in: $RESULTS_DIR"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

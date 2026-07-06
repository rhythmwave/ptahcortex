#!/bin/bash
# Run Ptahcortex benchmark
# Usage: ./run-ptahcortex.sh [model]

set -e

MODEL="${1:-glm-5.1}"
RESULTS_DIR="$(dirname "$0")/results"
CONFIG_TEMPLATE="$(dirname "$0")/ptahcortex-config.yaml"

mkdir -p "$RESULTS_DIR"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Ptahcortex Benchmark - Model: $MODEL"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create config
CONFIG="/tmp/ptahcortex-benchmark-${MODEL}.yaml"
sed "s/{{MODEL}}/$MODEL/g" "$CONFIG_TEMPLATE" > "$CONFIG"

# Tasks
declare -A TASKS
TASKS[1]="List all Go files in the project"
TASKS[2]="Find all functions named Call or Start in the codebase"
TASKS[3]="Review error handling in the MCP client code"
TASKS[4]="Trace all callers of the CallTool function"
TASKS[5]="Find potential race conditions in concurrent code"

run_task() {
    local task_num=$1
    local task_desc="${TASKS[$task_num]}"
    local result_file="$RESULTS_DIR/ptahcortex_${MODEL}_task${task_num}_$(date +%Y%m%d_%H%M%S).json"
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "TASK $task_num: $task_desc"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local start_time=$(date +%s%N)
    local output=$(ssh deploy@202.43.249.114 "ptahcortex --config /opt/ptahcortex/configs/benchmark.yaml --task '$task_desc'" 2>&1)
    local exit_code=$?
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Extract metrics
    local total_tokens=$(echo "$output" | grep -oP 'total tokens: \K[0-9]+' | tail -1)
    local iterations=$(echo "$output" | grep -oP 'iteration=\K[0-9]+' | tail -1)
    local tool_calls=$(echo "$output" | grep -oP 'sandbox: [0-9]+ tokens \(\K[0-9]+' | tail -1)
    local plan_tokens=$(echo "$output" | grep -oP 'plan:    \K[0-9]+' | tail -1)
    local sandbox_tokens=$(echo "$output" | grep -oP 'sandbox: \K[0-9]+' | tail -1)
    local reflect_tokens=$(echo "$output" | grep -oP 'reflect: \K[0-9]+' | tail -1)
    local final_tokens=$(echo "$output" | grep -oP 'final:   \K[0-9]+' | tail -1)
    
    # Get final answer
    local final_answer=$(echo "$output" | sed -n '/═══.*FINAL/,/═══.*RUN COMPLETE/p' | head -n -1)
    
    # Create JSON
    cat > "$result_file" << EOF
{
    "agent": "ptahcortex",
    "model": "$MODEL",
    "task_num": $task_num,
    "task": "$task_desc",
    "timestamp": "$(date -Iseconds)",
    "metrics": {
        "total_tokens": ${total_tokens:-0},
        "iterations": ${iterations:-0},
        "tool_calls": ${tool_calls:-0},
        "duration_ms": $duration_ms,
        "plan_tokens": ${plan_tokens:-0},
        "sandbox_tokens": ${sandbox_tokens:-0},
        "reflect_tokens": ${reflect_tokens:-0},
        "final_tokens": ${final_tokens:-0}
    },
    "exit_code": $exit_code
}
EOF
    
    echo "Total Tokens: ${total_tokens:-N/A}"
    echo "Iterations: ${iterations:-N/A}"
    echo "Tool Calls: ${tool_calls:-N/A}"
    echo "Duration: ${duration_ms}ms"
    echo "Result: $result_file"
}

# Run tasks
if [ "${2:-all}" = "all" ]; then
    for i in 1 2 3 4 5; do
        run_task $i
    done
else
    run_task "$2"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Ptahcortex BENCHMARK COMPLETE"
echo "Results in: $RESULTS_DIR"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

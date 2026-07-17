#!/usr/bin/env python3
import os
import re
import sys

def parse_logs():
    artifact_dir = "/home/sprite/.gemini/antigravity-cli/brain/aa5a7f13-7d7d-42b7-bbd8-cd055443d7a8"
    exec_log_path = os.path.join(artifact_dir, "test_executions.log")
    debug_log_path = os.path.join(artifact_dir, "clare_debug.log")

    if not os.path.exists(exec_log_path):
        print(f"error: execution log missing")
        return

    # Parse executions to find failures
    failures = []
    with open(exec_log_path, "r") as f:
        content = f.read()

    # Look for FAIL markers in test executions
    # Title loops print progress. We can look at Title summaries.
    # We can also parse the debug log for errors
    print("=== CI FLOW FEEDBACK SUMMARY ===")
    
    # Read debug log for errors/warnings
    if os.path.exists(debug_log_path):
        print("\n--- error patterns in clare_debug.log ---")
        errors = set()
        with open(debug_log_path, "r") as f:
            for line in f:
                if "[ERROR]" in line or "[WARN]" in line:
                    # Clean up timestamp
                    clean = re.sub(r"^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] ", "", line.strip())
                    errors.add(clean)
        
        for err in sorted(list(errors))[:15]:
            print(f"- {err}")

    # Check for direct mpv/ytdlp exits
    print("\n--- process exit codes ---")
    with open(exec_log_path, "r") as f:
        for line in f:
            if "EXIT" in line:
                print(line.strip())

if __name__ == "__main__":
    parse_logs()

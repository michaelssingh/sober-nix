#!/usr/bin/env bash
# Continuous improvement runner for Clare TUI

echo "Starting CI verification run..."

# Rebuild clare binary
(cd packages/clare && go build -o clare .)

# Run the 20-anime test suite
bash bin/test-20-anime.sh

EXIT_CODE=$?

# Run parser to expose actionable feedback
python3 bin/parse-results.py

echo "Test suite execution finished with code: $EXIT_CODE"
exit $EXIT_CODE

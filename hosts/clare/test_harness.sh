#!/usr/bin/env bash
# Autonomous Verification Harness for Clare
set -euo pipefail

# Setup environment
export SOJU_DB="/tmp/soju_test.db"
export SOJU_ADMIN_SOCK="/tmp/soju_admin.sock"
export API_PORT="8081"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up..."
    [ -f "$SOJU_DB" ] && rm "$SOJU_DB"
    [ -S "$SOJU_ADMIN_SOCK" ] && rm "$SOJU_ADMIN_SOCK"
    kill $(jobs -p) 2>/dev/null || true
}
trap cleanup EXIT

# 1. Start a mock Soju service (minimal listener for admin socket)
# Since we can't easily start the real soju here, we'll create a mock socket server
python3 -c "
import socket, os
if os.path.exists('$SOJU_ADMIN_SOCK'): os.remove('$SOJU_ADMIN_SOCK')
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.bind('$SOJU_ADMIN_SOCK')
s.listen(1)
while True:
    conn, _ = s.accept()
    data = conn.recv(1024)
    if b'user create' in data:
        conn.sendall(b'NOTICE BouncerServ :created user testuser\r\n')
    conn.close()
" &
sleep 1

# 2. Start the API server
cd api && go run main.go &
sleep 2

# 3. Perform test registration
echo "🧪 Running registration test..."
RESPONSE=$(curl -s -X POST http://localhost:$API_PORT/api/v1/users -d '{"username": "testuser", "password": "password"}' -H "Content-Type: application/json")

if echo "$RESPONSE" | grep -q "User created successfully"; then
    echo "✅ Registration test passed!"
else
    echo "❌ Registration test failed: $RESPONSE"
    exit 1
fi

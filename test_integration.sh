#!/usr/bin/env bash
mkdir -p /tmp/soju_test
echo "listen unix+admin:///tmp/soju_test/admin.sock" > /tmp/soju_test/soju.conf
# Run soju in the background, redirecting output for debugging
soju -config /tmp/soju_test/soju.conf > /tmp/soju.log 2>&1 &
SOJU_PID=$!
echo "Soju started (PID: $SOJU_PID)"

# Wait for the admin socket to be created
for i in {1..5}; do
    if [ -S /tmp/soju_test/admin.sock ]; then
        echo "Socket created."
        break
    fi
    echo "Waiting for socket..."
    sleep 1
done

# Run raki-api
./raki-api -socket /tmp/soju_test/admin.sock -listen :8082 -api-keys "test-key" > /tmp/raki.log 2>&1 &
API_PID=$!
sleep 2

echo "--- Testing Health ---"
curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/health
echo ""

echo "--- Testing List Users (Needs Auth) ---"
curl -s -H "X-API-Key: test-key" http://localhost:8082/api/users
echo ""

# Inspect logs
echo "--- Soju Log ---"
cat /tmp/soju.log
echo "--- Raki Log ---"
cat /tmp/raki.log

# Cleanup
kill $API_PID $SOJU_PID
rm -rf /tmp/soju_test /tmp/soju.log /tmp/raki.log

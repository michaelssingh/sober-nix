#!/usr/bin/env bash
touch /tmp/soju_admin.sock
./raki-api -socket /tmp/soju_admin.sock -listen :8082 -api-keys "test-key" &
API_PID=$!
sleep 2
STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/health)
echo "Health check status: $STATUS_CODE"
AUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/api/users)
echo "Unauthenticated user request status (should be 401): $AUTH_CODE"
kill $API_PID
rm /tmp/soju_admin.sock

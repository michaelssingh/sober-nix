#!/usr/bin/env python3
import urllib.request
import urllib.error
import json
import sys

API_BASE = "https://sober-clare.fly.dev:8081/api"
HEALTH_URL = "https://sober-clare.fly.dev:8081/health"
API_KEY = "test-api-key"

def request(method, path, data=None):
    url = f"{API_BASE}{path}" if not path.startswith("http") else path
    headers = {"X-API-Key": API_KEY}
    if data:
        data = json.dumps(data).encode('utf-8')
        headers["Content-Type"] = "application/json"
    
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as response:
            return response.status, response.read().decode('utf-8')
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8')
    except urllib.error.URLError as e:
        return 0, str(e)

def run_integration_test():
    print("--- Starting Raki API Integration Test ---")
    
    # 1. Health Check
    print("1. Checking API Health...")
    status, body = request("GET", HEALTH_URL)
    if status != 200:
        print(f"FAILED: Health check returned {status}: {body}")
        sys.exit(1)
    print("   SUCCESS: API is healthy.")

    # 2. Create User
    test_user = "api_test_user"
    print(f"\n2. Creating user '{test_user}'...")
    status, body = request("POST", "/users", {"username": test_user, "password": "securepassword123"})
    if status != 200:
        print(f"FAILED: Could not create user: {status} {body}")
        sys.exit(1)
    print(f"   SUCCESS: User '{test_user}' created (Response: {body.strip()})")

    # 3. Verify User Status
    print(f"\n3. Verifying user '{test_user}' status...")
    status, body = request("GET", f"/users/{test_user}")
    if status != 200:
        # Note: Depending on raki-api implementation, this might return 200 with details or just OK
        print(f"FAILED: Could not fetch user status: {status} {body}")
        sys.exit(1)
    print(f"   SUCCESS: User status fetched: {body.strip()}")

    # 4. Add Network (Login from localhost simulation)
    print(f"\n4. Adding 'localhost' network for '{test_user}'...")
    # Matches the 'createNetwork' endpoint in raki-api
    network_req = {
        "user": test_user,
        "addr": "localhost:6667",
        "name": "local_net"
    }
    status, body = request("POST", "/networks", network_req)
    if status != 200:
        print(f"FAILED: Could not add network: {status} {body}")
        sys.exit(1)
    print(f"   SUCCESS: Network 'localhost' added for user (Response: {body.strip()})")

    # 5. Clean up
    print(f"\n5. Deleting test user '{test_user}'...")
    status, body = request("DELETE", f"/users/{test_user}")
    if status != 200:
        print(f"FAILED: Could not delete user: {status} {body}")
        sys.exit(1)
    print(f"   SUCCESS: User '{test_user}' deleted.")

    print("\n✅ Raki-API Integration Test Completed Successfully!")

if __name__ == "__main__":
    run_integration_test()

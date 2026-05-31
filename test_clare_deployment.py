#!/usr/bin/env python3
import urllib.request
import urllib.error
import json
import socket
import time
import sys

API_URL = "https://sober-clare.fly.dev:8081/api"
HEALTH_URL = "https://sober-clare.fly.dev:8081/health"
API_KEY = "test-api-key"
IRC_HOST = "sober-clare.fly.dev"
IRC_PORT = 6697

def request(method, url, data=None):
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

def test_raki_and_soju():
    print("1. Testing Raki API Health...")
    status, body = request("GET", HEALTH_URL)
    if status != 200:
        print(f"Health check failed: {status} {body}")
        sys.exit(1)
    print("   Health check passed!")

    print("\n2. Creating test user via Raki API...")
    status, body = request("POST", f"{API_URL}/users", {"username": "testuser", "password": "testpassword"})
    if status != 200:
        print(f"Failed to create user: {status} {body}")
        sys.exit(1)
    print("   User created successfully!")

    print("\n3. Testing Soju IRC connectivity...")
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(10)
        s.connect((IRC_HOST, IRC_PORT))
        
        # Authenticate via SASL PLAIN syntax or PASS
        # Soju accepts PASS <username>:<password>
        s.sendall(b"PASS testuser:testpassword\r\n")
        s.sendall(b"NICK testuser\r\n")
        s.sendall(b"USER testuser 0 * :Test User\r\n")
        
        connected = False
        start_time = time.time()
        while time.time() - start_time < 5:
            resp = s.recv(4096).decode('utf-8', errors='ignore')
            if not resp:
                break
            for line in resp.strip().split('\r\n'):
                print(f"   [IRC] {line}")
                if "001" in line or "MODE testuser" in line:
                    connected = True
            if connected:
                break
            
        s.close()
        
        if not connected:
            print("   Failed to receive IRC welcome message.")
            sys.exit(1)
        print("   IRC connectivity passed!")
        
    except Exception as e:
        print(f"   IRC test failed with exception: {e}")
        sys.exit(1)
        
    print("\n4. Cleaning up test user via Raki API...")
    status, body = request("DELETE", f"{API_URL}/users/testuser")
    if status != 200:
        print(f"Failed to delete user: {status} {body}")
        sys.exit(1)
    print("   User deleted successfully!")
    
    print("\n✅ All tests passed successfully!")

if __name__ == "__main__":
    test_raki_and_soju()

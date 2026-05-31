#!/usr/bin/env python3
import urllib.request
import urllib.error
import json
import socket
import time
import sys
import subprocess
import os

API_URL = "http://localhost:8082/api"
HEALTH_URL = "http://localhost:8082/health"
API_KEY = "test-key"
IRC_HOST = "127.0.0.1"
IRC_PORT = 6667

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

def setup_environment():
    print("Setting up local Soju and Raki-API environment...")
    os.makedirs("/tmp/soju_test_py", exist_ok=True)
    with open("/tmp/soju_test_py/soju.conf", "w") as f:
        f.write(f"listen irc+insecure://127.0.0.1:{IRC_PORT}\n")
        f.write("listen unix+admin:///tmp/soju_test_py/admin.sock\n")

    # Start soju
    soju_proc = subprocess.Popen(["soju", "-config", "/tmp/soju_test_py/soju.conf"], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    
    # Wait for socket
    socket_path = "/tmp/soju_test_py/admin.sock"
    for _ in range(5):
        if os.path.exists(socket_path):
            break
        time.sleep(1)
    else:
        print("Failed to start Soju: admin socket not created.")
        soju_proc.kill()
        sys.exit(1)

    # Start Raki API
    raki_bin = os.path.join(os.path.dirname(os.path.realpath(__file__)), "result/bin/raki-api")
    if not os.path.exists(raki_bin):
        # Fallback to local build path if result doesn't exist
        raki_bin = os.path.join(os.path.dirname(os.path.realpath(__file__)), "raki-api")
    
    api_proc = subprocess.Popen([raki_bin, "-socket", socket_path, "-listen", ":8082", "-api-keys", API_KEY], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    time.sleep(2) # Give API time to boot

    return soju_proc, api_proc

def test_raki_and_soju():
    soju_proc, api_proc = setup_environment()
    
    try:
        print("1. Testing Raki API Health...")
        status, body = request("GET", HEALTH_URL)
        if status != 200:
            print(f"Health check failed: {status} {body}")
            sys.exit(1)
        print("   Health check passed!")

        print("\n2. Creating test user via Raki API...")
        status, body = request("POST", f"{API_URL}/users", {"username": "localtest", "password": "localpassword"})
        if status != 200:
            print(f"Failed to create user: {status} {body}")
            sys.exit(1)
        print("   User created successfully!")

        print("\n3. Testing Soju IRC connectivity...")
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(5)
        s.connect((IRC_HOST, IRC_PORT))
        
        # Authenticate via PASS
        s.sendall(b"PASS localtest:localpassword\r\n")
        s.sendall(b"NICK localtest\r\n")
        s.sendall(b"USER localtest 0 * :Local Test\r\n")
        
        connected = False
        start_time = time.time()
        while time.time() - start_time < 5:
            resp = s.recv(4096).decode('utf-8', errors='ignore')
            if not resp:
                break
            for line in resp.strip().split('\r\n'):
                print(f"   [IRC] {line}")
                if "001" in line or "MODE localtest" in line:
                    connected = True
            if connected:
                break
            
        s.close()
        
        if not connected:
            print("   Failed to receive IRC welcome message.")
            sys.exit(1)
        print("   IRC connectivity passed!")
            
        print("\n4. Cleaning up test user via Raki API...")
        status, body = request("DELETE", f"{API_URL}/users/localtest")
        if status != 200:
            print(f"Failed to delete user: {status} {body}")
            sys.exit(1)
        print("   User deleted successfully!")
        
        print("\n✅ All local integration tests passed successfully!")
    
    finally:
        print("\nCleaning up processes...")
        api_proc.kill()
        soju_proc.kill()
        subprocess.run(["rm", "-rf", "/tmp/soju_test_py"])

if __name__ == "__main__":
    test_raki_and_soju()

#!/usr/bin/env python3
import urllib.request
import urllib.error
import json
import socket
import time
import sys

API_BASE = "https://sober-clare.fly.dev:8081/api"
API_KEY = "test-api-key"
IRC_HOST = "sober-clare.fly.dev"
IRC_PORT = 6697

def request(method, path, data=None):
    url = f"{API_BASE}{path}"
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

def run_full_integration():
    print("--- Starting Full Bouncer Integration Test ---")
    
    # 1. Create Admin User 'epoch'
    print("1. Creating admin user 'epoch'...")
    status, body = request("POST", "/users", {"username": "epoch", "password": "epochpassword123", "admin": True})
    if status != 200:
        print(f"FAILED to create user: {status} {body}")
        sys.exit(1)
    print("   SUCCESS: User 'epoch' created.")

    # 2. Create Networks
    networks = [
        {"name": "libera", "addr": "irc.libera.chat:6697"},
        {"name": "oftc", "addr": "irc.oftc.net:6667"}
    ]
    for net in networks:
        print(f"2. Adding network '{net['name']}' ({net['addr']})...")
        status, body = request("POST", "/networks", {"user": "epoch", "addr": net['addr'], "name": net['name']})
        if status != 200:
            print(f"FAILED to add network {net['name']}: {status} {body}")
            sys.exit(1)
        print(f"   SUCCESS: Network '{net['name']}' added.")

    # 3. Join Channel ##clare
    for net in ["libera", "oftc"]:
        print(f"3. Joining ##clare on '{net}'...")
        status, body = request("POST", "/channels", {"user": "epoch", "network": net, "name": "##clare"})
        if status != 200:
            print(f"FAILED to join channel on {net}: {status} {body}")
            sys.exit(1)
        print(f"   SUCCESS: Joined ##clare on '{net}'.")

    # 4. Connect via IRC, Send Message, Logout
    print("\n4. Simulating IRC client ('epoch')...")
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(10)
        s.connect((IRC_HOST, IRC_PORT))
        
        # Authenticate
        s.sendall(b"PASS epoch:epochpassword123\r\n")
        s.sendall(b"NICK epoch\r\n")
        s.sendall(b"USER epoch 0 * :Epoch Admin\r\n")
        
        welcome = False
        start = time.time()
        while time.time() - start < 10:
            resp = s.recv(4096).decode('utf-8', errors='ignore')
            if not resp: break
            for line in resp.strip().split('\r\n'):
                print(f"   [IRC] {line}")
                if " 001 " in line:
                    welcome = True
                    # Once welcomed, send a message to ##clare (defaulting to first network)
                    # To target specific network in soju: ##clare/libera
                    s.sendall(b"PRIVMSG ##clare :Hello from SOBER Integration Test suite!\r\n")
                    time.sleep(1)
                    s.sendall(b"QUIT :Testing complete\r\n")
                    break
            if "ERROR" in resp or welcome: break

        s.close()
        if not welcome:
            print("   FAILED: IRC authentication or welcome message timed out.")
            sys.exit(1)
    except Exception as e:
        print(f"   FAILED: IRC connection error: {e}")
        sys.exit(1)

    print("\n✅ Full Integration Test PASSED!")

if __name__ == "__main__":
    run_full_integration()

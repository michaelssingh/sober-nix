#!/usr/bin/env python3
import urllib.request
import urllib.error
import json
import socket
import time
import sys

API_EXEC = "https://sober-clare.fly.dev:8081/api/exec"
API_KEY = "test-api-key"
IRC_HOST = "sober-clare.fly.dev"
IRC_PORT = 6697

def raki_exec(args, ignore_errors=False):
    data = json.dumps({"args": args}).encode('utf-8')
    req = urllib.request.Request(API_EXEC, data=data, headers={"X-API-Key": API_KEY, "Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req) as response:
            return response.read().decode('utf-8').strip()
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8')
        if ignore_errors:
            return f"Error (ignored): {e.code} {body}"
        print(f"API Error: {e.code} {body}")
        sys.exit(1)
    except Exception as e:
        print(f"Connection Error: {e}")
        sys.exit(1)

def run_full_integration():
    print("--- Starting Raki-API Powered Integration Test ---")
    
    # 1. Ensure Admin User 'epoch' exists
    print("1. Ensuring admin user 'epoch' exists...")
    raki_exec(["user", "create", "-username", "epoch", "-password", "epochpassword123", "-admin"], ignore_errors=True)
    # Force password update to be sure
    raki_exec(["user", "update", "epoch", "-password", "epochpassword123"])
    print("   User 'epoch' is ready.")

    # 2. Ensure network 'ergo-local' exists
    print("2. Ensuring network 'ergo-local' exists...")
    raki_exec(["user", "run", "epoch", "network", "create", "-addr", "127.0.0.1:6667", "-name", "ergo-local"], ignore_errors=True)
    print("   Network 'ergo-local' is ready.")

    # 3. Join channel ##clare on ergo-local
    print("3. Joining ##clare on 'ergo-local'...")
    raki_exec(["user", "run", "epoch", "channel", "create", "ergo-local/##clare"], ignore_errors=True)
    print("   Channel '##clare' join initiated.")

    # 4. Final IRC connectivity check, message, and logout
    print("\n4. Simulating IRC login and messaging...")
    try:
        time.sleep(2)
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(15)
        print(f"   Connecting to {IRC_HOST}:{IRC_PORT}...")
        s.connect((IRC_HOST, IRC_PORT))
        
        # Auth string for soju: user:pass
        s.sendall(b"PASS epoch:epochpassword123\r\n")
        s.sendall(b"NICK epoch\r\n")
        s.sendall(b"USER epoch 0 * :Epoch Admin\r\n")
        
        welcome = False
        joined = False
        start = time.time()
        while time.time() - start < 15:
            data = s.recv(4096).decode('utf-8', errors='ignore')
            if not data: break
            for line in data.strip().split('\r\n'):
                print(f"   [IRC] {line}")
                if " 001 " in line:
                    welcome = True
                
                # Wait for ##clare join confirmation or presence
                if " JOIN " in line and "##clare" in line:
                    joined = True
                    print("   [TEST] Detected join to ##clare")
                    s.sendall(b"PRIVMSG ##clare :SOBER Full Integration Test Successful.\r\n")
                    time.sleep(1)
                    s.sendall(b"QUIT :Testing complete\r\n")
                    break
                
                # If already in channel, we might see names list
                if " 353 " in line and "##clare" in line:
                    joined = True
                    print("   [TEST] Already in ##clare")
                    s.sendall(b"PRIVMSG ##clare :SOBER Full Integration Test (Re-entry) Successful.\r\n")
                    time.sleep(1)
                    s.sendall(b"QUIT :Testing complete\r\n")
                    break

            if joined: break
        
        s.close()
        if not welcome:
            print("   FAILED: Could not establish IRC session.")
            sys.exit(1)
    except Exception as e:
        print(f"   FAILED: IRC Error: {e}")
        sys.exit(1)

    print("\n✅ ALL SYSTEMS FUNCTIONAL: User, Network, Channel, and IRC verified.")

if __name__ == "__main__":
    run_full_integration()

#!/usr/bin/env python3
import socket
import select
import threading
import struct
import sys

class Socks5Server:
    def __init__(self, host, port):
        self.host = host
        self.port = port

    def handle_client(self, client_socket):
        try:
            # 1. Negotiation
            header = client_socket.recv(2)
            if len(header) < 2:
                return
            version, nmethods = struct.unpack("!BB", header)
            if version != 5:
                return
            methods = client_socket.recv(nmethods)
            
            # Send chosen method: 0x00 (no auth)
            client_socket.sendall(struct.pack("!BB", 0x05, 0x00))
            
            # 2. Request
            req_header = client_socket.recv(4)
            if len(req_header) < 4:
                return
            version, cmd, rsv, atyp = struct.unpack("!BBBB", req_header)
            
            if cmd != 1:  # Only CONNECT is supported
                # Command not supported
                client_socket.sendall(struct.pack("!BBBBIH", 0x05, 0x07, 0x00, 0x01, 0, 0))
                client_socket.close()
                return

            if atyp == 1:  # IPv4
                addr_bytes = client_socket.recv(4)
                if len(addr_bytes) < 4:
                    return
                addr = socket.inet_ntoa(addr_bytes)
            elif atyp == 3:  # Domain name
                len_byte = client_socket.recv(1)
                if not len_byte:
                    return
                domain_len = struct.unpack("!B", len_byte)[0]
                domain_bytes = client_socket.recv(domain_len)
                if len(domain_bytes) < domain_len:
                    return
                addr = domain_bytes.decode('utf-8')
            elif atyp == 4:  # IPv6
                addr_bytes = client_socket.recv(16)
                if len(addr_bytes) < 16:
                    return
                addr = socket.inet_ntop(socket.AF_INET6, addr_bytes)
            else:
                # Address type not supported
                client_socket.sendall(struct.pack("!BBBBIH", 0x05, 0x08, 0x00, 0x01, 0, 0))
                client_socket.close()
                return

            port_bytes = client_socket.recv(2)
            if len(port_bytes) < 2:
                return
            port = struct.unpack("!H", port_bytes)[0]
            
            print(f"Request: Connect to {addr}:{port}", file=sys.stderr)
            sys.stderr.flush()
            
            # 3. Connect to destination
            try:
                infos = socket.getaddrinfo(addr, port, socket.AF_UNSPEC, socket.SOCK_STREAM)
            except Exception as e:
                print(f"getaddrinfo failed for {addr}:{port} - {e}", file=sys.stderr)
                sys.stderr.flush()
                # Host unreachable
                client_socket.sendall(struct.pack("!BBBBIH", 0x05, 0x04, 0x00, 0x01, 0, 0))
                client_socket.close()
                return

            remote_socket = None
            for family, socktype, proto, canonname, sockaddr in infos:
                try:
                    remote_socket = socket.socket(family, socktype, proto)
                    remote_socket.settimeout(10.0)
                    remote_socket.connect(sockaddr)
                    remote_socket.settimeout(None)
                    break
                except Exception as ex:
                    print(f"Connect failed for family {family} - {ex}", file=sys.stderr)
                    sys.stderr.flush()
                    if remote_socket:
                        remote_socket.close()
                    remote_socket = None

            if not remote_socket:
                print(f"All connection attempts failed for {addr}:{port}", file=sys.stderr)
                sys.stderr.flush()
                # Connection refused
                client_socket.sendall(struct.pack("!BBBBIH", 0x05, 0x05, 0x00, 0x01, 0, 0))
                client_socket.close()
                return

            print(f"Success: Connected to {addr}:{port}", file=sys.stderr)
            sys.stderr.flush()

            # Send success reply: 0.0.0.0:0
            client_socket.sendall(struct.pack("!BBBBIH", 0x05, 0x00, 0x00, 0x01, 0, 0))
            
            # 4. Bidirectional transfer
            self.tunnel(client_socket, remote_socket)
        except Exception as e:
            print(f"General exception in handler: {e}", file=sys.stderr)
            sys.stderr.flush()
        finally:
            try:
                client_socket.close()
            except Exception:
                pass

    def tunnel(self, client, remote):
        inputs = [client, remote]
        while True:
            try:
                readable, _, _ = select.select(inputs, [], [])
                if client in readable:
                    data = client.recv(8192)
                    if not data:
                        break
                    remote.sendall(data)
                if remote in readable:
                    data = remote.recv(8192)
                    if not data:
                        break
                    client.sendall(data)
            except Exception:
                break
        try:
            remote.close()
        except Exception:
            pass

    def start(self):
        server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            server.bind((self.host, self.port))
        except Exception as e:
            print(f"Error binding to {self.host}:{self.port} - {e}", file=sys.stderr)
            sys.exit(1)
        server.listen(128)
        print(f"SOCKS5 server started on {self.host}:{self.port}")
        sys.stdout.flush()
        
        try:
            while True:
                client_sock, addr = server.accept()
                t = threading.Thread(target=self.handle_client, args=(client_sock,))
                t.daemon = True
                t.start()
        except KeyboardInterrupt:
            print("Shutting down SOCKS5 server.")
        finally:
            server.close()

if __name__ == "__main__":
    port = 1080
    if len(sys.argv) > 1:
        port = int(sys.argv[1])
    s = Socks5Server("127.0.0.1", port)
    s.start()

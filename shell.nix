{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.soju
    pkgs.curl
    pkgs.procps
  ];

  shellHook = ''
    echo "Starting Soju bouncer..."
    mkdir -p /tmp/soju_test
    # Create a simple config
    echo "listen unix+admin:///tmp/soju_test/admin.sock" > /tmp/soju_test/soju.conf
    
    # Start Soju (this might need adjustments depending on how it's installed)
    soju -config /tmp/soju_test/soju.conf &
    SOJU_PID=$!
    
    echo "Soju started with PID $SOJU_PID. Admin socket at /tmp/soju_test/admin.sock"
    
    # Alias to cleanup
    cleanup() {
      kill $SOJU_PID
      rm -rf /tmp/soju_test
      echo "Soju stopped."
    }
    trap cleanup EXIT
  '';
}

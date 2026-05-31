{ config, lib, pkgs, ... }:

let
  injectKeysScript = pkgs.writeScript "inject-rbw-ssh-keys.py" ''
    #!${pkgs.python3}/bin/python3
    import subprocess
    import os
    import time
    import concurrent.futures

    # Wait for the agent socket to be available
    sock_path = os.environ.get("SSH_AUTH_SOCK")
    while not sock_path or not os.path.exists(sock_path):
        time.sleep(0.1)

    # Fetch all items
    rbw_bin = "${pkgs.rbw}/bin/rbw"
    ssh_add_bin = "${pkgs.openssh}/bin/ssh-add"

    try:
        list_output = subprocess.check_output([rbw_bin, "list"], text=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to list rbw items: {e}")
        exit(1)

    # Filter for those ending in .key
    keys = [line.strip() for line in list_output.splitlines() if line.strip().endswith(".key")]

    if not keys:
        print("No .key items found.")
        exit(0)

    def inject_key(key_name):
        try:
            # Get the private key
            get_proc = subprocess.Popen(
                [rbw_bin, "get", "--field", "private-key", key_name],
                stdout=subprocess.PIPE,
                text=True
            )
            
            # Pipe it to ssh-add
            add_proc = subprocess.Popen(
                [ssh_add_bin, "-"],
                stdin=get_proc.stdout,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            get_proc.stdout.close() # Allow get_proc to receive a SIGPIPE if add_proc exits.
            stdout, stderr = add_proc.communicate()
            
            if add_proc.returncode != 0:
                print(f"Failed to add key '{key_name}': {stderr.strip()}")
            else:
                print(f"Successfully added key '{key_name}'")
        except Exception as e:
            print(f"Exception while injecting key '{key_name}': {e}")

    # Iterate over the keys and inject them in parallel
    with concurrent.futures.ThreadPoolExecutor() as executor:
        executor.map(inject_key, keys)
  '';
in
{
  # Ensure sops-nix is available
  sops.secrets.bw_master_password = {
    path = "%h/.config/rbw/master_password";
  };

  systemd.user.services.rbw-unlock = {
    Unit = {
      Description = "Unlock rbw on login";
      After = [ "graphical-session.target" ];
    };

    Service = {
      Type = "oneshot";
      ExecStart = "${pkgs.bash}/bin/bash -c '${pkgs.rbw}/bin/rbw unlock < ${config.home.homeDirectory}/.config/rbw/master_password'";
      Restart = "on-failure";
    };

    Install = {
      WantedBy = [ "graphical-session.target" ];
    };
  };

  systemd.user.services.ssh-keys-inject = {
    Unit = {
      Description = "Inject SSH keys from rbw into ssh-agent";
      Requires = [ "rbw-unlock.service" "ssh-agent.service" ];
      After = [ "rbw-unlock.service" "ssh-agent.service" ];
    };

    Service = {
      Type = "oneshot";
      Environment = "SSH_AUTH_SOCK=%t/ssh-agent";
      ExecStart = "${injectKeysScript}";
      Restart = "on-failure";
    };

    Install = {
      WantedBy = [ "graphical-session.target" ];
    };
  };
}

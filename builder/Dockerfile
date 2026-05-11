FROM nixos/nix:latest

# 1. Install OpenSSH and the MOUNT command (util-linux)
RUN nix-env -iA \
    nixpkgs.openssh \
    nixpkgs.bash \
    nixpkgs.coreutils \
    nixpkgs.util-linux

# 2. Create the missing 'sshd' user for privilege separation
RUN echo "sshd:x:74:74:Privilege-separated SSH:/var/empty:/sbin/nologin" >> /etc/passwd && \
    echo "sshd:x:74:" >> /etc/group && \
    mkdir -p /var/empty && \
    chmod 555 /var/empty

# 3. Setup SSH directories and keys
RUN mkdir -p /var/run/sshd /root/.ssh /etc/ssh && \
    chmod 700 /root/.ssh && \
    /root/.nix-profile/bin/ssh-keygen -A

# 4. Inject your Public Key
COPY id_ed25519.pub /root/.ssh/authorized_keys
RUN chmod 600 /root/.ssh/authorized_keys

# 5. SSH Configuration
RUN echo "PermitRootLogin yes" > /etc/ssh/sshd_config && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config

# 6. Nix Builder Configuration
RUN echo "trusted-users = root" >> /etc/nix/nix.conf && \
    echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf

COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 22
ENTRYPOINT ["/bin/sh", "/start.sh"]

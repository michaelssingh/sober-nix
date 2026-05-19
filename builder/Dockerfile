FROM fedora:latest

# 1. Update base image and install necessary packages
RUN dnf upgrade --refresh -y && dnf install -y openssh-server xz curl procps-ng sudo && dnf clean all

# 2. Install Nix using the Determinate Systems installer
# We use the multi-user installer and tell it not to use init systems since this is a container.
RUN curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install linux \
    --init none \
    --no-confirm

# 3. Setup SSH Configuration
# We use port 2222 internally to avoid collision with Fly's proxy on port 22
RUN sed -i 's/^#Port 22/Port 2222/' /etc/ssh/sshd_config && \
    sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config && \
    sed -i 's/^#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    mkdir -p /var/run/sshd && \
    chmod 0755 /var/run/sshd

# 4. Install Soju IRC Bouncer
RUN . /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh && \
    nix profile install nixpkgs#soju

# 5. Add Entrypoint Script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 2222
ENTRYPOINT ["/entrypoint.sh"]

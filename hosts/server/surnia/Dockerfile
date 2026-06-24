FROM ubuntu:24.04

# Install basic packages
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    openssh-server \
    sudo \
    curl \
    git \
    rsync \
    tmux \
    fish \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Setup sshd
RUN mkdir -p /var/run/sshd

# Create user 'init'
RUN useradd -m -s /usr/bin/fish init && \
    echo "init ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Copy SSH configuration
RUN echo "Port 2222" > /etc/ssh/sshd_config && \
    echo "PermitRootLogin no" >> /etc/ssh/sshd_config && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "UsePAM yes" >> /etc/ssh/sshd_config && \
    echo "Subsystem sftp internal-sftp" >> /etc/ssh/sshd_config

# Setup authorized keys for init
RUN mkdir -p /home/init/.ssh && chmod 700 /home/init/.ssh
RUN echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHy4pVdTzFoOPdQkXdwG9cJuSeHCXm3UTaDgRl/UXGWI" >> /home/init/.ssh/authorized_keys && \
    echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAPGyqlLfLc3PTAQ00M2fg4kaEnoOkmMfECNGOQo/2FI" >> /home/init/.ssh/authorized_keys && \
    echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEk7iS+9SpdVSG/RVdVjPP13RDyd/xLBNMvcVbpgJrAX" >> /home/init/.ssh/authorized_keys && \
    echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIF1XgGtZv4XLasXUbDtRmFn7FvaAojdPxSTpQg/A2gnF sprite@sprite" >> /home/init/.ssh/authorized_keys && \
    chmod 600 /home/init/.ssh/authorized_keys && \
    chown -R init:init /home/init/.ssh

EXPOSE 2222

CMD ["/usr/sbin/sshd", "-D", "-p", "2222"]

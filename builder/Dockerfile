FROM nixos/nix:latest
RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf
RUN mkdir -p /root/.ssh
# You must place your id_ed25519.pub here
COPY authorized_keys /root/.ssh/authorized_keys
RUN apk add openssh && ssh-keygen -A
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]

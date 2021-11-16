FROM lscr.io/linuxserver/openssh-server

# Enable TCP forwarding (critical to normal tunnels)
RUN sed -i 's/AllowTcpForwarding no/AllowTcpForwarding yes/g' /etc/ssh/sshd_config
FROM centos:6
MAINTAINER "Robin Verlangen"

# Install golang
RUN yum install -y epel-release
RUN yum install -y golang

# Git for downloading source
RUN yum install -y git
RUN git clone https://github.com/RobinUS2/indispenso.git
WORKDIR /indispenso

# Start tests
WORKDIR /indispenso/tests
CMD ["./start_nodes.sh"] 

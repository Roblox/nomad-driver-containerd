# Docker image for nomad-driver-containerd
# This image will contain nomad (server + client), golang and containerd.
FROM ubuntu:18.04

ENV NOMAD_VERSION 0.11.2
ENV GO_VERSION 1.14.3
ENV CONTAINERD_VERSION 1.3.3

RUN apt-get update \
  && apt-get install -y apt-utils curl unzip

# Install golang
RUN set -x \
  && cd /tmp \
  && curl -L -o go${GO_VERSION}.linux-amd64.tar.gz https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
  && chmod +x /usr/local/go \
  && echo "export PATH=$PATH:/usr/local/go/bin" >> $HOME/.bashrc \
  && rm -f go${GO_VERSION}.linux-amd64.tar.gz

# Install containerd
RUN set -x \
  && cd /tmp \
  && curl -L -o containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz \
  && rm -f containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz

# Install nomad
RUN set -x \
  && cd /tmp \
  && curl -L -o nomad_${NOMAD_VERSION}_linux_amd64.zip https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_linux_amd64.zip \
  && unzip -d /usr/local/bin nomad_${NOMAD_VERSION}_linux_amd64.zip \
  && chmod +x /usr/local/bin/nomad \
  && rm -f nomad_${NOMAD_VERSION}_linux_amd64.zip

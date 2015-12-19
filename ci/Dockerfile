FROM ubuntu:vivid
MAINTAINER https://github.com/cloudfoundry/dea_ng

# install build dependencies
RUN \
  apt-get update && \
  apt-get -y dist-upgrade && \
  apt-get -y install \
    module-init-tools \
    aufs-tools \
    build-essential \
    curl \
    git \
    graphviz \
    htop \
    libpython-dev \
    lsof \
    psmisc \
    python \
    strace \
    jq \
    wget \
    iptables \
    quota \
    ulogd \
    zip \
    pkg-config \
    autoconf \
    libssl-dev libreadline-dev zlib1g-dev \
    sudo \
    vim \
    && \
  apt-get -y autoremove && \
  apt-get clean


RUN \
  locale-gen en_US.UTF-8 && \
  dpkg-reconfigure locales

ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

RUN \
  sed -i -e '/Defaults\s\+env_reset/a Defaults\texempt_group=sudo' /etc/sudoers && \
  sed -i -e 's/%sudo\s*ALL=(ALL:ALL) ALL/%sudo\tALL=NOPASSWD:ALL/g' /etc/sudoers

RUN \
  useradd -m -p "*" cf -G sudo

# install Go 1.4
RUN \
  wget -qO- https://storage.googleapis.com/golang/go1.4.3.linux-amd64.tar.gz | tar -C /usr/local -xzf -

# setup git
RUN \
  git config --system user.email "nobody@example.com" && \
  git config --system user.name "Anonymous Coward"

# install Ruby 2.1.7
RUN \
  git clone https://github.com/rbenv/ruby-build.git /tmp/ruby-build && \
  cd /tmp/ruby-build && \
  PREFIX=/usr/local ./install.sh && \
  /usr/local/bin/ruby-build 2.1.7 /usr && \
  cd && \
  rm -rf /tmp/ruby-build && \
  gem install bundler --no-ri --no-doc

USER cf

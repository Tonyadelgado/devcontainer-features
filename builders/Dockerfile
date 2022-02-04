FROM ubuntu:bionic as base

# Set required environment variables, label
ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="com.chuxel.stacks.test.bionic"
LABEL io.buildpacks.stack.id="com.chuxel.stacks.test.bionic"

# Add required non-root user
RUN groupadd cnb --gid ${CNB_GROUP_ID} && \
  useradd --uid ${CNB_USER_ID} --gid ${CNB_GROUP_ID} -m -s /bin/bash cnb

# https://github.com/docker-library/buildpack-deps/blob/master/ubuntu/bionic/curl/Dockerfile
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
		curl \
		netbase \
		wget \
		tzdata \
        gnupg2 \
        dirmngr \
        procps \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*

#  ******* Build image  *******
FROM base as build

# https://github.com/docker-library/buildpack-deps/blob/master/ubuntu/bionic/Dockerfile
# https://github.com/docker-library/buildpack-deps/blob/master/ubuntu/bionic/scm/Dockerfile
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        git \
        openssh-client \
        autoconf \
        automake \
        bzip2 \
        dpkg-dev \
        file \
        g++ \
        gcc \
        git \
        imagemagick \
        libbz2-dev \
        libc6-dev \
        libcurl4-openssl-dev \
        libdb-dev \
        libevent-dev \
        libffi-dev \
        libgdbm-dev \
        libglib2.0-dev \
        libgmp-dev \
        libjpeg-dev \
        libkrb5-dev \
        liblzma-dev \
        libmagickcore-dev \
        libmagickwand-dev \
        libmaxminddb-dev \
        libncurses5-dev \
        libncursesw5-dev \
        libpng-dev \
        libpq-dev \
        libreadline-dev \
        libsqlite3-dev \
        libssl-dev \
        libtool \
        libwebp-dev \
        libxml2-dev \
        libxslt-dev \
        libyaml-dev \
        make \
        patch \
        unzip \
        xz-utils \
        zlib1g-dev \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*

#  ******* Dev container "build" image  *******
FROM build as devcontainer-build

#  ******* Dev container "run" image  *******
FROM devcontainer-build as devcontainer-run

RUN apt-get update \
    && yes | unminimize 2>&1 \ 
    && bash -c "$(curl -fsSL "https://raw.githubusercontent.com/microsoft/vscode-dev-containers/main/script-library/common-debian.sh")" -- true cnb ${CNB_USER_ID} ${CNB_GROUP_ID} false true true \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*

USER ${CNB_USER_ID}:${CNB_GROUP_ID}

#  ******* Production run image  *******
FROM base as run
USER ${CNB_USER_ID}:${CNB_GROUP_ID}

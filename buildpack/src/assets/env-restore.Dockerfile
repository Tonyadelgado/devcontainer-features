ARG IMAGE_NAME
FROM ${IMAGE_NAME}

USER root
RUN --mount=type=bind,source=.,target=/host \
    mkdir -p /usr/local/etc/vscode-dev-containers/ \
    && cp -f /host/buildpack.env /usr/local/etc/vscode-dev-containers/ \
    && echo "BP_LAUNCHER_ENV_RESTORED=true" >> /usr/local/etc/vscode-dev-containers/buildpack.env \
    && cat /usr/local/etc/vscode-dev-containers/buildpack.env >> /etc/environment \
    # Handle any casees where /etc/environment is not used
    && echo 'if [ "${BP_LAUNCHER_ENV_RESTORED}"!=true ]; then set -a; . /usr/local/etc/vscode-dev-containers/buildpack.env; set +a; fi' \
        | tee -a /etc/bash.bashrc /etc/zsh/zshenv >> /etc/profile.d/00-restore-launcher-env.sh \
    && chmod +x /etc/profile.d/00-restore-launcher-env.sh
USER cnb
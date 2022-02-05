ARG IMAGE_NAME
FROM ${IMAGE_NAME}

USER root
RUN --mount=type=bind,source=.,target=/host \
    mkdir -p /usr/local/etc/vscode-dev-containers/ \
    # The CNB_LAYERS_DIR is filtered out by the launcher, so this lets us know what we should do in that case.
    # For the dev container case, we'll also check DCNB_ENV_LOADED since we can't export an env var unset.
    # https://github.com/buildpacks/spec/blob/main/platform.md#launch-environment
    && echo 'if [ ! -z "${CNB_LAYERS_DIR}" ] && [ -z "${DCNB_ENV_LOADED}" ]; then\nenv\nset -a\nDCNB_ENV_LOADED=true\n' > /usr/local/etc/vscode-dev-containers/buildpack-launch.sh \
    && cat /host/buildpack.env >> /usr/local/etc/vscode-dev-containers/buildpack-launch.sh \
    && echo 'set +a\nfi\nunset CNB_APP_DIR CNB_LAYERS_DIR CNB_PROCESS_TYPE CNB_DEPRECATION_MODE CNB_PLATFORM_API' >> /usr/local/etc/vscode-dev-containers/buildpack-launch.sh \
    && chmod +x /usr/local/etc/vscode-dev-containers/buildpack-launch.sh \
    # Fire this from bashrc, zshenv and profile for full coverage.
    && echo '. /usr/local/etc/vscode-dev-containers/buildpack-launch.sh' | tee -a /etc/bash.bashrc >> /etc/zsh/zshenv \
    && ln -s /usr/local/etc/vscode-dev-containers/buildpack-launch.sh /etc/profile.d/99-buildpack-launch.sh
USER cnb
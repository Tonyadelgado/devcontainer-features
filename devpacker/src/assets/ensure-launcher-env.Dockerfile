ARG IMAGE_NAME
FROM ${IMAGE_NAME}

USER root
RUN --mount=type=bind,source=.,target=/host bash /host/ensure-launcher-env.sh
USER cnb
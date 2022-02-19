ARG IMAGE_NAME
FROM ${IMAGE_NAME}

ARG POST_PROCESSING_DONE
ARG POST_PROCESSING_REQUIRED
USER root
RUN --mount=type=bind,source=.,target=/host bash /host/post-processing.sh ${POST_PROCESSING_REQUIRED}
USER cnb
LABEL com.microsoft.devcontainer.features.done="${POST_PROCESSING_DONE}"
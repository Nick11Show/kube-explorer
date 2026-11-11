# The binary is a fully static ELF (CGO_ENABLED=0, -extldflags -static), so
# scratch is sufficient and keeps the image tiny + network-independent to build.
# Build context must contain the staged binaries:
#   kube-explorer-linux-amd64
#   kube-explorer-linux-arm64
# TARGETOS/TARGETARCH are set per platform by the container builder, so both
# per-arch builds (composed into a multi-arch manifest) use this same file.
# Images are built and pushed to the private registry locally, not in CI.
FROM scratch
ARG TARGETARCH
ARG TARGETOS
COPY kube-explorer-${TARGETOS}-${TARGETARCH} /usr/bin/kube-explorer
ENTRYPOINT ["/usr/bin/kube-explorer"]

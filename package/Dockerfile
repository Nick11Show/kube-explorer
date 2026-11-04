# The binary is a fully static ELF (CGO_ENABLED=0, -extldflags -static), so
# scratch is sufficient and keeps the image tiny + network-independent to build.
# Build context must contain the staged binaries:
#   kube-explorer-linux-amd64
#   kube-explorer-linux-arm64
# buildx sets TARGETOS/TARGETARCH per platform, so this serves both the
# single-arch local builds and the multi-arch CI build.
FROM scratch
ARG TARGETARCH
ARG TARGETOS
COPY kube-explorer-${TARGETOS}-${TARGETARCH} /usr/bin/kube-explorer
ENTRYPOINT ["/usr/bin/kube-explorer"]

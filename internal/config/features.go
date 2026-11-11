package config

import "os"

// WatchListClientFeatureEnv is the client-go feature-gate environment variable
// that controls streaming LIST/WATCH ("watch list").
const WatchListClientFeatureEnv = "KUBE_FEATURE_WatchListClient"

// EnsureKubeFeatureDefaults applies client-go compatibility defaults for this
// build line. It must be called before any Kubernetes client/informer is
// constructed, because client-go reads these gates once, on first use.
//
// This build line uses client-go v0.36 (Kubernetes 1.36 libraries), where
// WatchListClient is enabled by default. It relies on the apiserver sending an
// "initial events end" bookmark for streaming LIST. Against Kubernetes 1.34
// apiservers (verified: RKE2 v1.34.7+rke2r1) that bookmark is not delivered,
// informers never report HasSynced, and steve LIST requests such as
// /v1/namespaces and /v1/pods hang indefinitely (single-object GET and
// ?limit=... still work because they bypass the cache).
//
// Disabling the gate restores the classic chunked LIST and works across the
// whole 1.34-1.36 support range. Operators targeting 1.35+ clusters who want
// the optimisation can opt back in by setting the environment variable
// explicitly, e.g. KUBE_FEATURE_WatchListClient=true.
func EnsureKubeFeatureDefaults() {
	if _, set := os.LookupEnv(WatchListClientFeatureEnv); !set {
		_ = os.Setenv(WatchListClientFeatureEnv, "false")
	}
}

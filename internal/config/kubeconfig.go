package config

import (
	"os"
)

// Token, when set, is used as the bearer token for authenticating to the
// kube-apiserver. It is combined with --kubeconfig (which supplies the
// API server URL and CA) so that a static/service-account token can be used
// instead of the kubeconfig's own user credentials.
var Token string

// RancherKubeConfigPaths are the well-known kubeconfig locations written by
// RKE2 and K3s on cluster nodes. They are consulted (in order) when neither
// --kubeconfig nor the KUBECONFIG environment variable is set. It is a package
// variable so it can be overridden in tests without touching the real paths.
var RancherKubeConfigPaths = []string{
	"/etc/rancher/rke2/rke2.yaml",
	"/etc/rancher/k3s/k3s.yaml",
}

// ResolveKubeConfig returns the kubeconfig path to use, applying this
// precedence:
//  1. explicit value (--kubeconfig flag / KUBECONFIG env, already captured by
//     steve's CLI into keconfig.Steve.KubeConfig)
//  2. KUBECONFIG environment variable
//  3. RKE2/K3s well-known paths (RancherKubeConfigPaths)
//  4. empty string — client-go then falls back to its default loading rules
//     (including in-cluster service-account config when run as a Pod)
func ResolveKubeConfig(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	for _, p := range RancherKubeConfigPaths {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

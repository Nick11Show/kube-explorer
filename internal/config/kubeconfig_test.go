package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKubeConfig(t *testing.T) {
	dir := t.TempDir()
	rke2 := filepath.Join(dir, "rke2.yaml")
	k3s := filepath.Join(dir, "k3s.yaml")
	if err := os.WriteFile(rke2, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(k3s, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override well-known Rancher paths so the test does not depend on the
	// real /etc/rancher/* files on the host.
	origPaths := RancherKubeConfigPaths
	t.Cleanup(func() { RancherKubeConfigPaths = origPaths })

	cases := []struct {
		name     string
		explicit string
		env      string
		paths    []string
		want     string
	}{
		{
			name:     "explicit flag wins",
			explicit: "/from/flag.yaml",
			env:      "",
			paths:    []string{rke2, k3s},
			want:     "/from/flag.yaml",
		},
		{
			name:     "KUBECONFIG env wins over rancher paths",
			explicit: "",
			env:      rke2,
			paths:    []string{k3s},
			want:     rke2,
		},
		{
			name:     "rancher rke2 path when nothing else set",
			explicit: "",
			env:      "",
			paths:    []string{rke2, k3s},
			want:     rke2,
		},
		{
			name:     "rancher k3s path when rke2 absent",
			explicit: "",
			env:      "",
			paths:    []string{"/no/such/rke2.yaml", k3s},
			want:     k3s,
		},
		{
			name:     "empty when nothing set and no rancher path",
			explicit: "",
			env:      "",
			paths:    []string{"/no/such/rke2.yaml", "/no/such/k3s.yaml"},
			want:     "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			RancherKubeConfigPaths = c.paths
			if c.env != "" {
				t.Setenv("KUBECONFIG", c.env)
			} else {
				t.Setenv("KUBECONFIG", "")
			}
			got := ResolveKubeConfig(c.explicit)
			if got != c.want {
				t.Fatalf("ResolveKubeConfig(%q) = %q, want %q", c.explicit, got, c.want)
			}
		})
	}
}

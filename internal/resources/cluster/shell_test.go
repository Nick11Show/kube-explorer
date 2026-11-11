package cluster

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestCreatePodTolerations_RKE2K3s(t *testing.T) {
	s := &shell{}
	pod := s.createPod()

	got := map[string]v1.Toleration{}
	for _, tol := range pod.Spec.Tolerations {
		got[tol.Key] = tol
	}

	// RKE2 / K3s use the hyphenated control-plane taint key. This is the bug fix.
	cp, ok := got["node-role.kubernetes.io/control-plane"]
	if !ok {
		t.Fatalf("missing toleration for node-role.kubernetes.io/control-plane (RKE2/K3s)")
	}
	if cp.Effect != v1.TaintEffectNoSchedule {
		t.Errorf("control-plane toleration effect = %v, want NoSchedule", cp.Effect)
	}

	// RKE1 legacy key kept for backward compatibility.
	if _, ok := got["node-role.kubernetes.io/controlplane"]; !ok {
		t.Errorf("missing toleration for legacy node-role.kubernetes.io/controlplane (RKE1)")
	}

	// etcd (RKE2 external etcd) toleration retained.
	if _, ok := got["node-role.kubernetes.io/etcd"]; !ok {
		t.Errorf("missing toleration for node-role.kubernetes.io/etcd")
	}

	// Must still select linux nodes.
	if sel, ok := pod.Spec.NodeSelector["kubernetes.io/os"]; !ok || sel != "linux" {
		t.Errorf("NodeSelector kubernetes.io/os = %q (present=%v), want linux", sel, ok)
	}
}

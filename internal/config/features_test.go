package config

import (
	"os"
	"testing"
)

func withFeatureEnv(t *testing.T, value string, set bool) {
	t.Helper()
	old, existed := os.LookupEnv(WatchListClientFeatureEnv)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(WatchListClientFeatureEnv, old)
		} else {
			_ = os.Unsetenv(WatchListClientFeatureEnv)
		}
	})
	if set {
		if err := os.Setenv(WatchListClientFeatureEnv, value); err != nil {
			t.Fatal(err)
		}
	} else {
		_ = os.Unsetenv(WatchListClientFeatureEnv)
	}
}

func TestEnsureKubeFeatureDefaultsDisablesWhenUnset(t *testing.T) {
	withFeatureEnv(t, "", false)
	EnsureKubeFeatureDefaults()
	if got := os.Getenv(WatchListClientFeatureEnv); got != "false" {
		t.Fatalf("expected WatchListClient to be disabled, got %q", got)
	}
}

func TestEnsureKubeFeatureDefaultsRespectsExplicitOptIn(t *testing.T) {
	withFeatureEnv(t, "true", true)
	EnsureKubeFeatureDefaults()
	if got := os.Getenv(WatchListClientFeatureEnv); got != "true" {
		t.Fatalf("explicit opt-in was overwritten, got %q", got)
	}
}

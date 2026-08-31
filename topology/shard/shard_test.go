package shard

import "testing"

func TestShardZeroValue(t *testing.T) {
	t.Parallel()

	var current Shard

	if got := current.ID(); got != "" {
		t.Fatalf("ID() = %q, want empty", got)
	}

	if value, ok := current.Label("region"); ok || value != "" {
		t.Fatalf(
			"Label() = %q, %v; want empty, false",
			value,
			ok,
		)
	}

	if labels := current.Labels(); labels != nil {
		t.Fatalf("Labels() = %#v, want nil", labels)
	}

	if pool := current.Pool(); pool != nil {
		t.Fatalf("Pool() = %p, want nil", pool)
	}
}

func TestShardDelegatesPoolMetadata(t *testing.T) {
	t.Parallel()

	pool := newTestPool(
		t,
		"shard-a",
		map[string]string{
			"region": "eu-west",
			"role":   "",
		},
	)

	topology, err := NewTopology(pool)
	if err != nil {
		t.Fatalf("NewTopology() error = %v", err)
	}
	t.Cleanup(func() {
		_ = topology.Close()
	})

	resolved := topology.At(0)

	if got, want := resolved.ID(), ID("shard-a"); got != want {
		t.Fatalf("ID() = %q, want %q", got, want)
	}

	if got, ok := resolved.Label("region"); !ok || got != "eu-west" {
		t.Fatalf("Label(region) = %q, %v", got, ok)
	}

	if got, ok := resolved.Label("role"); !ok || got != "" {
		t.Fatalf("Label(role) = %q, %v", got, ok)
	}

	labels := resolved.Labels()
	labels["region"] = "changed"

	if got, _ := resolved.Label("region"); got != "eu-west" {
		t.Fatalf(
			"Label(region) after mutation = %q, want eu-west",
			got,
		)
	}

	if resolved.Pool() != pool {
		t.Fatal("Pool() did not return the backing pool")
	}
}

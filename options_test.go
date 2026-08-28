package xch

import (
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolMetadata(t *testing.T) {
	t.Parallel()

	labels := map[string]string{
		"region": "eu-west",
		"source": "initial",
	}
	labelsOption := WithLabels(labels)

	labels["source"] = "changed"
	labels["ignored"] = "value"

	pool, err := New(
		&clickhouse.Options{},
		WithName("  analytics  "),
		labelsOption,
		WithLabel("region", "eu-central"),
		WithLabels(map[string]string{
			"environment": "production",
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	assert.Equal(t, "analytics", pool.Name())
	assert.Equal(t, map[string]string{
		"region":      "eu-central",
		"source":      "initial",
		"environment": "production",
	}, pool.Labels())

	got := pool.Labels()
	got["region"] = "modified"
	got["new"] = "value"

	assert.Equal(t, map[string]string{
		"region":      "eu-central",
		"source":      "initial",
		"environment": "production",
	}, pool.Labels())
}

func TestOptionsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		option  Option
		wantErr string
	}{
		{
			name:    "blank name",
			option:  WithName(" \t\n "),
			wantErr: "xch: apply option: pool name must not be blank",
		},
		{
			name:    "empty label key",
			option:  WithLabel("", "value"),
			wantErr: "xch: apply option: label key must not be empty",
		},
		{
			name: "empty labels key",
			option: WithLabels(map[string]string{
				"": "value",
			}),
			wantErr: "xch: apply option: label key must not be empty",
		},
		{
			name:    "nil option",
			option:  nil,
			wantErr: "xch: option is nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(&clickhouse.Options{}, test.option)
			require.EqualError(t, err, test.wantErr)
		})
	}
}

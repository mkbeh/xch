package resolver

import (
	"errors"

	"github.com/mkbeh/xch/topology/shard"
)

func requireTopology(topology *shard.Topology) error {
	if topology == nil || topology.Len() == 0 {
		return errors.New("xch/topology/shard/resolver: topology is nil or empty")
	}

	return nil
}

func requireShardID(id shard.ID) error {
	if id == "" {
		return errors.New("shard ID must not be empty")
	}

	return nil
}

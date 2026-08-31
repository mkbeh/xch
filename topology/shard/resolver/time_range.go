package resolver

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/mkbeh/xch/topology/shard"
)

// TimeRange maps the bounded half-open interval [Start, End) to one shard.
//
// Start must be before End. Ranges may be supplied in any order. Gaps are
// allowed and resolve to shard.ErrNoShard.
type TimeRange struct {
	Start   time.Time
	End     time.Time
	ShardID shard.ID
}

// TimeRangeResolver resolves time instants through bounded, non-overlapping
// ranges.
type TimeRangeResolver struct {
	ranges []timeRangeEntry
}

type timeRangeEntry struct {
	start time.Time
	end   time.Time
	shard shard.Shard

	sourceIndex int
}

// NewTimeRange creates a resolver from bounded, non-overlapping half-open time
// ranges.
//
// Boundaries are normalized to UTC. Every ShardID is resolved to its immutable
// Shard handle once. The caller's slice and time values are not modified.
func NewTimeRange(
	topology *shard.Topology,
	ranges []TimeRange,
) (*TimeRangeResolver, error) {
	if err := requireTopology(topology); err != nil {
		return nil, err
	}
	if len(ranges) == 0 {
		return nil, errors.New("xch/topology/shard/resolver: time range resolver requires at least one range")
	}

	entries := make([]timeRangeEntry, len(ranges))

	for index, valueRange := range ranges {
		if err := requireShardID(valueRange.ShardID); err != nil {
			return nil, fmt.Errorf("xch/topology/shard/resolver: time range %d: %w", index, err)
		}

		start := valueRange.Start.UTC()
		end := valueRange.End.UTC()

		if !start.Before(end) {
			return nil, fmt.Errorf("xch/topology/shard/resolver: time range %d must satisfy start < end", index)
		}

		resolved, ok := topology.Shard(valueRange.ShardID)
		if !ok {
			return nil, fmt.Errorf(
				"xch/topology/shard/resolver: time range %d: %w",
				index,
				&shard.UnknownShardError{ShardID: valueRange.ShardID},
			)
		}

		entries[index] = timeRangeEntry{
			start:       start,
			end:         end,
			shard:       resolved,
			sourceIndex: index,
		}
	}

	slices.SortStableFunc(
		entries,
		func(left, right timeRangeEntry) int {
			return left.start.Compare(right.start)
		},
	)

	for index := 1; index < len(entries); index++ {
		previous := entries[index-1]
		current := entries[index]

		if !previous.end.After(current.start) {
			continue
		}

		return nil, fmt.Errorf(
			"xch/topology/shard/resolver: time ranges %d and %d overlap",
			previous.sourceIndex,
			current.sourceIndex,
		)
	}

	return &TimeRangeResolver{ranges: entries}, nil
}

// Resolve returns the shard whose time range contains key.
//
// Resolve performs only an in-memory lookup. It does not consult topology,
// acquire a connection, or execute a ClickHouse query.
func (resolver *TimeRangeResolver) Resolve(key time.Time) (shard.Shard, error) {
	if resolver == nil || len(resolver.ranges) == 0 {
		return shard.Shard{}, errors.New("xch/topology/shard/resolver: time range resolver is not initialized")
	}

	key = key.UTC()

	index := sort.Search(
		len(resolver.ranges),
		func(index int) bool {
			return key.Before(resolver.ranges[index].end)
		},
	)

	if index == len(resolver.ranges) {
		return shard.Shard{}, shard.ErrNoShard
	}

	entry := resolver.ranges[index]
	if key.Before(entry.start) {
		return shard.Shard{}, shard.ErrNoShard
	}

	return entry.shard, nil
}

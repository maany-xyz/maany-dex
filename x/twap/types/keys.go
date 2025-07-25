package types

import (
	"bytes"
	"errors"
	"fmt"
	time "time"

	"github.com/cosmos/gogoproto/proto"

	storetypes "cosmossdk.io/store/types"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/osmoutils"
)

const (
	ModuleName = "twap"

	StoreKey          = ModuleName
	TransientStoreKey = "transient_" + ModuleName // this is silly we have to do this
	RouterKey         = ModuleName

	QuerierRoute = ModuleName
	// Contract: Coin denoms cannot contain this character
	KeySeparator = "|"
)

var (
	PruningStateKey = []byte{0x01}
	// TODO: Delete in v26
	DeprecatedHistoricalTWAPsIsPruningKey = []byte{0x02}
	mostRecentTWAPsNoSeparator            = "recent_twap"
	historicalTWAPPoolIndexNoSeparator    = "historical_pool_index"

	// We do key management to let us easily meet the goals of (AKA minimal iteration):
	// * Get most recent twap for a (pool id, asset 1, asset 2) with no iteration
	// * Get all records for all pools, within a given time range
	// * Get all records for a (pool id, asset 1, asset 2), within a given time range

	// format is just pool id | denom1 | denom2
	// made for getting most recent key
	mostRecentTWAPsPrefix = mostRecentTWAPsNoSeparator + KeySeparator
	// format is pool id | denom1 | denom2 | time
	// made for efficiently getting records given (pool id, denom1, denom2) and time bounds
	HistoricalTWAPPoolIndexPrefix = historicalTWAPPoolIndexNoSeparator + KeySeparator
)

// TODO: make utility command to automatically interlace separators

func FormatKeyPoolTwapRecords(poolId uint64) []byte {
	return []byte(fmt.Sprintf("%s%d", HistoricalTWAPPoolIndexPrefix, poolId))
}

func FormatMostRecentTWAPKey(poolId uint64, denom1, denom2 string) []byte {
	poolIdS := osmoutils.FormatFixedLengthU64(poolId)
	return []byte(fmt.Sprintf("%s%s%s%s%s%s", mostRecentTWAPsPrefix, poolIdS, KeySeparator, denom1, KeySeparator, denom2))
}

func FormatHistoricalPoolIndexDenomPairTWAPKey(poolId uint64, denom1, denom2 string) []byte {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s%d%s%s%s%s%s", HistoricalTWAPPoolIndexPrefix, poolId, KeySeparator, denom1, KeySeparator, denom2, KeySeparator)
	return buffer.Bytes()
}

func FormatHistoricalPoolIndexTWAPKey(poolId uint64, denom1, denom2 string, accumulatorWriteTime time.Time) []byte {
	timeS := osmoutils.FormatTimeString(accumulatorWriteTime)
	return FormatHistoricalPoolIndexTWAPKeyFromStrTime(poolId, denom1, denom2, timeS)
}

func FormatHistoricalPoolIndexTWAPKeyFromStrTime(poolId uint64, denom1, denom2 string, accumulatorWriteTimeString string) []byte {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s%d%s%s%s%s%s%s", HistoricalTWAPPoolIndexPrefix, poolId, KeySeparator, denom1, KeySeparator, denom2, KeySeparator, accumulatorWriteTimeString)
	return buffer.Bytes()
}

func FormatHistoricalPoolIndexTimePrefix(poolId uint64, denom1, denom2 string) []byte {
	return []byte(fmt.Sprintf("%s%d%s%s%s%s%s", HistoricalTWAPPoolIndexPrefix, poolId, KeySeparator, denom1, KeySeparator, denom2, KeySeparator))
}

func FormatHistoricalPoolIndexTimeSuffix(poolId uint64, denom1, denom2 string, accumulatorWriteTime time.Time) []byte {
	timeS := osmoutils.FormatTimeString(accumulatorWriteTime)
	// . acts as a suffix for lexicographical orderings
	return []byte(fmt.Sprintf("%s%d%s%s%s%s%s%s.", HistoricalTWAPPoolIndexPrefix, poolId, KeySeparator, denom1, KeySeparator, denom2, KeySeparator, timeS))
}

// GetAllMostRecentTwapsForPool returns all of the most recent twap records for a pool id.
// if the pool id doesn't exist, then this returns a blank list.
func GetAllMostRecentTwapsForPool(store storetypes.KVStore, poolId uint64) ([]TwapRecord, error) {
	poolIdS := osmoutils.FormatFixedLengthU64(poolId)
	poolIdPlusOneS := osmoutils.FormatFixedLengthU64(poolId + 1)
	startPrefix := fmt.Sprintf("%s%s%s", mostRecentTWAPsPrefix, poolIdS, KeySeparator)
	endPrefix := fmt.Sprintf("%s%s%s", mostRecentTWAPsPrefix, poolIdPlusOneS, KeySeparator)
	return osmoutils.GatherValuesFromStore(store, []byte(startPrefix), []byte(endPrefix), ParseTwapFromBz)
}

func GetMostRecentTwapForPool(store storetypes.KVStore, poolId uint64, denom1, denom2 string) (TwapRecord, error) {
	key := FormatMostRecentTWAPKey(poolId, denom1, denom2)
	bz := store.Get(key)
	return ParseTwapFromBz(bz)
}

func ParseTwapFromBz(bz []byte) (twap TwapRecord, err error) {
	if len(bz) == 0 {
		return TwapRecord{}, errors.New("twap not found")
	}
	err = proto.Unmarshal(bz, &twap)
	if twap.GeometricTwapAccumulator.IsNil() {
		twap.GeometricTwapAccumulator = osmomath.ZeroDec()
	}
	return twap, err
}

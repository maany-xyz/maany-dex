package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/neutron-org/neutron/v5/x/epochs/types"
)

func TestEpochsExportGenesis(t *testing.T) {
	ctx, epochsKeeper := Setup()

	chainStartTime := ctx.BlockTime()
	chainStartHeight := ctx.BlockHeight()

	genesis := epochsKeeper.ExportGenesis(ctx)
	require.Len(t, genesis.Epochs, 3)

	expectedEpochs := types.DefaultGenesis().Epochs
	for i := 0; i < len(expectedEpochs); i++ {
		expectedEpochs[i].CurrentEpochStartHeight = chainStartHeight
		expectedEpochs[i].CurrentEpochStartTime = chainStartTime
	}
	require.Equal(t, expectedEpochs, genesis.Epochs)
}

func TestEpochsInitGenesis(t *testing.T) {
	ctx, epochsKeeper := Setup()

	// On init genesis, default epochs information is set
	// To check init genesis again, should make it fresh status
	epochInfos := epochsKeeper.AllEpochInfos(ctx)
	for _, epochInfo := range epochInfos {
		epochsKeeper.DeleteEpochInfo(ctx, epochInfo.Identifier)
	}

	now := time.Now()
	ctx = ctx.WithBlockHeight(1)
	ctx = ctx.WithBlockTime(now)

	// test genesisState validation
	genesisState := types.GenesisState{
		Epochs: []types.EpochInfo{
			{
				Identifier:              "monthly",
				StartTime:               time.Time{},
				Duration:                time.Hour * 24,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    true,
			},
			{
				Identifier:              "monthly",
				StartTime:               time.Time{},
				Duration:                time.Hour * 24,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    true,
			},
		},
	}
	require.EqualError(t, genesisState.Validate(), "epoch identifier should be unique")

	genesisState = types.GenesisState{
		Epochs: []types.EpochInfo{
			{
				Identifier:              "monthly",
				StartTime:               time.Time{},
				Duration:                time.Hour * 24,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: ctx.BlockHeight(),
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    true,
			},
		},
	}

	epochsKeeper.InitGenesis(ctx, genesisState)
	epochInfo := epochsKeeper.GetEpochInfo(ctx, "monthly")
	require.Equal(t, epochInfo.Identifier, "monthly")
	require.Equal(t, epochInfo.StartTime.UTC().String(), now.UTC().String())
	require.Equal(t, epochInfo.Duration, time.Hour*24)
	require.Equal(t, epochInfo.CurrentEpoch, int64(0))
	require.Equal(t, epochInfo.CurrentEpochStartHeight, ctx.BlockHeight())
	require.Equal(t, epochInfo.CurrentEpochStartTime.UTC().String(), time.Time{}.String())
	require.Equal(t, epochInfo.EpochCountingStarted, true)
}

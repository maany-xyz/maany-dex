package keeper_test

import (
	gocontext "context"

	"github.com/neutron-org/neutron/v5/x/epochs/types"
)

func (s *KeeperTestSuite) TestQueryEpochInfos() {
	s.SetupTest()
	queryClient := s.queryClient

	// Check that querying epoch infos on default genesis returns the default genesis epoch infos
	epochInfosResponse, err := queryClient.EpochInfos(gocontext.Background(), &types.QueryEpochsInfoRequest{})
	s.Require().NoError(err)
	s.Require().Len(epochInfosResponse.Epochs, 3)
	expectedEpochs := types.DefaultGenesis().Epochs
	for id := range expectedEpochs {
		expectedEpochs[id].StartTime = s.Ctx.BlockTime()
		expectedEpochs[id].CurrentEpochStartHeight = s.Ctx.BlockHeight()
	}

	s.Require().Equal(expectedEpochs, epochInfosResponse.Epochs)
}

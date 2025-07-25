package cli

import (
	"github.com/spf13/cobra"

	"github.com/neutron-org/neutron/v5/osmoutils/osmocli"
	"github.com/neutron-org/neutron/v5/x/twap/client/queryproto"
	"github.com/neutron-org/neutron/v5/x/txfees/types"
)

// GetQueryCmd returns the cli query commands for this module.
func GetQueryCmd() *cobra.Command {
	cmd := osmocli.QueryIndexCmd(types.ModuleName)

	cmd.AddCommand(
		GetCmdFeeTokens(),
		GetCmdDenomPoolID(),
		GetCmdBaseDenom(),
		osmocli.GetParams[*queryproto.ParamsRequest](
			types.ModuleName, queryproto.NewQueryClient),
	)

	osmocli.AddQueryCmd(cmd, types.NewQueryClient, GetCmdQueryBaseFee)

	return cmd
}

func GetCmdFeeTokens() *cobra.Command {
	return osmocli.SimpleQueryCmd[*types.QueryFeeTokensRequest](
		"fee-tokens",
		"Query the list of non-basedenom fee tokens and their associated pool ids",
		`{{.Short}}{{.ExampleHeader}}
{{.CommandPrefix}} fee-tokens
`,
		types.ModuleName, types.NewQueryClient,
	)
}

func GetCmdDenomPoolID() *cobra.Command {
	return osmocli.SimpleQueryCmd[*types.QueryDenomPoolIdRequest](
		"denom-pool-id",
		"Query the pool id associated with a specific whitelisted fee token",
		`{{.Short}}{{.ExampleHeader}}
{{.CommandPrefix}} denom-pool-id [denom]
`,
		types.ModuleName, types.NewQueryClient,
	)
}

func GetCmdBaseDenom() *cobra.Command {
	return osmocli.SimpleQueryCmd[*types.QueryBaseDenomRequest](
		"base-denom",
		"Query the base fee denom",
		`{{.Short}}{{.ExampleHeader}}
{{.CommandPrefix}} base-denom
`,
		types.ModuleName, types.NewQueryClient,
	)
}

func GetCmdQueryBaseFee() (*osmocli.QueryDescriptor, *types.QueryEipBaseFeeRequest) {
	return &osmocli.QueryDescriptor{
		Use:   "base-fee",
		Short: "Query the eip base fee.",
		Long: `{{.Short}}{{.ExampleHeader}}
{{.CommandPrefix}} base-fee`,
		QueryFnName: "GetEipBaseFee",
	}, &types.QueryEipBaseFeeRequest{}
}

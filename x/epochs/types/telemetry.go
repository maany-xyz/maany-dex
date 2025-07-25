package types

import "github.com/neutron-org/neutron/v5/osmoutils/observability"

var (
	// epoch_hook_failed
	//
	// counter that is increased if epoch hook fails
	//
	// Has the following labels:
	// * module_name - the name of the module that errored or panicked
	// * err - the error or panic returned
	// * is_before_hook - true if this is a before epoch hook. False otherwise.
	EpochHookFailedMetricName = formatEpochMetricName("hook_failed")
)

// formatTxFeesMetricName formats the epochs module metric name.
func formatEpochMetricName(metricName string) string {
	return observability.FormatMetricName(ModuleName, metricName)
}

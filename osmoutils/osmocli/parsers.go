package osmocli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/osmoutils"
)

var DefaultGovAuthority = sdk.AccAddress(address.Module("gov"))

const (
	FlagIsExpedited = "is-expedited"
	FlagAuthority   = "authority"
)

func isFieldDeprecated(field reflect.StructField) bool {
	tag := string(field.Tag)
	return strings.Contains(tag, "deprecated:\"true\"")
}

// Parses arguments 1-1 from args
// makes an exception, where it allows Pagination to come from flags.
func ParseFieldsFromFlagsAndArgs[reqP any](flagAdvice FlagAdvice, flags *pflag.FlagSet, args []string) (reqP, error) {
	req := osmoutils.MakeNew[reqP]()
	v := reflect.ValueOf(req).Elem()
	t := v.Type()

	argIndexOffset := 0
	// Iterate over the fields in the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if isFieldDeprecated(field) {
			argIndexOffset -= 1
			continue
		}

		arg := ""
		if len(args) > i+argIndexOffset {
			arg = args[i+argIndexOffset]
		}
		usedArg, err := ParseField(v, t, i, arg, flagAdvice, flags)
		if err != nil {
			return req, err
		}
		if !usedArg {
			argIndexOffset -= 1
		}
	}
	return req, nil
}

func ParseNumFields[reqP any]() int {
	req := osmoutils.MakeNew[reqP]()
	v := reflect.ValueOf(req).Elem()
	t := v.Type()
	return t.NumField()
}

func ParseExpectedQueryFnName[reqP any]() string {
	req := osmoutils.MakeNew[reqP]()
	v := reflect.ValueOf(req).Elem()
	s := v.Type().String()
	// handle some non-std queries
	var prefixTrimmed string
	if strings.Contains(s, "Query") {
		prefixTrimmed = strings.Split(s, "Query")[1]
	} else {
		prefixTrimmed = strings.Split(s, ".")[1]
	}
	suffixTrimmed := strings.TrimSuffix(prefixTrimmed, "Request")
	return suffixTrimmed
}

func ParseHasPagination[reqP any]() bool {
	req := osmoutils.MakeNew[reqP]()
	t := reflect.ValueOf(req).Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		fType := t.Field(i)
		if fType.Type.String() == paginationType {
			return true
		}
	}
	return false
}

const paginationType = "*query.PageRequest"

// ParseField parses field #fieldIndex from either an arg or a flag.
// Returns true if it was parsed from an argument.
// Returns error if there was an issue in parsing this field.
func ParseField(v reflect.Value, t reflect.Type, fieldIndex int, arg string, flagAdvice FlagAdvice, flags *pflag.FlagSet) (bool, error) {
	fVal := v.Field(fieldIndex)
	fType := t.Field(fieldIndex)
	// fmt.Printf("Field %d: %s %s %s\n", fieldIndex, fType.Name, fType.Type, fType.Type.Kind())

	lowercaseFieldNameStr := strings.ToLower(fType.Name)
	if parseFn, ok := flagAdvice.CustomFieldParsers[lowercaseFieldNameStr]; ok {
		v, usedArg, err := parseFn(arg, flags)
		if err == nil {
			fVal.Set(reflect.ValueOf(v))
		}
		return usedArg, err
	}

	parsedFromFlag, err := ParseFieldFromFlag(fVal, fType, flagAdvice, flags)
	if err != nil {
		return false, err
	}
	if parsedFromFlag {
		return false, nil
	}
	return true, ParseFieldFromArg(fVal, fType, arg)
}

// ParseFieldFromFlag attempts to parses the value of a field in a struct from a flag.
// The field is identified by the provided `reflect.StructField`.
// The flag advice and `pflag.FlagSet` are used to determine the flag to parse the field from.
// If the field corresponds to a value from a flag, true is returned.
// Otherwise, `false` is returned.
// In the true case, the parsed value is set on the provided `reflect.Value`.
// An error is returned if there is an issue parsing the field from the flag.
func ParseFieldFromFlag(fVal reflect.Value, fType reflect.StructField, flagAdvice FlagAdvice, flags *pflag.FlagSet) (bool, error) {
	lowercaseFieldNameStr := strings.ToLower(fType.Name)
	if flagName, ok := flagAdvice.CustomFlagOverrides[lowercaseFieldNameStr]; ok {
		return true, parseFieldFromDirectlySetFlag(fVal, fType, flagName, flags)
	}

	kind := fType.Type.Kind()
	switch kind {
	case reflect.String:
		if flagAdvice.IsTx {
			// matchesFieldName is true if lowercaseFieldNameStr is the same as TxSenderFieldName,
			// or if TxSenderFieldName is left blank, then matches fields named "sender" or "owner"
			matchesFieldName := (flagAdvice.TxSenderFieldName == lowercaseFieldNameStr) ||
				(flagAdvice.TxSenderFieldName == "" && (lowercaseFieldNameStr == "sender" || lowercaseFieldNameStr == "owner"))
			if matchesFieldName {
				fVal.SetString(flagAdvice.FromValue)
				return true, nil
			}
		}
	case reflect.Ptr:
		if flagAdvice.HasPagination {
			typeStr := fType.Type.String()
			if typeStr == paginationType {
				pageReq, err := client.ReadPageRequest(flags)
				if err != nil {
					return true, err
				}
				fVal.Set(reflect.ValueOf(pageReq))
				return true, nil
			}
		}
	}
	return false, nil
}

func parseFieldFromDirectlySetFlag(fVal reflect.Value, fType reflect.StructField, flagName string, flags *pflag.FlagSet) error {
	// get string. If its a string great, run through arg parser. Otherwise try setting directly
	s, err := flags.GetString(flagName)
	if err != nil {
		flag := flags.Lookup(flagName)
		if flag == nil {
			return fmt.Errorf("Programmer set the flag name wrong. Flag %s does not exist", flagName)
		}
		t := flag.Value.Type()
		if t == "uint64" {
			u, err := flags.GetUint64(flagName)
			if err != nil {
				return err
			}
			fVal.SetUint(u)
			return nil
		}
	}
	return ParseFieldFromArg(fVal, fType, s)
}

func ParseFieldFromArg(fVal reflect.Value, fType reflect.StructField, arg string) error {
	// We can't pass in a negative number due to the way pflags works...
	// This is an (extraordinarily ridiculous) workaround that checks if a negative int is encapsulated in square brackets,
	// and if so, trims the square brackets
	if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") && arg[1] == '-' {
		arg = strings.TrimPrefix(arg, "[")
		arg = strings.TrimSuffix(arg, "]")
	}

	switch fType.Type.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(arg)
		if err != nil {
			return fmt.Errorf("could not parse %s as bool for field %s: %w", arg, fType.Name, err)
		}
		fVal.SetBool(b)
		return nil
	// SetUint allows anyof type u8, u16, u32, u64, and uint
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		u, err := ParseUint(arg, fType.Name)
		if err != nil {
			return err
		}
		fVal.SetUint(u)
		return nil
	// SetInt allows anyof type i8,i16,i32,i64 and int
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		typeStr := fType.Type.String()
		var i int64
		var err error
		if typeStr == "time.Duration" {
			dur, err2 := time.ParseDuration(arg)
			i, err = int64(dur), err2
		} else {
			i, err = ParseInt(arg, fType.Name)
		}
		if err != nil {
			return err
		}
		fVal.SetInt(i)
		return nil
	case reflect.Float32, reflect.Float64:
		typeStr := fType.Type.String()
		f, err := ParseFloat(arg, typeStr)
		if err != nil {
			return err
		}
		fVal.SetFloat(f)
		return nil
	case reflect.String:
		s, err := ParseDenom(arg, fType.Name)
		if err != nil {
			return err
		}
		fVal.SetString(s)
		return nil
	case reflect.Ptr:
	case reflect.Slice:
		typeStr := fType.Type.String()
		if typeStr == "[]uint64" {
			// Parse comma-separated uint64 values into []uint64 slice
			strValues := strings.Split(arg, ",")
			values := make([]uint64, len(strValues))
			for i, strValue := range strValues {
				u, err := strconv.ParseUint(strValue, 10, 64)
				if err != nil {
					return err
				}
				values[i] = u
			}
			fVal.Set(reflect.ValueOf(values))
			return nil
		}
		if typeStr == "types.Coins" {
			coins, err := ParseCoins(arg, fType.Name)
			if err != nil {
				return err
			}
			fVal.Set(reflect.ValueOf(coins))
			return nil
		}
	case reflect.Struct:
		typeStr := fType.Type.String()
		var v any
		var err error
		if typeStr == "types.Coin" {
			v, err = ParseCoin(arg, fType.Name)
		} else if typeStr == "math.Int" {
			v, err = ParseSdkInt(arg, fType.Name)
		} else if typeStr == "time.Time" {
			v, err = ParseUnixTime(arg, fType.Name)
		} else if typeStr == "math.LegacyDec" {
			v, err = ParseSdkDec(arg, fType.Name)
		} else {
			return fmt.Errorf("struct field type not recognized. Got type %v", fType)
		}

		if err != nil {
			return err
		}
		fVal.Set(reflect.ValueOf(v))
		return nil
	}
	fmt.Println(fType.Type.Kind().String())
	return fmt.Errorf("field type not recognized. Got type %v", fType)
}

func ParseUint(arg string, fieldName string) (uint64, error) {
	v, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s as uint for field %s: %w", arg, fieldName, err)
	}
	return v, nil
}

func ParseFloat(arg string, fieldName string) (float64, error) {
	v, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s as float for field %s: %w", arg, fieldName, err)
	}
	return v, nil
}

func ParseInt(arg string, fieldName string) (int64, error) {
	v, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s as int for field %s: %w", arg, fieldName, err)
	}
	return v, nil
}

func ParseUnixTime(arg string, fieldName string) (time.Time, error) {
	timeUnix, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		parsedTime, err := time.Parse(sdk.SortableTimeFormat, arg)
		if err != nil {
			return time.Time{}, fmt.Errorf("could not parse %s as time for field %s: %w", arg, fieldName, err)
		}

		return parsedTime, nil
	}
	startTime := time.Unix(timeUnix, 0)
	return startTime, nil
}

func ParseDenom(arg string, fieldName string) (string, error) {
	return strings.TrimSpace(arg), nil
}

// TODO: Make this able to read from some local alias file for denoms.
func ParseCoin(arg string, fieldName string) (sdk.Coin, error) {
	coin, err := sdk.ParseCoinNormalized(arg)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("could not parse %s as sdk.Coin for field %s: %w", arg, fieldName, err)
	}
	return coin, nil
}

// TODO: Make this able to read from some local alias file for denoms.
func ParseCoins(arg string, fieldName string) (sdk.Coins, error) {
	coins, err := sdk.ParseCoinsNormalized(arg)
	if err != nil {
		return sdk.Coins{}, fmt.Errorf("could not parse %s as sdk.Coins for field %s: %w", arg, fieldName, err)
	}
	return coins, nil
}

// TODO: This really shouldn't be getting used in the CLI, its misdesign on the CLI ux
func ParseSdkInt(arg string, fieldName string) (osmomath.Int, error) {
	i, ok := osmomath.NewIntFromString(arg)
	if !ok {
		return osmomath.Int{}, fmt.Errorf("could not parse %s as osmomath.Int for field %s", arg, fieldName)
	}
	return i, nil
}

func ParseSdkDec(arg, fieldName string) (osmomath.Dec, error) {
	i, err := osmomath.NewDecFromStr(arg)
	if err != nil {
		return osmomath.Dec{}, fmt.Errorf("could not parse %s as osmomath.Dec for field %s: %w", arg, fieldName, err)
	}
	return i, nil
}

func ParseStringTo2DArray(input string) ([][]uint64, error) {
	// Split the input string into sub-arrays
	subArrays := strings.Split(input, ";")

	// Initialize the 2D array
	result := make([][]uint64, len(subArrays))

	// Iterate over each sub-array
	for i, subArray := range subArrays {
		// Split the sub-array into elements
		elements := strings.Split(subArray, ",")

		// Initialize the sub-array in the 2D array
		result[i] = make([]uint64, len(elements))

		// Iterate over each element
		for j, element := range elements {
			// Parse the element into a uint64
			value, err := strconv.ParseUint(element, 10, 64)
			if err != nil {
				return nil, err
			}

			// Store the value in the 2D array
			result[i][j] = value
		}
	}

	return result, nil
}

// ParseUint64SliceToString converts a slice of uint64 values into a string.
// Each uint64 value in the slice is converted to a string using base 10.
// The resulting strings are then joined together with a comma separator.
// The resulting string is returned.
func ParseUint64SliceToString(values []uint64) string {
	strs := make([]string, len(values))
	for i, v := range values {
		strs[i] = strconv.FormatUint(v, 10)
	}
	return strings.Join(strs, ", ")
}

func GetProposalInfo(cmd *cobra.Command) (client.Context, string, string, sdk.Coins, bool, sdk.AccAddress, error) {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return client.Context{}, "", "", nil, false, nil, err
	}

	proposalTitle, err := cmd.Flags().GetString(cli.FlagTitle)
	if err != nil {
		return clientCtx, proposalTitle, "", nil, false, nil, err
	}

	summary, err := cmd.Flags().GetString(cli.FlagSummary)
	if err != nil {
		return client.Context{}, proposalTitle, summary, nil, false, nil, err
	}

	depositArg, err := cmd.Flags().GetString(cli.FlagDeposit)
	if err != nil {
		return client.Context{}, proposalTitle, summary, nil, false, nil, err
	}

	deposit, err := sdk.ParseCoinsNormalized(depositArg)
	if err != nil {
		return client.Context{}, proposalTitle, summary, deposit, false, nil, err
	}

	isExpedited, err := cmd.Flags().GetBool(FlagIsExpedited)
	if err != nil {
		return client.Context{}, proposalTitle, summary, deposit, false, nil, err
	}

	authorityString, err := cmd.Flags().GetString(FlagAuthority)
	if err != nil {
		return client.Context{}, proposalTitle, summary, deposit, false, nil, err
	}
	authority, err := sdk.AccAddressFromBech32(authorityString)
	if err != nil {
		return client.Context{}, proposalTitle, summary, deposit, false, nil, err
	}

	return clientCtx, proposalTitle, summary, deposit, isExpedited, authority, nil
}

func AddCommonProposalFlags(cmd *cobra.Command) {
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagSummary, "", "Summary of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	cmd.Flags().Bool(FlagIsExpedited, false, "Whether the proposal is expedited")
	cmd.Flags().String(FlagAuthority, DefaultGovAuthority.String(), "The address of the governance account. Default is the sdk gov module account")
}

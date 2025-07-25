package osmocli

import (
	"reflect"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/neutron-org/neutron/v5/osmomath"
)

type testingStruct struct {
	Int      int64
	UInt     uint64
	String   string
	Float    float64
	Duration time.Duration
	Pointer  *testingStruct
	Slice    sdk.Coins
	Struct   interface{}
	Dec      osmomath.Dec
}

func TestParseFieldFromArg(t *testing.T) {
	tests := map[string]struct {
		testingStruct
		arg        string
		fieldIndex int

		expectedStruct testingStruct
		expectingErr   bool
	}{
		"Int value changes from -20 to 10": {
			testingStruct:  testingStruct{Int: -20},
			arg:            "10",
			fieldIndex:     0,
			expectedStruct: testingStruct{Int: 10},
		},
		"Attempt to change Int value 20 to string value": { // does not return error, simply does not change the struct
			testingStruct:  testingStruct{Int: 20},
			arg:            "hello",
			fieldIndex:     0,
			expectedStruct: testingStruct{Int: 20},
		},
		"UInt value changes from 20 to 10": {
			testingStruct:  testingStruct{UInt: 20},
			arg:            "10",
			fieldIndex:     1,
			expectedStruct: testingStruct{UInt: 10},
		},
		"String value change": {
			testingStruct:  testingStruct{String: "hello"},
			arg:            "world",
			fieldIndex:     2,
			expectedStruct: testingStruct{String: "world"},
		},
		"Changing unset value (simply sets the value)": {
			testingStruct:  testingStruct{Int: 20},
			arg:            "hello",
			fieldIndex:     2,
			expectedStruct: testingStruct{Int: 20, String: "hello"},
		},
		"Float value change": {
			testingStruct:  testingStruct{Float: 20.0},
			arg:            "30.0",
			fieldIndex:     3,
			expectedStruct: testingStruct{Float: 30.0},
		},
		"Duration value changes from .Hour to .Second": {
			testingStruct:  testingStruct{Duration: time.Hour},
			arg:            "1s",
			fieldIndex:     4,
			expectedStruct: testingStruct{Duration: time.Second},
		},
		"Attempt to change pointer": { // for reflect.Ptr kind ParseFieldFromArg does nothing, hence no changes take place
			testingStruct:  testingStruct{Pointer: &testingStruct{}},
			arg:            "*whatever",
			fieldIndex:     5,
			expectedStruct: testingStruct{Pointer: &testingStruct{}},
		},
		"Slice change": {
			testingStruct: testingStruct{Slice: sdk.Coins{
				sdk.NewCoin("foo", osmomath.NewInt(100)),
				sdk.NewCoin("bar", osmomath.NewInt(100)),
			}},
			arg:        "10foo,10bar", // Should be of a format suitable for ParseCoinsNormalized
			fieldIndex: 6,
			expectedStruct: testingStruct{Slice: sdk.Coins{ // swapped places due to lexicographic order
				sdk.NewCoin("bar", osmomath.NewInt(10)),
				sdk.NewCoin("foo", osmomath.NewInt(10)),
			}},
		},
		"Struct (sdk.Coin) change": {
			testingStruct:  testingStruct{Struct: sdk.NewCoin("bar", osmomath.NewInt(10))}, // only supports osmomath.Int, sdk.Coin or time.Time, other structs are not recognized
			arg:            "100bar",
			fieldIndex:     7,
			expectedStruct: testingStruct{Struct: sdk.NewCoin("bar", osmomath.NewInt(10))},
		},
		"Unrecognizable struct": {
			testingStruct: testingStruct{Struct: testingStruct{}}, // only supports osmomath.Int, sdk.Coin or time.Time, other structs are not recognized
			arg:           "whatever",
			fieldIndex:    7,
			expectingErr:  true,
		},
		"Multiple fields in struct are set": {
			testingStruct:  testingStruct{Int: 20, UInt: 10, String: "hello", Pointer: &testingStruct{}},
			arg:            "world",
			fieldIndex:     2,
			expectedStruct: testingStruct{Int: 20, UInt: 10, String: "world", Pointer: &testingStruct{}},
		},
		"All fields in struct set": {
			testingStruct: testingStruct{
				Int:      20,
				UInt:     10,
				String:   "hello",
				Float:    30.0,
				Duration: time.Second,
				Pointer:  &testingStruct{},
				Slice: sdk.Coins{
					sdk.NewCoin("foo", osmomath.NewInt(100)),
					sdk.NewCoin("bar", osmomath.NewInt(100)),
				},
				Struct: sdk.NewCoin("bar", osmomath.NewInt(10)),
			},
			arg:        "1foo,15bar",
			fieldIndex: 6,
			expectedStruct: testingStruct{
				Int:      20,
				UInt:     10,
				String:   "hello",
				Float:    30.0,
				Duration: time.Second,
				Pointer:  &testingStruct{},
				Slice: sdk.Coins{
					sdk.NewCoin("bar", osmomath.NewInt(15)),
					sdk.NewCoin("foo", osmomath.NewInt(1)),
				},
				Struct: sdk.NewCoin("bar", osmomath.NewInt(10)),
			},
		},
		"Dec struct": {
			testingStruct:  testingStruct{Dec: osmomath.MustNewDecFromStr("100")},
			arg:            "10",
			fieldIndex:     8,
			expectedStruct: testingStruct{Dec: osmomath.MustNewDecFromStr("10")},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			val := reflect.ValueOf(&tc.testingStruct).Elem()
			typ := reflect.TypeOf(&tc.testingStruct).Elem()

			fVal := val.Field(tc.fieldIndex)
			fType := typ.Field(tc.fieldIndex)

			err := ParseFieldFromArg(fVal, fType, tc.arg)

			if !tc.expectingErr {
				require.Equal(t, tc.expectedStruct, tc.testingStruct)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParseUint64SliceToString(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint64
		expected string
	}{
		{
			name:     "Test with empty slice",
			input:    []uint64{},
			expected: "",
		},
		{
			name:     "Test with single element",
			input:    []uint64{1},
			expected: "1",
		},
		{
			name:     "Test with multiple elements",
			input:    []uint64{1, 2, 3, 4, 5},
			expected: "1, 2, 3, 4, 5",
		},
		{
			name:     "Test with multiple elements out of order",
			input:    []uint64{9, 1, 2, 3, 4, 5},
			expected: "9, 1, 2, 3, 4, 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseUint64SliceToString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

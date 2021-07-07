package magmasc

import (
	"context"
	"net/url"
	"reflect"
	"testing"
)

func TestMagmaSmartContract_GetAddress(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetAddress(); got != Address {
			t.Errorf("GetAddress() got: %v | want: %v", got, Address)
		}
	})
}

func TestMagmaSmartContract_GetExecutionStats(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetExecutionStats(); !reflect.DeepEqual(got, msc.SmartContractExecutionStats) {
			t.Errorf("GetExecutionStats() got: %#v | want: %#v", got, msc.SmartContractExecutionStats)
		}
	})
}

func TestMagmaSmartContract_GetHandlerStats(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()

	tests := [1]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  nil,
			msc:   msc,
			want:  "type string",
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.GetHandlerStats(test.ctx, test.vals)
			if (err != nil) != test.error {
				t.Errorf("GetHandlerStats() error: %v, want: %v", err, test.error)
				return
			}
			if _, ok := got.(string); !ok {
				t.Errorf("GetHandlerStats() got: %#v | want: %v", got, test.want)
			}
		})
	}
}

func TestMagmaSmartContract_GetName(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		if got := msc.GetName(); got != Name {
			t.Errorf("GetName() got: %v | want: %v", got, Name)
		}
	})
}

func TestMagmaSmartContract_GetRestPoints(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		if got := msc.GetRestPoints(); !reflect.DeepEqual(got, msc.RestHandlers) {
			t.Errorf("GetRestPoints() got: %#v | want: %#v", got, msc.RestHandlers)
		}
	})
}

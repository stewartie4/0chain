package magmasc

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

func Test_DataUsage_Decode(t *testing.T) {
	t.Parallel()

	dataUsage := mockDataUsage()
	blob, _ := json.Marshal(dataUsage)

	tests := [2]struct {
		name    string
		blob    []byte
		want    DataUsage
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: dataUsage,
		},
		{
			name:    "ERR",
			blob:    []byte(":"), // invalid json,
			wantErr: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := DataUsage{}
			if err := got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_Encode(t *testing.T) {
	t.Parallel()

	dataUsage := mockDataUsage()
	blob, _ := json.Marshal(dataUsage)

	tests := [1]struct {
		name      string
		dataUsage DataUsage
		want      []byte
	}{
		{
			name:      "OK",
			dataUsage: dataUsage,
			want:      blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.dataUsage.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_uid(t *testing.T) {
	t.Parallel()

	const (
		scID         = "sc_uid"
		sessionID    = "session_id"
		dataUsageUID = "sc:" + scID + ":datausage:" + sessionID
	)

	dataUsage := DataUsage{SessionID: sessionID}

	tests := [1]struct {
		name      string
		dataUsage DataUsage
		scID      datastore.Key
		want      datastore.Key
	}{
		{
			name:      "OK",
			dataUsage: dataUsage,
			scID:      scID,
			want:      dataUsageUID,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.dataUsage.uid(test.scID); got != test.want {
				t.Errorf("uid() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_validate(t *testing.T) {
	t.Parallel()

	duValid := mockDataUsage()

	duEmptySessionID := duValid
	duEmptySessionID.SessionID = ""

	duZeroDownloadBytes := duValid
	duZeroDownloadBytes.DownloadBytes = 0

	duZeroUploadBytes := duValid
	duZeroUploadBytes.UploadBytes = 0

	duAfterRangeTimestamp := duValid
	duAfterRangeTimestamp.Timestamp += common.Timestamp(providerDataUsageDuration + 1)

	duBeforeRangeTimestamp := duValid
	duBeforeRangeTimestamp.Timestamp -= common.Timestamp(providerDataUsageDuration + 1)

	tests := [6]struct {
		name      string
		dataUsage DataUsage
		want      error
	}{
		{
			name:      "OK",
			dataUsage: duValid,
			want:      nil,
		},
		{
			name:      "EmptySessionID",
			dataUsage: duEmptySessionID,
			want:      errDataUsageInvalid,
		},
		{
			name:      "ZeroDownloadBytes",
			dataUsage: duZeroDownloadBytes,
			want:      errDataUsageInvalid,
		},
		{
			name:      "ZeroUploadBytes",
			dataUsage: duZeroUploadBytes,
			want:      errDataUsageInvalid,
		},
		{
			name:      "AfterRangeTimestamp",
			dataUsage: duAfterRangeTimestamp,
			want:      errDataUsageInvalid,
		},
		{
			name:      "BeforeRangeTimestamp",
			dataUsage: duBeforeRangeTimestamp,
			want:      errDataUsageInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.dataUsage.validate(); !errors.Is(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}

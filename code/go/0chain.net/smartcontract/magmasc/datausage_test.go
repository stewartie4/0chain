package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_DataUsage_Decode(t *testing.T) {
	t.Parallel()

	dataUsage := mockDataUsage()
	blob, err := json.Marshal(dataUsage)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	dataUsageInvalid := mockDataUsage()
	dataUsageInvalid.SessionID = ""
	blobInvalid, err := json.Marshal(dataUsageInvalid)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		blob  []byte
		want  *DataUsage
		error error
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  dataUsage,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &DataUsage{},
			error: errDecodeData,
		},
		{
			name:  "Invalid_ERR",
			blob:  blobInvalid,
			want:  &DataUsage{},
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &DataUsage{}
			if err = got.Decode(test.blob); !errIs(err, test.error) {
				t.Errorf("Decode() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_Encode(t *testing.T) {
	t.Parallel()

	dataUsage := mockDataUsage()
	blob, err := json.Marshal(dataUsage)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name      string
		dataUsage *DataUsage
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
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_validate(t *testing.T) {
	t.Parallel()

	duValid := mockDataUsage()

	duEmptySessionID := mockDataUsage()
	duEmptySessionID.SessionID = ""

	duZeroDownloadBytes := mockDataUsage()
	duZeroDownloadBytes.DownloadBytes = 0

	duZeroUploadBytes := mockDataUsage()
	duZeroUploadBytes.UploadBytes = 0

	duZeroSessionTime := mockDataUsage()
	duZeroSessionTime.SessionTime = 0

	tests := [5]struct {
		name      string
		dataUsage *DataUsage
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
			name:      "ZeroSessionTime",
			dataUsage: duZeroSessionTime,
			want:      errDataUsageInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.dataUsage.validate(); !errIs(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}

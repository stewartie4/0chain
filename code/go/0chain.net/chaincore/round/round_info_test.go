package round

import (
	"reflect"
	"testing"
	"time"
)

func TestInfo_GetKey(t *testing.T) {
	num := int64(5)
	type fields struct {
		TimeStamp                 *time.Time
		Number                    int64
		NotarizedBlocksCount      int8
		ZeroNotarizedBlocksCount  int64
		MultiNotarizedBlocksCount int64
		MissedBlocks              int64
		RollbackCount             int64
		LongestRollbackLength     int8
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name:   "OK",
			fields: fields{Number: num},
			want:   num,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{
				TimeStamp:                 tt.fields.TimeStamp,
				Number:                    tt.fields.Number,
				NotarizedBlocksCount:      tt.fields.NotarizedBlocksCount,
				ZeroNotarizedBlocksCount:  tt.fields.ZeroNotarizedBlocksCount,
				MultiNotarizedBlocksCount: tt.fields.MultiNotarizedBlocksCount,
				MissedBlocks:              tt.fields.MissedBlocks,
				RollbackCount:             tt.fields.RollbackCount,
				LongestRollbackLength:     tt.fields.LongestRollbackLength,
			}
			if got := info.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInfo_GetTime(t *testing.T) {
	ts := time.Now()

	type fields struct {
		TimeStamp                 *time.Time
		Number                    int64
		NotarizedBlocksCount      int8
		ZeroNotarizedBlocksCount  int64
		MultiNotarizedBlocksCount int64
		MissedBlocks              int64
		RollbackCount             int64
		LongestRollbackLength     int8
	}
	tests := []struct {
		name   string
		fields fields
		want   *time.Time
	}{
		{
			name:   "OK",
			fields: fields{TimeStamp: &ts},
			want:   &ts,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{
				TimeStamp:                 tt.fields.TimeStamp,
				Number:                    tt.fields.Number,
				NotarizedBlocksCount:      tt.fields.NotarizedBlocksCount,
				ZeroNotarizedBlocksCount:  tt.fields.ZeroNotarizedBlocksCount,
				MultiNotarizedBlocksCount: tt.fields.MultiNotarizedBlocksCount,
				MissedBlocks:              tt.fields.MissedBlocks,
				RollbackCount:             tt.fields.RollbackCount,
				LongestRollbackLength:     tt.fields.LongestRollbackLength,
			}
			if got := info.GetTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

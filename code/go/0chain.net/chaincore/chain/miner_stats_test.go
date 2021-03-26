package chain

import (
	"reflect"
	"testing"
)

func TestMinerStats_Clone(t *testing.T) {
	ms := &MinerStats{
		GenerationCountByRank:     []int64{1},
		FinalizationCountByRank:   []int64{2},
		VerificationTicketsByRank: []int64{3},
		VerificationFailures:      2,
	}

	type fields struct {
		GenerationCountByRank     []int64
		FinalizationCountByRank   []int64
		VerificationTicketsByRank []int64
		VerificationFailures      int64
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		{
			name: "OK",
			fields: fields{
				GenerationCountByRank:     ms.GenerationCountByRank,
				FinalizationCountByRank:   ms.FinalizationCountByRank,
				VerificationTicketsByRank: ms.VerificationTicketsByRank,
				VerificationFailures:      ms.VerificationFailures,
			},
			want: ms,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MinerStats{
				GenerationCountByRank:     tt.fields.GenerationCountByRank,
				FinalizationCountByRank:   tt.fields.FinalizationCountByRank,
				VerificationTicketsByRank: tt.fields.VerificationTicketsByRank,
				VerificationFailures:      tt.fields.VerificationFailures,
			}
			if got := m.Clone(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Clone() = %v, want %v", got, tt.want)
			}
		})
	}
}

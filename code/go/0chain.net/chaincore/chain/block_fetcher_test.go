package chain

import (
	"github.com/spf13/viper"
	"reflect"
	"testing"
)

func TestNewBlockFetcher(t *testing.T) {
	t.Skip() //TODO
	fm := 1
	viper.Set("async_blocks_fetching.max_simultaneous_from_miners", fm)
	fs := 2
	viper.Set("async_blocks_fetching.max_simultaneous_from_sharders", fs)

	tests := []struct {
		name   string
		wantBf *BlockFetcher
	}{
		{
			name: "OK",
			wantBf: &BlockFetcher{
				fetchBlock: make(chan *blockFetchRequest, fm+fs),
				statq:      make(chan FetchQueueStat),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotBf := NewBlockFetcher(); !reflect.DeepEqual(gotBf, tt.wantBf) {
				t.Errorf("NewBlockFetcher() = %v, want %v", gotBf, tt.wantBf)
			}
		})
	}
}

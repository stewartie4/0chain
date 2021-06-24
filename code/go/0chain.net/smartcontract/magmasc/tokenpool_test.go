package magmasc

import (
	"testing"

	"0chain.net/core/datastore"
)

func Test_tokenPool_uid(t *testing.T) {
	t.Parallel()

	const (
		parentUID    = "parent_uid"
		tokenPoolID  = "token_pool_id"
		tokenPoolUID = parentUID + ":tokenpool:" + tokenPoolID
	)

	pool := tokenPool{}
	pool.ID = tokenPoolID

	tests := [1]struct {
		name      string
		pool      tokenPool
		parentUID datastore.Key
		want      datastore.Key
	}{
		{
			name:      "OK",
			pool:      pool,
			parentUID: parentUID,
			want:      tokenPoolUID,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.pool.uid(test.parentUID); got != test.want {
				t.Errorf("uid() got: %v | want: %v", got, test.want)
			}
		})
	}
}

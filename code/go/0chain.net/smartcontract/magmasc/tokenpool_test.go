package magmasc

import (
	"testing"
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
		name string
		pool tokenPool
	}{
		{
			name: "OK",
			pool: pool,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.pool.uid(parentUID); got != tokenPoolUID {
				t.Errorf("uid() got: %v | want: %v", got, tokenPoolUID)
			}
		})
	}
}

package magmasc

import (
	"testing"
)

func Test_Billing_uid(t *testing.T) {
	t.Parallel()

	const (
		scID      = "sc_uid"
		sessionID = "session_id"
		billUID   = "sc:" + scID + ":datausage:" + sessionID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		bill := newBilling(sessionID)
		if got := bill.uid(scID); got != billUID {
			t.Errorf("uid() got: %v | want: %v", got, billUID)
		}
	})
}

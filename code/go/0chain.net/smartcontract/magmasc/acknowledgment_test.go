package magmasc

import (
	"testing"
)

func Test_Acknowledgment_uid(t *testing.T) {
	t.Parallel()

	const (
		scID      = "sc_uid"
		sessionID = "session_id"
		acknUID   = "sc:" + scID + ":acknowledgment:" + sessionID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		ackn := newAcknowledgment(sessionID)
		if got := ackn.uid(scID); got != acknUID {
			t.Errorf("uid() got: %v | want: %v", got, acknUID)
		}
	})
}

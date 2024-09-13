package scamp

import (
	"context"
	"testing"
)

func TestMockedScampRequester(t *testing.T) {

	mocks := map[string]string{
		"kids.tv.remote.fetch": `{"channel_id":123}`,
		"godfriend.favors.ask": `{"items":[4,5,6]}`,
		"bank.funds.withdraw":  "",
	}

	r := NewMockedScampRequester(mocks)

	for act, res := range mocks {
		mssg := NewMessage()
		mssg.SetAction(act)
		recv, _ := r.MakeJSONRequest(context.Background(), mssg, 10, false)

		if res == "" {
			if len(r.MockErrors()) == 0 {
				t.Errorf("expected errors for empty response")
			}
			continue
		}

		if string(recv.Bytes()) != res {
			t.Errorf("wrong response for action %s", act)
		}
	}

}

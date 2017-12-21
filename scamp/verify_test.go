package scamp

import "testing"

var suspiciousBase64 = []byte(`OSEeu8fWTcq+AliFG3PlZ0eYR8zFWWAdkCwb3XbPE96wvAsiF1W6v2Udg5KoDe7M2d0oQMmpoNeC
ZQWRMBHarz5vHzfTSXXCjvoLfZJVA1FLiJ9RYk8ulFyEJF19nxd2GLArnWjiqsP9RslhFB3BvYnZ
O9IsuyRqWKpa1nl5B68=`)

func TestBase64Decode(t *testing.T) {
	_,err := decodeUnpaddedBase64(suspiciousBase64, false)
	if err != nil {
		t.Errorf("could not decode: `%s`", err)
	}
}
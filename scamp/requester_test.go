package scamp

// func TestRequester(t *testing.T) {
// 	var err error

// 	msg := NewRequestMessage()
// 	msg.SetEnvelope(EnvelopeJSON)

// 	resp, err := MakeJSONRequest("main", "Logger.info", 1, msg)
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// 	if resp == nil || len(resp.Bytes()) == 0 {
// 		t.Fail()
// 	}

// }

// func TestMain(m *testing.M) {
// 	flag.Parse()
// 	Initialize("/etc/SCAMP/soa.conf")
// 	os.Exit(m.Run())
// }

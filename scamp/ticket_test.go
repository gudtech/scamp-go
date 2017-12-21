package scamp

import "testing"
import "bytes"

var signingPubKey = []byte(`-----BEGIN PUBLIC KEY-----
MIICIDANBgkqhkiG9w0BAQEFAAOCAg0AMIICCAKCAgEApSmU3y4DzPhjnpOrdpPs
cIosWJ4zSV8h02b0abLW6nk7cnb5jSwBZKLrryAlF4vs+cF1mtMYjX0QKtEYq2V6
WVDnoXj3BeLYVbhsHuvxYmwXmAkNsSnhMfSCxsck9y6zuNeH0ovzBD90nISIJw+c
VAnUt0dzc7YKjBqThHRAvi8HoGZlzB7Ryb8ePSW+Mfr4jcH3Mio5T0OH3HTavN6Y
zpnohzQo0blwtwEXZOwrNPjQNrSigdPDrtvM32+hLTIJ75Z2NbIRLBjNlwznu7dQ
Asb/AiPTHXihxCRDm+dH70dps5JfT5Zg9LKsPhANk6fNK3e4wdN89ybQsBaswp9h
xzORVD3UiG4LuqP4LMCadjoEazShEiiveeRBgyiFlIldybuPwSq/gUuFveV5Jnqt
txNG6DnJBlIeYhVlA25XDMjxnJ3w6mi/pZyn9ZR9+hFic7Nm1ra7hRUoigfD/lS3
3AsDoRLy0xZqCWGRUbkhlo9VjDxo5znjv870Td1/+fp9QzSaESPfFAUBFcykDXIU
f1nVeKAkmhkEC9/jGF+VpUsuRV3pjjrLMcuI3+IimfWhWK1C56JJakfT3WB6nwY3
A92g4fyVGaWFKfj83tTNL2rzMkfraExPEP+VGesr8b/QMdBlZRR4WEYG3ObD2v/7
jgOS2Ol4gq8/QdNejP5J4wsCAQM=
-----END PUBLIC KEY-----`)

var fullTicketBytes = []byte(`1,3063,21,1438783424,660,1+20+31+32+34+35+36+37+38+39+40+41+42+43+44+46+47+48+50+53+56+59+60+61+62+67+68+69+70+71+75+76+80+81+82+86+87+88+102+104+105+107+109+110+122+124,PcFNyWjoz_iiVMgEe8I3IBfzSlUcqUGtsuN7536PTiBW7KDovIqCaSi_8nZWcj-j1dfbQRA8mftwYUWMhhZ4DD78-BH8MovNVucbmTmf2Wzbx9bsI-dmUADY5Q2ol4qDXG4YQJeyZ6f6F9s_1uxHTH456QcsfNxFWh18ygo5_DVmQQSXCHN7EXM5M-u2DSol9MSROeBolYnHZyE093LgQ2veWQREbrwg5Fcp2VZ6VqIC7yu6f_xYHEvU0-ZsSSRMAMUmhLNhmFM4KDjl8blVgC134z7XfCTDDjCDiynSL6b-D-`)

func TestSplitTicketPayload(t *testing.T) {
	ticketBytes, sigBytes := splitTicketPayload(fullTicketBytes)

	if !bytes.Equal(ticketBytes, []byte("1,3063,21,1438783424,660,1+20+31+32+34+35+36+37+38+39+40+41+42+43+44+46+47+48+50+53+56+59+60+61+62+67+68+69+70+71+75+76+80+81+82+86+87+88+102+104+105+107+109+110+122+124")) {
		t.Errorf("did not extract ticketBytes. got `%s`", ticketBytes)
	}
	if !bytes.Equal(sigBytes, []byte("PcFNyWjoz_iiVMgEe8I3IBfzSlUcqUGtsuN7536PTiBW7KDovIqCaSi_8nZWcj-j1dfbQRA8mftwYUWMhhZ4DD78-BH8MovNVucbmTmf2Wzbx9bsI-dmUADY5Q2ol4qDXG4YQJeyZ6f6F9s_1uxHTH456QcsfNxFWh18ygo5_DVmQQSXCHN7EXM5M-u2DSol9MSROeBolYnHZyE093LgQ2veWQREbrwg5Fcp2VZ6VqIC7yu6f_xYHEvU0-ZsSSRMAMUmhLNhmFM4KDjl8blVgC134z7XfCTDDjCDiynSL6b-D-")) {
		t.Errorf("did not extract the sigBytes")
	}
}

func TestReadTicketNoVerify(t *testing.T) {
	ticket, err := readTicketNoVerify(fullTicketBytes)
	if err != nil {
		t.Errorf("failed to parse: `%s`", err)
	}

	if ticket.Version != 1 {
		t.Errorf("wrong Version")
	}

	if ticket.UserID != 3063 {
		t.Errorf("wrong UserId")
	}

	if ticket.ClientID != 21 {
		t.Errorf("wrong ClientId")
	}

	if ticket.ValidityStart != 1438783424 {
		t.Errorf("wrong ValidityStart")
	}

	if ticket.ValidityEnd != 1438783424+660 {
		t.Errorf("wrong ValidityEnd")
	}
}

func TestSigVerification(t *testing.T) {
	_, err := readTicket(fullTicketBytes, signingPubKey)

	if err == nil {
		t.Errorf("succeeded in parsing ticket. that's unexpected.")
	}
}

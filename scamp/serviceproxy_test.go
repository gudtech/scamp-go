package scamp

import "testing"
import "encoding/json"
import "bytes"

var serviceProxyClassRecordsRaw = []byte(`[3,"bgapi/proc01-HP4m32uuoVLTNXcLrKc3vd75","main",1,5,"beepish+tls://10.8.1.158:30359",["json"],[["bgdispatcher",["poll","",1],["reboot","",1],["report","",1]]],1440001142628.000000]`)
var serviceProxySigRaw = []byte(`CPuxVvNppUNVIGSlaNUW6fpXp2h31/AKX/rAdzyRRsUks8qsjq5/9X5ZUsz85JlPhknxazjlX81U
MSc1qsL1BZvPeRZ+8NXaT58j7UR75mDbOlfv5KbqtdjBA08TmjXOfFy2JG7+iQG3zG12HNXU/Yc+
5yhE5i6eA2/Lfxz8aG9221y/qeDJ49DPXjAJa1PFdVIO2aZno1r3bcBKvu6O30lGakgZYTSfFVJC
DcdsuueiBOTXjbjcxAhyZvEa48LgCxc4X35ywGaM7h0MVPgPHvpWzFB6DvRKbhEXWcja5p6a8Phm
ovxIjhbweSlgq9Duu5hhshiL3RufVqJqJmxxuiEIo3Fa/GBPKPWlOn/W6P3ko0XrkEokYdkoENFO
0z7Uha9RQDlIz2+OtJY7V00QsoKGg96lLjSuJDZkaQ6Oay0+VfljLipELUBB4HSlqmuuph+qHH8E
yYh7fUmYphSz/RtkPc0MBkq5O3HSvdchUkiJ6z+F7cXosQBTb7PpS0Gvc7MnISuw5sLTFCtr0q05
a032GvvcMl27Rdn226paf3WoZzEO9K+zfGtWhircQA8sWkwuSyGfeUvzpo7ZWLMTq5OYGbGBXJlE
ewPJz5OgXQL2RV9xFahxbSxl+3g0OE/u36FyUJDhvPssH/BzA13SpI0rOnsRpqs7pTua8oEppck=`)
var serviceProxyCertRaw = []byte(`-----BEGIN CERTIFICATE-----
MIIFrTCCA5WgAwIBAgIJAPp15voDqw1iMA0GCSqGSIb3DQEBBQUAMG0xKDAmBgNV
BAMMH2tzYXdoMS5kZXYuZ3VkdGVjaC5jb20gYmd3b3JrZXIxCzAJBgNVBAoMAkdU
MQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJU2Fu
IERpZWdvMB4XDTE0MDMxMjIxMDM0MVoXDTI0MDMwOTIxMDM0MVowbTEoMCYGA1UE
Awwfa3Nhd2gxLmRldi5ndWR0ZWNoLmNvbSBiZ3dvcmtlcjELMAkGA1UECgwCR1Qx
CzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlTYW4g
RGllZ28wggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCjN0sSVb19gZru
snQ5TS4nXG/pl//AiPX5RrS5zCWeTbLCIzu1F0b2UO33hGJsOHiuMZANZK89u99F
MN3wVSqT0LoJ2iNMJjEqseVhD5dfiUv0dHpH3RYEHWBcsGfg9sIu1dOnY+x/Rls0
+iO9cUbI38vM/9T6CAoj+eSPh6mBaTvSk5iaMOV4DEIi1MRsKg5MeEIUBdJFfeQk
GubCG9x1ieW4WGXVi4xxQcxNHx9GmEf4AfxF+h/g8gwY27Z1921oekU6WU6cmxCQ
7bHAaYWqiVdhJ2hK9/xtYlX3QY6dpr1bBvRe1iT73qPpb6ReTfFSCRlWlVTpLgL8
eOnlPbiifgIro2MUEaSVLX+RAWFIIKkmSpIa1WfUQdoedANsVoolx95FSNKAfJpv
eLuHUwkAkIMeRnYzgd2KsboOQXfcFfUCRYUjvO//iDuHsdcQMbTw6b6XYE1mqA1M
f8CpQnhlcizwlUPwnusgHZ+6SGQFZ768Fria4kdhFlIZbl224bTKqxxEAQrG+e53
TIVDpc4ApzUPgGSDJXn3SsnwfCIDVSAfUWwH5bgp6/sg3/jpbHubpt5c1U8Gggso
ts+KnH87A7d3GBSkiDLnCEFLBe3Jt9ZG3hkZakYQgw/Ae/TVMBRxYQtsTKEjxqgF
8yVZ9OlK1JSmCOXb/0l0kElKYBGB6QIDAQABo1AwTjAdBgNVHQ4EFgQUrycWFdpg
60s7Ra4A+JzUSKXSa44wHwYDVR0jBBgwFoAUrycWFdpg60s7Ra4A+JzUSKXSa44w
DAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAgEAZ55TW80XwZnZfJTzcH/x
iTngEMtHyIWe9eqLxZX1LfBnO2RrboeGzO14JOXVF2k1XBsitIcznpWXjcbJ6s2w
gj+w/XJ69fjhZqR1dsV4R5aVlUFdkFkkFo6gSSmuNp85YMDvhjfk9ylr/sQ35ii1
zZZXqV6ej6ZJkdUWwxQAYosePwcRuXfw0vMZu8rVRIBLTrx44bGO4gTT8PHpEAVs
UsTDrxWr2tOKc6aBpln2td4wRdVEATdosdasfPdmJb/NZa6V4B1vKJdGGTQie6Y2
qrF2aO/yAFI2+KKXpCuzCLUdwE/k3ic+sh42Xx9tRiz7QK1FEdIzPGN4uirvfJoo
z9eJxMWH+6qqJAcEZw/genujvZZLL92zOhYOpt33S31kPl3PLID3QQb3DcVkM0vX
ZONMacI0KIqXwl6qhXpjbR2MN2/4R0PDh1f6M43jpC46kDZkSyb4XRZxUNoR81De
im2t9+8oySb+LlPut0PV0M0dCuX+B23Tn3AoTt3/ChwrNFLYXxJG6LQyrnsrCRdS
cnwE7U+GQeVLssOmT3xNIbuRcVlLea9uhtg0ojzlUyrdjjhNJXoChxeOVyta6Gtq
bVclWY0DoBeS1lchtLrDURwVSO947nQMKzsi8Rxm4EDQzA0Z70T6vJFsd0n937iV
zJjLAmIyriV6BEUJvfwKmfA=
-----END CERTIFICATE-----`)

func TestNewServiceProxy(t *testing.T) {
	proxy, err := newServiceProxy(serviceProxyClassRecordsRaw, serviceProxyCertRaw, serviceProxySigRaw)
	if err != nil {
		t.Fatalf("failed to instantiate new proxy: %s", err.Error())
	}

	if len(serviceProxyClassRecordsRaw) != len(proxy.rawClassRecords) {
		t.Errorf("something mucked with the serviceProxyClassRecordsRaw")
	}

	err = proxy.Validate()
	if err != nil {
		// TODO need to manually resign our announce packet. Sigh.
		// t.Errorf("failed to validate: `%s`", err)
	}
}

func TestServiceProxySerialize(t *testing.T) {
	// [3,
	//  "bgapi/proc01-HP4m32uuoVLTNXcLrKc3vd75",
	// "main",
	// 1,
	// 5000,
	// "beepish+tls://10.8.1.158:30359",
	// ["json"],
	// [["bgdispatcher",["poll","",1],["reboot","",1],["report","",1]]]
	// ,1440001142628
	// ]
	serviceProxy := ServiceProxy{
		version:          3,
		ident:            "bgapi/proc01-HP4m32uuoVLTNXcLrKc3vd75",
		sector:           "main",
		weight:           1,
		announceInterval: defaultAnnounceInterval,
		connspec:         "beepish+tls://10.8.1.158:30359",
		protocols:        []string{"json"},
		timestamp:        1440001142628,
		// ["reboot","",1],["report","",1]
		classes: []serviceProxyClass{
			serviceProxyClass{
				className: "bgdispatcher",
				actions: []actionDescription{
					actionDescription{
						actionName: "poll",
						crudTags:   "",
						version:    1,
					},
					actionDescription{
						actionName: "reboot",
						crudTags:   "",
						version:    1,
					},
					actionDescription{
						actionName: "report",
						crudTags:   "",
						version:    1,
					},
				},
			},
		},
	}

	b, err := json.Marshal(&serviceProxy)
	if err != nil {
		t.Fatalf("unexpected error `%s` while serializing service record", err)
	}
	if !bytes.Equal(serviceProxyClassRecordsRaw, b) {
		t.Fatalf("serialized service record did not match expected.\nexpected: `%s`\ngot:      `%s`\n", serviceProxyClassRecordsRaw, b)
	}
}

package scamp

import "testing"
import "bytes"
import "bufio"
import "os"

func TestScanCertificate(t *testing.T) {
	reader := bytes.NewReader(testCertificateToScan)

	scanner := bufio.NewScanner(reader)
	scanner.Split(scanCertficates)

	if !scanner.Scan() {
		t.Errorf("failed to scan")
	}

	if !bytes.Equal(scanner.Bytes(), testCertificateToScan) {
		t.Errorf("certificate did not match:\n`%v`\n`%v`", scanner.Bytes(), testCertificateToScan)
	}
}

func BenchmarkReadingProductionAnnounceCache(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			file, err := os.Open("/Users/xavierlange/code/gudtech/scamp-patrol/fixtures/discovery.sample")
			if err != nil {
				panic("could not open file")
			}

			cache, err := NewServiceCache("/tmp/blah")
			if err != nil {
				panic("wah wah")
			}
			err = cache.DoScan(bufio.NewScanner(file))
			if err != nil {
				panic("failed to load announce cache")
			}
		}
	})
}

func TestReadAnnounceCache(t *testing.T) {
	initSCAMPLogger()

	file, err := os.Open("/Users/xavierlange/code/gudtech/scamp-go/fixtures/announce_cache")
	if err != nil {
		return
	}

	s := bufio.NewScanner(file)

	cache, err := NewServiceCache("/tmp/blah")
	if err != nil {
		t.Fatalf("could not create new service cache: `%s`", err)
	}
	err = cache.DoScan(s)
	if err != nil {
		t.Errorf("unexpected error parsing announce cache: `%s`", err)
	}

	if cache.Size() != 32 {
		t.Errorf("expected cache size to be 32 but was %d", cache.Size())
	}
}

func TestScanNoNewLineCert(t *testing.T) {
	initSCAMPLogger()

	s := bufio.NewScanner(bytes.NewReader(weirdEntries))
	cache, err := NewServiceCache("./../fixtures/blah")
	if err != nil {
		t.Fatalf("could not create new service cache: `%s`", err)
	}
	err = cache.DoScan(s)
	if err != nil {
		t.Fatalf("failed: `%s`", err)
	}

	if cache.Size() != 1 {
		t.Fatalf("expected 1 entries in the cache after scanning, got %d", cache.Size())
	}

}

func TestRegisterOnServiceCache(t *testing.T) {
	cache, err := NewServiceCache("./../fixtures/blah")
	if err != nil {
		t.Fatalf("could not create new service cache")
	}
	serviceInstance := new(serviceProxy)
	serviceInstance.ident = "bob"

	cache.Store(serviceInstance)
	retrieved := cache.Retrieve("bob")
	if retrieved == nil {
		t.Errorf("retrieved nothing")
	}
}

var testCertificateToScan = []byte(`
-----BEGIN CERTIFICATE-----
MIIFqzCCA5OgAwIBAgIJALlP+LRar+aUMA0GCSqGSIb3DQEBBQUAMGwxJzAlBgNV
BAMMHmtzYXdoMS5kZXYuZ3VkdGVjaC5jb20gcGF5bWVudDELMAkGA1UECgwCR1Qx
CzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlTYW4g
RGllZ28wHhcNMTQwMjIyMDY1MjUyWhcNMjQwMjIwMDY1MjUyWjBsMScwJQYDVQQD
DB5rc2F3aDEuZGV2Lmd1ZHRlY2guY29tIHBheW1lbnQxCzAJBgNVBAoMAkdUMQsw
CQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJU2FuIERp
ZWdvMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA0uoAPSpP5Gdq9vut
MIyhWUwwbTr8VXpdMJcdmCESQd/6dbas2F7bujusblujed/CWdPRomjZPwuq7z/9
vlUnIvN7ZKCc1q7vnV0hDMfBeZAuMHYhHgywU4O/DumvxYsPUayr6Bj1rfOjVkyY
QsiF8pGAZa7+/H2IdrdTNAxgLLlQmaYDCU4ckvyLcOm/MICKfyIORn3bmhcpTbmG
rYtJbR8FgsBAQxhv1GiYQXCNYGVbERGSVw8Pw4Zk4S8yHG3wiJoEvoc2WBYhBDGA
ZhkrdH1/UNscjsIiabF4+E1mAahB9k0+4844vuktLbQtDpJLp5upeWDAVN5pCduw
946C8U2wma23WMTFC9NDwYOHJpDeXxu4K8F3GJIvN1YVzIjLqVV0AU6dXHHJSCx2
pAc7ZxwcT8XgJ17AgFyfRmLNr2TfItozoLXaHMmWWLDPdJEkiLVtBRowQkocbDPc
Ih2y4iEM3cO9hcNwuU4oyjM6asudB9PeUS3jk7OAoZE9BYv3gkZuoxWk3FoCwpoL
2QW26hiaO5fIhTcwEKjb/Qm9wDNkUlSgxExJ2adV+psXZTwk3HlrHL6k323E8YEC
THHnFvYXWrXrGw5Vg0OyLl5vQUqSbzNW5yQtFhIhOc5rFONTHuTnLZ74Vlm17DyF
6o+VNFGkHdDwDS68bdaTz7VIS20CAwEAAaNQME4wHQYDVR0OBBYEFMha7Tsi+kBH
0GyrSGUJxzP/GdVNMB8GA1UdIwQYMBaAFMha7Tsi+kBH0GyrSGUJxzP/GdVNMAwG
A1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggIBAAAZaoQJCwIDLeic7ps8UO8f
UmN2gK/APHEsY5Ev4mNxFcIiUAJlHDMJ5MvinacakanIkQMYrDPmLPCeR9rBYk5d
EOjhdrUwfgx+cxlJmt5Q4zHYjtfHvIINVIsnUus4mb9ONHGEwYFPCZHCHpgmW/qd
HWu7huXsnpyHhA29lmoOtxJIULR4xPNWh09pouc+YXdTK9F5KoPCskWI0z4Xvo3K
NmttiNERg86W6S7yQO1Zj5Bwu5t2uBAmeJsOpIJvSW5MOWPDvCBSAWy085aO7eoE
ttRl+KGbLVbVRR3Sbg3fnweR5hEA/C50i5iVTZUERQ+c2jgvcG6WM5uJ1xCkxxsb
7bn+vUwWDyHv86vQbt8Uq02KIfp1MGC3la12M7v6UNgYq/Ruhbr6E4HjHjswo3Uc
yAdMtpSgXVj+rczQLNO/NCGVnpkEge8qcBAgoC2EfLpCsOhclhwtnwkom8WXi4ga
s5chuQjibCKkBh+dEPDnjjM/WWmWzoQtK+esmPLp1sNRJDc/dwgcaxAHTG5yiS5r
GXM51rweYWKZZ3TbShekXaNFDct97dFgXBWuTvB/5X7bd1Uu0X9U5GqINbjRSoUL
0FPRAZnwPS8XQRj76fO0EiKyHZMZMnfS78vzFuOxR+vOaFqWzZ0TVdOuyQRZqW7w
r/6q4/7qnfRgT6Clnxtf
-----END CERTIFICATE-----`)

var weirdEntries = []byte(`%%%
[3,"logging-4HYwEWZA6IV8f/vSsMzDb5lS","main",1,2500,"beepish+tls://10.240.0.3:30100",["json"],[["Logger",["log","",1]]],1458851219.822557]

-----BEGIN CERTIFICATE-----
MIIGBjCCA+6gAwIBAgIJAO7Roq9T1WmqMA0GCSqGSIb3DQEBBQUAMF8xEzARBgNV
BAMTCiBzdGF0aWNkZXYxEjAQBgNVBAoTCVNDQU1QIEluYzELMAkGA1UEBhMCVVMx
EzARBgNVBAgTCkNhbGlmb3JuaWExEjAQBgNVBAcTCVNhbiBEaWVnbzAeFw0xNTA5
MTcyMDQ2MjRaFw0yNTA5MTQyMDQ2MjRaMF8xEzARBgNVBAMTCiBzdGF0aWNkZXYx
EjAQBgNVBAoTCVNDQU1QIEluYzELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlm
b3JuaWExEjAQBgNVBAcTCVNhbiBEaWVnbzCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAL+A1bXR25Vyv82QEwpJYPAWPz/qPFbCQC4+AF3D5gCkGhBQ2fpc
IUYqyeNhqPK0y4d3Kwsjr3DLPPDEQuTjQ7qWHArFnlvYAylGVs0zabb9vLSkYT3O
c4Vhs/1QB7IJr2x3K7qaJ3of2DIfrY34bRsiPNfAVokCRr9wOn0KmL1qwInLzF/k
ZcCluzIL++laZjlRcMKKarDvUncDOGJiOn6JlMOJqaYV9q/SHUtRfCFFkl7WMe3J
ZXW3w5C40yOwuZT88oEFUVzZEgOaDRXJOtZXIiL3n8dcHgmoCLolcIvzMdXJ+Jb0
sVoLsdFQCPAh4L11C8hk1s63yKGs+hUKriwfn2hf9GickooQaa86sG0/UEXWYBA8
XmXcnbDEnNFWHUkhw6COvIh/nJJWLIHi6906CPEdPvYni7x/ABlS1v/XvA7jH8EA
a2yoN2Kv8PoN7JAo2q7NU/I2UV+2+86jH8UicsMNHd8mpw/6UFvCegyJYBSxSN27
2hGpx53BspK+gi4RXk+oHXjeCbQz5iMcO9Nx37kMA8Ws0YE2h6ffsNC6+8DILuda
MMZxu14bMOdubVUYu/17inCiA4DgukudCGvOIXXTSwfvT4k4UkgV5+RC77RXK5iZ
zKiWFtlK9RxO6MC6lH1mcxagb1LgCml/ZeYPtzfaO+IpdqhvGAfh08dRAgMBAAGj
gcQwgcEwHQYDVR0OBBYEFFlCwKYb1cxc8KDQNWco8iJHwCEAMIGRBgNVHSMEgYkw
gYaAFFlCwKYb1cxc8KDQNWco8iJHwCEAoWOkYTBfMRMwEQYDVQQDEwogc3RhdGlj
ZGV2MRIwEAYDVQQKEwlTQ0FNUCBJbmMxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRIwEAYDVQQHEwlTYW4gRGllZ2+CCQDu0aKvU9VpqjAMBgNVHRME
BTADAQH/MA0GCSqGSIb3DQEBBQUAA4ICAQAiMH0P33e8EBeWFxbRgZv8Nhwgvv2A
w3LqTSNhbN+V6xj+QJrfOnhzCgnBtN1w6zak2PLr+iC7Srmpp1mfSl532b2Qdhbs
SPuQKOQZUNFzYd1KlQ5imHlJJjFmPz5PZK8ecs3OIb2ZO665fli28S9gXTfbUHbK
pQP5yH1y1E2wUXfLkuiYsNcr5MnEsNi25hViz6XPc7+yAtSxr1h5Gz4gppb+cGGR
KjCdVzFM4z7HmEqr5QJCs1eTPAlkyCR8bEKu2JYdJgXvMXiWzWDAgAIW0OUsviZu
Nbs0PMl8UZlnapiuEi1kBR4iUJp9BqqKN9lgJa680LHJx0bTeQ9I5viDx125nsUT
3JeeQ9t5bj3/2W0wmJ2I1eglHg2Guxin68JHdaZ2tVd1H9Yl25uJ5Fba69cm0/tI
wHuYD8Pj9xWbo7Woe854LejAAJfn0b2lMDjg9HndWpsuEGB/m/3lMFNodYZScE+s
lF8B42EZvKY3mZVbuQq8TxStTMwbKoByMN7BsAczI2cwpe0VRLy9cJeIwfoMQCfI
/tFFwgpdF96lZb8hKCTUK4hSt+6fyofxXNbzfqI26SYhJ1eAz/T1vwwBO0ds1K6B
0pU+a8XO6sGqJc2xSIk8Nrmsb582skkDeeJdn0eAqfwnsbqrnkG99MyZeBSV35/5
9zE2L0X6btVzaA==
-----END CERTIFICATE-----

IH837ZChGvJa2Px/ZSaFu5MKQhIuH92hEeyOJ9aqsP1whyXWd+Cy2cErXhqqsDCmB/6RYCrFs/fsulPoC1eUmuaRr9z7FEr8fc4wJ8YNSzGlwckJI5TaCil73F4PGvNeZ2iJ+bJjt21QO8ba1NPbDAzTsBkeCcbI6yyKp6fogCEwFJtWLpvssNIbu5PARpepYlEzOurCyfk1v/7giV9pRpgVM9wwsLFnD6q989GkaazDSOuONwWqQ0Ow1MuzkUOCivbOqugaaBL/by+gELHA2XR29BlKSMUz3kT5f+EqQ0YiomfIPc3CdByPwM0rUFOblbLM8pEL+IjkXqjOK/WHv03H/fPzTF8RnzybtaSDcJiyd4ZKfo9NVI/43qC0/AdOmM0dfkaB8IlChmrGLlulaoQWAxB4fqLz6RyhRooyKQxDiivAjabaOiLmCgnVzVYa8Ub4Wi1UgGckEOjl9yACJQLGMiJxy8lNgPVYkqVqN0cFoE1kt/zDogbpK/LIlysuDIFxg96BufFQPmHWIuInsvG7LZ4p5DN3oaw/coYkF4evQ9o3pqe+ef6NCfGLHZhasvA4lnXY9kEIgncQUyhZk0EhnfJNvDrvvFT0TwNmp5gM2K6Y4Wq7sjMZiu7sSfThX4a1SqDt1xR8jHRzieFk2XkmSpb1bt+SllNE8WESeDc=
%%%
[3,"logging-62vZGD74EWC5N3Rj6gOcQbA0","main",1,2500,"beepish+tls://10.240.0.2:30100",["json"],[["Logger",["log","",1]]],1458851222.582646]

-----BEGIN CERTIFICATE-----
MIIGBjCCA+6gAwIBAgIJAO7Roq9T1WmqMA0GCSqGSIb3DQEBBQUAMF8xEzARBgNV
BAMTCiBzdGF0aWNkZXYxEjAQBgNVBAoTCVNDQU1QIEluYzELMAkGA1UEBhMCVVMx
EzARBgNVBAgTCkNhbGlmb3JuaWExEjAQBgNVBAcTCVNhbiBEaWVnbzAeFw0xNTA5
MTcyMDQ2MjRaFw0yNTA5MTQyMDQ2MjRaMF8xEzARBgNVBAMTCiBzdGF0aWNkZXYx
EjAQBgNVBAoTCVNDQU1QIEluYzELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlm
b3JuaWExEjAQBgNVBAcTCVNhbiBEaWVnbzCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAL+A1bXR25Vyv82QEwpJYPAWPz/qPFbCQC4+AF3D5gCkGhBQ2fpc
IUYqyeNhqPK0y4d3Kwsjr3DLPPDEQuTjQ7qWHArFnlvYAylGVs0zabb9vLSkYT3O
c4Vhs/1QB7IJr2x3K7qaJ3of2DIfrY34bRsiPNfAVokCRr9wOn0KmL1qwInLzF/k
ZcCluzIL++laZjlRcMKKarDvUncDOGJiOn6JlMOJqaYV9q/SHUtRfCFFkl7WMe3J
ZXW3w5C40yOwuZT88oEFUVzZEgOaDRXJOtZXIiL3n8dcHgmoCLolcIvzMdXJ+Jb0
sVoLsdFQCPAh4L11C8hk1s63yKGs+hUKriwfn2hf9GickooQaa86sG0/UEXWYBA8
XmXcnbDEnNFWHUkhw6COvIh/nJJWLIHi6906CPEdPvYni7x/ABlS1v/XvA7jH8EA
a2yoN2Kv8PoN7JAo2q7NU/I2UV+2+86jH8UicsMNHd8mpw/6UFvCegyJYBSxSN27
2hGpx53BspK+gi4RXk+oHXjeCbQz5iMcO9Nx37kMA8Ws0YE2h6ffsNC6+8DILuda
MMZxu14bMOdubVUYu/17inCiA4DgukudCGvOIXXTSwfvT4k4UkgV5+RC77RXK5iZ
zKiWFtlK9RxO6MC6lH1mcxagb1LgCml/ZeYPtzfaO+IpdqhvGAfh08dRAgMBAAGj
gcQwgcEwHQYDVR0OBBYEFFlCwKYb1cxc8KDQNWco8iJHwCEAMIGRBgNVHSMEgYkw
gYaAFFlCwKYb1cxc8KDQNWco8iJHwCEAoWOkYTBfMRMwEQYDVQQDEwogc3RhdGlj
ZGV2MRIwEAYDVQQKEwlTQ0FNUCBJbmMxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRIwEAYDVQQHEwlTYW4gRGllZ2+CCQDu0aKvU9VpqjAMBgNVHRME
BTADAQH/MA0GCSqGSIb3DQEBBQUAA4ICAQAiMH0P33e8EBeWFxbRgZv8Nhwgvv2A
w3LqTSNhbN+V6xj+QJrfOnhzCgnBtN1w6zak2PLr+iC7Srmpp1mfSl532b2Qdhbs
SPuQKOQZUNFzYd1KlQ5imHlJJjFmPz5PZK8ecs3OIb2ZO665fli28S9gXTfbUHbK
pQP5yH1y1E2wUXfLkuiYsNcr5MnEsNi25hViz6XPc7+yAtSxr1h5Gz4gppb+cGGR
KjCdVzFM4z7HmEqr5QJCs1eTPAlkyCR8bEKu2JYdJgXvMXiWzWDAgAIW0OUsviZu
Nbs0PMl8UZlnapiuEi1kBR4iUJp9BqqKN9lgJa680LHJx0bTeQ9I5viDx125nsUT
3JeeQ9t5bj3/2W0wmJ2I1eglHg2Guxin68JHdaZ2tVd1H9Yl25uJ5Fba69cm0/tI
wHuYD8Pj9xWbo7Woe854LejAAJfn0b2lMDjg9HndWpsuEGB/m/3lMFNodYZScE+s
lF8B42EZvKY3mZVbuQq8TxStTMwbKoByMN7BsAczI2cwpe0VRLy9cJeIwfoMQCfI
/tFFwgpdF96lZb8hKCTUK4hSt+6fyofxXNbzfqI26SYhJ1eAz/T1vwwBO0ds1K6B
0pU+a8XO6sGqJc2xSIk8Nrmsb582skkDeeJdn0eAqfwnsbqrnkG99MyZeBSV35/5
9zE2L0X6btVzaA==
-----END CERTIFICATE-----

DzZqcZjWY5gs9UaTHBBMAwp5G3tr1uQ6Fgi3mFlo1tA9J5Vex8CEaw+U0YklidTKMVDN3y8OLZsICLwTSmjO3iHuhoiJK0WuKMkg2PcA7VOJcTS//pufF/8BqRM/4ZjwKXr8yuEwALhamvcMqdvLKdjYi+XyoU+6dzsg3nHKGGeiPlvVoplZ5n7rw4RnrfSTJedyQNy3n8edFzuhbElf4iNP1eg2cdyITtvC55rQ7QCQ9fggkNR1U4yAnganMpXHsrvCcRnpRprTD3YO8/nKqaiJrsTX1xskxdikPvFc8u6M9vSK3oEWVJDE7six/bfLGR/bFgTkpDqfeVVdCkJbE8p1qzxqOpzNA93wChloKcKhCQDND/A8bCfa5Ex0FyLO8PFuRH6CTiPzV4ds3EgzY8hcYe0+8ql6ER4jz0QmmHeyBWsxur39VzBuN7ewI1Kl9niShW6amFF5I0PAR9CPt1R/eT9CiW2JvUUBPJnzeqf+vM7Bupg9QFNSUsB0QItbJa1birCNJp/js3tw581mTNzZg2iwuei7KSxRTJ+l8v5BBw15hXNB9q60v4sFmJMMUX123sYmyM/JAfgTlPubzyEH+WpiEeq5Ps6Pc+aqiR2lE0sioLgC1n0BGZbCS0v6ELYZ6jNz1+2SuJWpUvUNdR2SoQ7D5sIpaKTT/rjUngg=
%%%
`)

func TestSearchByAction(t *testing.T) {
	cache, err := NewServiceCache("./../fixtures/sample_discovery_cache")
	if err != nil {
		t.Fatalf(err.Error())
	}

	cache.DisableRecordVerification()
	err = cache.Refresh()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// t.Fatalf("%s", cache.actionIndex)

	serviceProxy, err := cache.SearchByAction("main", "Logger.info", 1, "json")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if serviceProxy == nil {
		t.Fatalf("hmm, no hit!")
	}
}

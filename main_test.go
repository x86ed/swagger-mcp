package main

import (
	"flag"
	"os"
	"testing"
)

func Test_getSseUrlAddr(t *testing.T) {
	cases := []struct {
		sseUrl, sseAddr, wantUrl, wantAddr string
		shouldPanic                        bool
	}{
		{"", ":8080", "http://localhost:8080", ":8080", false},
		{"", "127.0.0.1:9000", "http://127.0.0.1:9000", "127.0.0.1:9000", false},
		{"http://foo.com:1234", "", "http://foo.com:1234", "foo.com:1234", false},
		{"https://bar.com", "", "https://bar.com", "bar.com:443", false},
		{"http://baz.com", "", "http://baz.com", "baz.com:80", false},
		{"", "", "", "", true},
		{"", "badaddr", "", "", true},
		{"ftp://bad.com", "", "", "", true},
	}
	for _, c := range cases {
		gotUrl, gotAddr := "", ""
		func() {
			defer func() {
				if r := recover(); r != nil {
					if !c.shouldPanic {
						t.Errorf("getSseUrlAddr(%q, %q) panicked unexpectedly: %v", c.sseUrl, c.sseAddr, r)
					}
				}
			}()
			gotUrl, gotAddr = getSseUrlAddr(c.sseUrl, c.sseAddr)
		}()
		if !c.shouldPanic && (gotUrl != c.wantUrl || gotAddr != c.wantAddr) {
			t.Errorf("getSseUrlAddr(%q, %q) = (%q, %q), want (%q, %q)", c.sseUrl, c.sseAddr, gotUrl, gotAddr, c.wantUrl, c.wantAddr)
		}
	}
}

func Test_runMain_missingSpecUrl(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}
	// Reset flags for test
	for _, f := range []string{"specUrl"} {
		flag := flag.Lookup(f)
		if flag != nil {
			flag.Value.Set("")
		}
	}
	err := runMain()
	if err == nil || err.Error() != "Please provide the Swagger JSON URL or file path using the --specUrl flag" {
		t.Errorf("Expected error for missing specUrl, got: %v", err)
	}
}

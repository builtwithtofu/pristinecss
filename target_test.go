package pristinecss

import "testing"

func TestBrowserIDs(t *testing.T) {
	ids := map[BrowserID]string{
		Chrome: "chrome", Edge: "edge", Firefox: "firefox", Safari: "safari", IOSSafari: "ios_saf",
		ChromeAndroid: "and_chr", FirefoxAndroid: "and_ff", Samsung: "samsung", Opera: "opera", OperaAndroid: "op_mob", IE: "ie",
	}
	if len(ids) != len(capBrowsers) {
		t.Fatalf("browser constants=%d, mapped browsers=%d", len(ids), len(capBrowsers))
	}
	for id, want := range ids {
		if string(id) != want {
			t.Errorf("browser ID=%q, want %q", id, want)
		}
		if _, ok := capBrowsers[id]; !ok {
			t.Errorf("browser constant %q missing from mapped data", id)
		}
	}
}

func TestBrowserAt(t *testing.T) {
	for _, tc := range []struct {
		id      BrowserID
		version string
	}{
		{Chrome, "49"}, {IOSSafari, "15.2-15.3"}, {Firefox, "115.0esr"}, {Safari, "TP"},
		{"unknown", "unrecognised"},
	} {
		got := tc.id.At(tc.version)
		want := Target{Browser: tc.id, Version: tc.version, Caps: Browser(tc.id, tc.version)}
		if got != want {
			t.Errorf("%s.At(%q)=%+v, want %+v", tc.id, tc.version, got, want)
		}
	}
}

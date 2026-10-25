package pristinecss

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestTargetCorpusUnchanged(t *testing.T) {
	h := sha256.New()
	for _, path := range corpusFiles(t) {
		for _, fidelity := range []Fidelity{0, Identical} {
			bin, _ := ToBinary(readCSS(t, path), Options{Lower: lowerTargets(), MinFidelity: fidelity})
			fmt.Fprintf(h, "%d:", len(bin))
			h.Write(bin)
		}
	}
	// Captured before Lower changed from []Caps to []Target. The complete binary
	// covers variant count, ordering, requirements, and minified/pretty outputs.
	const want = "ec35795d05d928f7b1b302853a17b5f9f0d536cc36e6922c3534b1f643db9137"
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != want {
		t.Fatalf("corpus binaries changed: %s, want %s", got, want)
	}
}

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

func TestTargetDiagnosticsKeepIdentity(t *testing.T) {
	old, sameCaps, p3 := Chrome.At("49"), Chrome.At("49.0"), Safari.At("12")
	if old.Caps != sameCaps.Caps {
		t.Fatal("equivalent versions should have the same capabilities")
	}
	targets := []Target{old, sameCaps, {Caps: old.Caps}, p3}
	const src = "a{inset:var(--edges);color:oklch(.8 .35 30)}"
	bin, report := ToBinary([]byte(src), Options{Lower: append(targets, old)})
	if report.Variants != 3 || len(report.Gaps) != len(targets) || len(report.Mappings) != len(targets) {
		t.Fatalf("targets did not share variants while retaining diagnostics: %+v", report)
	}
	for i, target := range targets {
		if report.Gaps[i].Target != target || report.Mappings[i].Target != target {
			t.Errorf("diagnostics lost target %+v: gap=%+v mapping=%+v", target, report.Gaps[i], report.Mappings[i])
		}
	}
	capOnly, _ := ToBinary([]byte(src), Options{Lower: []Target{{Caps: old.Caps}, {Caps: p3.Caps}}})
	if !bytes.Equal(bin, capOnly) {
		t.Fatal("browser identity changed stored outputs or selection data")
	}
}

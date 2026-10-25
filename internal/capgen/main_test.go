package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSupportIntervals(t *testing.T) {
	// The last two full spans merge; aliases, prefixes, flags and partial or
	// unknown introductions must never fill the removal gap.
	const data = `{"css":{"feature":{"__compat":{"support":{"safari":[
		{"version_added":"17.4","version_removed":"26"},
		{"version_added":"26","version_removed":"26.4","partial_implementation":true},
		{"version_added":"26.4","version_removed":"27"},
		{"version_added":"27"},
		{"version_added":"1","prefix":"-webkit-"},
		{"version_added":"1","alternative_name":"old-feature"},
		{"version_added":"1","flags":[{"name":"flag"}]},
		{"version_added":"preview"},
		{"version_added":true}
	]}}}}}`
	var bcd map[string]any
	if err := json.Unmarshal([]byte(data), &bcd); err != nil {
		t.Fatal(err)
	}
	got, err := support(bcd, "css.feature", "safari")
	if err != nil {
		t.Fatal(err)
	}
	want := []interval{{0x11000400000000, 0x1a000000000000}, {0x1a000400000000, math.MaxUint64}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("support = %#v, want %#v", got, want)
	}
	if got, err := support(bcd, "css.feature", "ie"); err != nil || len(got) != 0 {
		t.Fatalf("missing browser = %v, %v", got, err)
	}
	if _, err := support(bcd, "css.missing", "safari"); err == nil {
		t.Fatal("missing BCD key accepted")
	}
}

func TestBoundedSupport(t *testing.T) {
	var bcd map[string]any
	if err := json.Unmarshal([]byte(`{"css":{"feature":{"__compat":{"support":{"chrome":{"version_added":"≤37","version_removed":"40"}}}}}}`), &bcd); err != nil {
		t.Fatal(err)
	}
	got, err := support(bcd, "css.feature", "chrome")
	want := []interval{{0x25000000000000, 0x28000000000000}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("bounded support = %v, %v; want %v", got, err, want)
	}
}

func TestBaselineLow(t *testing.T) {
	var f feature
	const data = `{"status":{"baseline_low_date":"2022-06-01","by_compat_key":{
		"css.a":{"baseline_low_date":"2019-05-01"},
		"css.b":{"baseline_low_date":"2021-02-03"},
		"css.bounded":{"baseline_low_date":"≤2020-01-15"},
		"css.none":{}
	}}}`
	if err := json.Unmarshal([]byte(data), &f); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		keys []string
		want string
	}{
		{"latest key wins", []string{"css.a", "css.b"}, "2021-02-03"},
		{"any key without a date gives none", []string{"css.a", "css.none"}, ""},
		{"absent key gives none", []string{"css.a", "css.missing"}, ""},
		{"bounded date keeps its date without the ≤", []string{"css.bounded"}, "2020-01-15"},
		{"bounded date loses to a later key", []string{"css.bounded", "css.b"}, "2021-02-03"},
		{"whole feature row uses the feature date", []string{"*"}, "2022-06-01"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := baselineLow(f, tt.keys); got != tt.want {
				t.Fatalf("baselineLow(%q) = %q, want %q", tt.keys, got, tt.want)
			}
		})
	}
}

func TestIntersect(t *testing.T) {
	for _, tt := range []struct{ a, b, want []interval }{
		{nil, []interval{{1, 10}}, nil},
		{[]interval{{1, 10}}, nil, nil},
		{[]interval{{1, 10}}, []interval{{10, 20}}, nil},
		{[]interval{{1, 10}, {20, 40}}, []interval{{5, 25}}, []interval{{5, 10}, {20, 25}}},
		{[]interval{{1, 10}}, []interval{{2, 3}, {5, 7}}, []interval{{2, 3}, {5, 7}}},
	} {
		if got := intersect(tt.a, tt.b); !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("intersect(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPinnedRegeneration(t *testing.T) {
	data := os.Getenv("PCSS_CAPS_DATA")
	if data == "" {
		t.Skip("set PCSS_CAPS_DATA to pinned extracted releases (fetch.sh)")
	}
	got, err := generate(data, "capabilities.tsv")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "caps_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("caps_gen.go differs from pinned regeneration")
	}
	again, err := generate(data, "capabilities.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, again) {
		t.Fatal("repeated generation is nondeterministic")
	}
	t.Logf("two regenerations byte-identical to caps_gen.go (%d bytes)", len(got))
}

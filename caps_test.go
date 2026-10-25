package pristinecss

import (
	"fmt"
	"strings"
	"testing"
)

func TestCapsBits(t *testing.T) {
	for _, bits := range [][4]uint64{
		{}, {1, 2, 4, 8}, {0x8000000000000000, 1, 0x8000000000000000, 1},
		{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)},
	} {
		c := CapsFromBits(bits)
		if c.Bits() != bits || CapsFromBits(c.Bits()) != c {
			t.Fatalf("round trip %x", bits)
		}
		m := map[Caps]string{c: "found"}
		if m[CapsFromBits(bits)] != "found" {
			t.Fatal("not usable as a map key")
		}
		for i := range 256 {
			want := bits[i/64]&(uint64(1)<<(i%64)) != 0
			if c.Has(Cap(i)) != want {
				t.Fatalf("Has(%d) for %x", i, bits)
			}
		}
		for _, x := range []Cap{256, 65535} {
			if c.Has(x) {
				t.Fatalf("Has(%d) outside the set", x)
			}
		}
	}
	if CapsFromBits([4]uint64{1}) == CapsFromBits([4]uint64{2}) {
		t.Fatal("different bits compare equal")
	}
	for i := range 256 {
		if !Modern.Has(Cap(i)) {
			t.Fatalf("Modern lacks bit %d", i)
		}
	}
}

var capsSink Caps
var bitsSink [4]uint64
var hasSink bool

func TestCapsNoAllocations(t *testing.T) {
	bits := [4]uint64{1, 2, 3, 4}
	if n := testing.AllocsPerRun(100, func() {
		capsSink = CapsFromBits(bits)
		bitsSink = capsSink.Bits()
		hasSink = capsSink.Has(CapInset)
		capsSink = Browser(Chrome, "120")
		capsSink = Browser(Safari, "TP")
		capsSink = Baseline(2024)
	}); n != 0 {
		t.Fatalf("allocations = %g, want 0", n)
	}
}

func TestBrowser(t *testing.T) {
	// Expected boundaries are from BCD 8.1.7, not generated support tables.
	for _, tt := range []struct {
		id      BrowserID
		version string
		cap     Cap
		want    bool
	}{
		{Chrome, "49", CapInset, false},
		{Chrome, "86.0.4240.198", CapInset, false},
		{Chrome, "87", CapInset, true},
		{Chrome, "87.0.0.1", CapInset, true},
		{Chrome, "110", CapOklch, false},
		{Chrome, "111", CapOklch, true},
		{Chrome, "110", CapDisplayP3, false},
		{Chrome, "111", CapDisplayP3, true},
		{Firefox, "112", CapDisplayP3, false},
		{Firefox, "113", CapDisplayP3, true},
		{Safari, "10", CapDisplayP3, false},
		{Safari, "10.1", CapDisplayP3, true},
		{Safari, "12", CapDisplayP3, true},
		{Safari, "14.1", CapColorFunction, false},
		{Safari, "15", CapColorFunction, true},
		{IOSSafari, "10.2", CapDisplayP3, false},
		{IOSSafari, "10.3", CapDisplayP3, true},
		{ChromeAndroid, "87", CapInset, true},
		{Edge, "86", CapInset, false},
		{Edge, "87", CapInset, true},
		{Firefox, "65", CapInset, false},
		{Firefox, "66", CapInset, true},
		{Samsung, "13.2", CapInset, false},
		{Samsung, "14.0", CapInset, true},
		{Safari, "14.0", CapInset, false},
		{Safari, "14.1", CapInset, true},
		{IOSSafari, "14.1", CapInset, false},
		{IOSSafari, "14.5", CapInset, true},
		{Safari, "17.3", CapScope, false},
		{Safari, "17.4", CapScope, true},
		{Safari, "18.6", CapScope, true},
		{Safari, "26", CapScope, false},
		{Safari, "26.0", CapScope, false},
		{Safari, "26.3", CapScope, false},
		{Safari, "26.4", CapScope, true},
		{IOSSafari, "26.0", CapScope, false},
		{IOSSafari, "26.4", CapScope, true},
		{Safari, "9", CapIs, false}, // :matches() is an alias, not full :is().
		{Safari, "14", CapIs, true},
		{Safari, "99", CapUserSelect, false}, // Preview and prefix do not count.
		{IE, "11", CapFlexbox, false},        // Partial implementation.
		{IE, "11", CapUserSelect, false},     // Prefix only.
		{Firefox, "27", CapFlexbox, false},   // No flex-wrap yet.
		{Firefox, "28", CapFlexbox, true},
		{Opera, "12.1", CapFlexbox, false}, // display:flex exists, but flex-wrap does not.
		{Opera, "15", CapFlexbox, false},   // Full support removed, prefix only.
		{Opera, "16", CapFlexbox, true},
		{Chrome, "999", CapInset, true},
	} {
		t.Run(fmt.Sprintf("%s/%s/%d", tt.id, tt.version, tt.cap), func(t *testing.T) {
			if got := Browser(tt.id, tt.version).Has(tt.cap); got != tt.want {
				t.Fatalf("Browser(%q, %q).Has(%d) = %v, want %v", tt.id, tt.version, tt.cap, got, tt.want)
			}
		})
	}
}

func TestBrowserslistForms(t *testing.T) {
	for _, tt := range []struct {
		id               BrowserID
		version, asPlain string
	}{
		{IOSSafari, "15.2-15.3", "15.2"},
		{Firefox, "115.0esr", "115"},
		{Firefox, "115esr", "115"},
	} {
		if Browser(tt.id, tt.version) != Browser(tt.id, tt.asPlain) {
			t.Fatalf("Browser(%q, %q) differs from Browser(%q, %q)", tt.id, tt.version, tt.id, tt.asPlain)
		}
	}
	if Browser(Safari, "TP") != Browser(Safari, "27") {
		t.Fatal("Safari TP should be the newest Safari version in the data (27)")
	}
	if Browser(Safari, "TP") == (Caps{}) {
		t.Fatal("Safari TP is empty")
	}
}

func TestBrowserUnknown(t *testing.T) {
	for _, id := range []BrowserID{"", "unknown", "Chrome", "safari_ios", "and_uc"} {
		if got := Browser(id, "100"); got != (Caps{}) {
			t.Fatalf("unknown ID %q: %x", id, got.Bits())
		}
	}
	for _, v := range []string{"", "preview", "latest", "≤87", "87-", "-87", "87-90-91", "87-90esr", "esr", "TPX", "87beta", "-1", "+87", "87.", ".87", "87..1", "87.0.0.0.1", "65536", "87.65536", " 87", "87 ", strings.Repeat("9", 100)} {
		if got := Browser(Chrome, v); got != (Caps{}) {
			t.Fatalf("invalid version %q: %x", v, got.Bits())
		}
	}
	for _, v := range []string{"87", "87.0", "87.0.0", "87.0.0.0", "087.00"} {
		if Browser(Chrome, v) != Browser(Chrome, "87") {
			t.Fatalf("numeric equivalence: %q", v)
		}
	}
}

func TestBaseline(t *testing.T) {
	for _, tt := range []struct {
		year int
		cap  Cap
		want bool
	}{
		{2020, CapInset, false}, {2021, CapInset, true},
		{2022, CapOklch, false}, {2023, CapOklch, true},
		{2023, CapLightDark, false}, {2024, CapLightDark, true},
		{2025, CapScope, false}, {2026, CapScope, true},
		{2020, CapImportSupports, false}, {2023, CapImportSupports, false},
		{2024, CapImportSupports, true}, {2026, CapLangList, false},
		{9999, CapWhen, false}, {9999, CapMixin, false},
	} {
		if got := Baseline(tt.year).Has(tt.cap); got != tt.want {
			t.Fatalf("Baseline(%d).Has(%d) = %v, want %v", tt.year, tt.cap, got, tt.want)
		}
	}
	if Baseline(0) != (Caps{}) || Baseline(-1) != (Caps{}) {
		t.Fatal("nonpositive Baseline should be empty")
	}
}

func TestCapWireNumbers(t *testing.T) {
	caps := []Cap{
		CapInset, CapOklch, CapOklab, CapHexAlpha, CapSpaceRGB, CapSpaceHSL,
		CapHwb, CapLab, CapLch, CapColorFunction, CapColorMix, CapRelativeColor,
		CapAlpha, CapLightDark, CapLightDarkImage, CapCurrentColor, CapTransparent,
		CapSystemColors, CapRebeccaPurple, CapNesting, CapNestedDeclarations,
		CapIs, CapWhere, CapNotList, CapHas, CapFocusVisible, CapFocusWithin,
		CapAnyLink, CapDir, CapLangList, CapMarker, CapPlaceholder,
		CapPlaceholderShown, CapCaseInsensitiveAttributes, CapDoubleColon,
		CapNthChildOf, CapUserValid, CapUserInvalid, CapFullscreen, CapAutofill,
		CapCalc, CapMin, CapMax, CapClamp, CapRound, CapMod, CapRemFunction,
		CapTrig, CapExp, CapAbs, CapSign, CapViewportUnits, CapDynamicViewportUnits,
		CapSmallViewportUnits, CapLargeViewportUnits, CapLogicalViewportUnits,
		CapRem, CapCh, CapIc, CapLh, CapRlh, CapCap, CapRcap, CapRex, CapRic,
		CapContainerUnits, CapImageSet, CapGradients, CapDoublePositionStops,
		CapConicGradient, CapGradientInterpolation, CapLogicalKeywords, CapEnv,
		CapCustomProperties, CapSystemUI, CapLogicalBoxShorthands,
		CapLogicalBoxLonghands, CapLogicalRadius, CapLogicalSize, CapLogicalBorder,
		CapFlexGap, CapGridGap, CapPlaceItems, CapPlaceSelf, CapPlaceContent,
		CapAspectRatio, CapTextDecorationColorStyle, CapTextDecorationThickness,
		CapOverflowClip, CapOverflowShorthand, CapTranslate, CapRotate, CapScale,
		CapTransform, CapTransform3D, CapOpacity, CapOpacityPercentage, CapClipPath,
		CapMask, CapBackdropFilter, CapUserSelect, CapAppearance, CapBackgroundClipText,
		CapColorScheme, CapTextWrapBalance, CapTextWrapPretty, CapHyphens,
		CapBoxSizing, CapBorderRadius, CapBoxShadow, CapBackgroundSize, CapLineClamp,
		CapPrintColorAdjust, CapFlowRoot, CapTwoValueDisplay, CapLayer, CapContainerSize,
		CapContainerStyle, CapMediaRange, CapMediaOr, CapPrefersColorScheme,
		CapPrefersReducedMotion, CapDynamicRange, CapColorGamut, CapCustomMedia,
		CapSupports, CapSupportsSelector, CapProperty, CapScope, CapStartingStyle,
		CapFontFace, CapWoff, CapWoff2, CapFontFormatKeyword, CapFontTech,
		CapImportLayer, CapImportSupports, CapNamespace, CapCounterStyle, CapNestedMedia,
		CapFlexbox, CapGrid, CapSubgrid, CapMultiColumn, CapSticky, CapDisplayContents,
		CapIf, CapWhen, CapFunction, CapMixin, CapTypedAttr, CapSiblingIndex,
		CapSiblingCount, CapRandom, CapProgress, CapContrastColor, CapColorContrast,
		CapColorMixVariadic, CapAnchorPositioning, CapFieldSizing, CapTextBoxTrim,
		CapCornerShape, CapInterpolateSize, CapCalcSize, CapScrollDrivenAnimations,
		CapViewTransitions, CapCrossDocumentViewTransitions, CapGridLanes,
		CapReadingFlow, CapTransitionBehavior, CapDisplayP3,
	}
	if len(caps) != len(capBaseline) {
		t.Fatal("append new capabilities to the wire-number test")
	}
	for i, c := range caps {
		if c != Cap(i) {
			t.Errorf("capability at index %d has wire number %d", i, c)
		}
	}
}

func TestBaselineEndOfYearSupport(t *testing.T) {
	// Latest dated releases by December 31 in BCD 8.1.7 browsers.*.releases.
	// Includes scheduled 2026 releases, matching the snapshot's Baseline dates.
	for _, tt := range []struct {
		year                          int
		chrome, edge, firefox, safari string
	}{
		{2015, "47", "13", "43", "9"},
		{2016, "55", "14", "50", "10"},
		{2017, "63", "16", "57", "11"},
		{2018, "71", "18", "64", "12"},
		{2019, "79", "18", "71", "13"},
		{2020, "87", "87", "84", "14"},
		{2021, "96", "96", "95", "15.2"},
		{2022, "108", "108", "108", "16.2"},
		{2023, "120", "120", "121", "17.2"},
		{2024, "131", "131", "133", "18.2"},
		{2025, "143", "143", "146", "26.2"},
		{2026, "158", "158", "161", "27"},
	} {
		for id, version := range map[BrowserID]string{
			Chrome: tt.chrome, Edge: tt.edge, Firefox: tt.firefox, Safari: tt.safari,
		} {
			baseline, browser := Baseline(tt.year), Browser(id, version)
			for x := range capBaseline {
				c := Cap(x)
				if baseline.Has(c) && !browser.Has(c) {
					t.Errorf("Baseline(%d) claims capability %d absent from %s %s at year end", tt.year, c, id, version)
				}
			}
		}
	}
}

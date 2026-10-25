package pristinecss

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"
)

// Captured with the parent revision's ToBinary implementation before lowering.
// Each value is the binary SHA-256 followed by CSS(dst, Modern)'s SHA-256.
var beforeLowering = map[string]string{
	"bootstrap.css":              "d74ed504fc14a234d8a491666a5663b51c93b62cdc5cc2e75da9e26b971c6020 df0b5b373d60fffedd9bd3f27ea59f22a14f85f79ba8753b35d939e38d3079ab",
	"bulma.css":                  "c9fb6c9676ec7563e7578f65e763527d7eaefd1ec74fc1c755abb7e76d903bb1 ccca6a61fea6300fddd8224adf859e04271d986556a9cb066229945d262c908a",
	"daisyui.css":                "2cf44aba54f488ccf29671e06291057c7e3f0cea95ed419032cdc96807192d8c ebd3b2235e3c28e0a705e56cda1aa04c43c85e02eb65559cccb726aa3418ce79",
	"foundation.css":             "f83239a463559a8eb981f14a05e162c8b298ffb8ada6cbb6f8b7b702370cbfb1 f023355c96e501becc6c95553fceb637397a825d9b7e7c47fe044e449c76001f",
	"materialize.css":            "bc6eb99d4fc545c3bfa6063e7150a92924eeb4fe511f79f25372647cd492b7f0 4ca8e567c7d3e1d51a74a82ddb73edfa7ea527f81e03fc6033728165efd88dbf",
	"milligram.css":              "0db75d35f900e75a49da63125316d9c409b7c8f6880d0c1744c78326880978c6 c92accd01caf353d798117ea3b9dbd9b40afb2e8dfb6acb819e9b7e1a877222b",
	"open-props.css":             "8d709f141ee4c6b4be35df8bbe4a4dad7d9df7028c0cb5e9e535ce6d4d182be0 da885ff946fd9a4a93c082a39e71f853ea65b176657c5cbb8c2e7f92b0732193",
	"pico.css":                   "a953d91d07b53b9c3f06ea93182f29df0fe2042ebb8547b61649f0cc62a3293c 11307f546c9bb2d16da8396a19c4320633570e3c7b549f994cd4ee8bdbc67c8b",
	"primeflex.css":              "46a47544e275365f5363c2b0ab4c87316d333123f980f50963e8ff6dd340cb61 5c171abf477e39bcb2ec6f980c61f57f56f4d7820008204dd9d39d7a4ba3dc83",
	"shoelace-dark.css":          "e00e277f34ee3494194813829bce46a278f25dcf45a2143cb099bbde4495f262 e36c84af4566325fe86a0157e530fb181799147aaf5b591e0d1704cab679594a",
	"shoelace-light.css":         "2251ecde2fb087e2f8c6b0f574241c6c92c6ad8ffd7e3a9bedc1cf1df795997b 15f815f4cf3b20d65ba625e15f4cf98c162c009785668c521661604c7a492b37",
	"spectre.css":                "bda3eaeff2d1f32dd8b5e71c95cb3da8328508cb04405357285b4435fe2bda35 aaf71e6b304765ca83051c07b8ff8132a50a018e225af00c8f6de54d994a7843",
	"water.css":                  "3a5937041dd74682b1baff11fee9e1f297c4e62c0c9374846ad526ac6322b5f4 890040c782bc471c9b7c70d378595c8d8d0ecb6e01c1ce2434354c0ccb618364",
	"animate-3.7.2.css":          "9e12f0e0cf114149b85b29ec9fbe426915eb8b291b73da9ab98a7b860d8a5022 c351c47f0f00a3a536ff43877dcf4a2c24114f4043d4e185b9460c91f7422ed4",
	"bootstrap-3.4.1.min.css":    "cd80bea74023ac82c4d5b253266bba2f19c892206aeb8c4991e1d74c8c6da494 80504e05cca74721295131958dab58ebc0f94cdbcfbb10569149243a0cace741",
	"dijit-1.10.4.css":           "57c21573108bd78431c87edaf3b1499d217d71f9aa870e0bb8ee7adbabb6aa54 542f1f98b26530bc58b1936167ef730c107049ea52cff0ca0c1755f93430620f",
	"fontawesome-free-6.7.2.css": "340d8d53ad354db7028cbea6750505d9e6e36d34344b9942c77ac46777634133 780d9dfc166e2e6e69082965f1f9e0deedc2d8c92f8a537c3d37c680ffa43ce1",
	"generated-data-uri.css":     "b1936d499175f3aac4761d39a380356be7531b6cecd5e5120f53111ea0982634 0a193ea944895095e7a34b9d74dc43dbe72a030a2ffffb18fc8d00cd25656375",
	"missing-1.3.0.css":          "35497699b18226f231c280301a192b38c623cd7746a5649b490ac30c4560fab6 8f1b833746da11bacc20b346e4a772a5c82e1598ea3bad1301de220e31347398",
	"pure-0.6.0.min.css":         "f8ba39a2729b19819b988eaa21030be76ee231364db24074c0508fc254e8bd8e 16a8029d9ac8f0b4b584b8715ff8e9b53bf9b7aba8b9fe6c24d68e00fc507aa7",
	"tailwind-4.1.18.css":        "485c555bec19c81801db9fb614901e905f1d0435a0114efe41654f6ebbe4bb94 6a2cf32770aaf7cae894a4cb9f8063dda408ea44100741360d8ee5cc7b23c6f5",
	"yui-console-3.5.1.min.css":  "1bc8b7472bf1a5ae04444d27c26d66ed151ce7fdcfb6c680f67801b1e1e16faf 93b0d15ca440f74ddc15c3d3a86699e6d5deacf6c49801255ac5bee7ef06ec65",
}

func TestLowerZeroUnchanged(t *testing.T) {
	for _, path := range corpusFiles(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			src := readCSS(t, path)
			for _, o := range []Options{{}, {Lower: []Target{}}} {
				bin, report := ToBinary(src, o)
				if report.Variants != 1 || len(report.Gaps) != 0 || len(report.Changes) != 0 || len(report.Mappings) != 0 {
					t.Fatalf("unexpected transformations: %+v", report)
				}
				got := fmt.Sprintf("%x %x", sha256.Sum256(bin), sha256.Sum256(Open(string(bin)).CSS(nil, Modern)))
				if got != beforeLowering[filepath.Base(path)] {
					t.Fatalf("zero-options binary or CSS changed: %s", got)
				}
			}
		})
	}
}

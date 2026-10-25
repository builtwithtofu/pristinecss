package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/builtwithtofu/pristinecss"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "raise" && args[0] != "lower") {
		fmt.Fprintln(stderr, "usage: pristinecss raise FILE --to baseline-YYYY\n       pristinecss lower FILE --target \"BROWSER VERSION\"\nUse - as FILE to read stdin.")
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
			return 0
		}
		return 2
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "usage: pristinecss %s FILE [flags] (FILE may be - for stdin)\n", command)
		flags.PrintDefaults()
	}
	var to, targetValue string
	switch command {
	case "raise":
		flags.StringVar(&to, "to", "", "capability floor: baseline-YYYY (required)")
	case "lower":
		flags.StringVar(&targetValue, "target", "", "browser and version, e.g. \"chrome 49\" (required)")
	}
	args = args[1:]
	// flag stops at the first positional argument; the documented form puts FILE first.
	if len(args) > 0 && (args[0] == "-" || !strings.HasPrefix(args[0], "-")) {
		args = append(args[1:], args[0])
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}

	var options pristinecss.Options
	caps := pristinecss.Modern
	switch command {
	case "raise":
		if to == "" {
			fmt.Fprintln(stderr, "--to is required")
			return 2
		}
		year, ok := strings.CutPrefix(to, "baseline-")
		n, err := strconv.Atoi(year)
		if !ok || err != nil {
			fmt.Fprintln(stderr, "invalid --to: expected baseline-YYYY")
			return 2
		}
		options.Raise = pristinecss.Baseline(n)
		if options.Raise == (pristinecss.Caps{}) {
			fmt.Fprintln(stderr, "unknown browser, or no recorded support for that version")
			return 2
		}
	case "lower":
		parts := strings.Fields(targetValue)
		if len(parts) != 2 {
			fmt.Fprintln(stderr, "invalid --target: expected \"BROWSER VERSION\"")
			return 2
		}
		target := pristinecss.BrowserID(parts[0]).At(parts[1])
		if target.Caps == (pristinecss.Caps{}) {
			fmt.Fprintln(stderr, "unknown browser, or no recorded support for that version; valid IDs: chrome and_chr edge firefox and_ff safari ios_saf ie samsung opera op_mob")
			return 2
		}
		options.Lower = []pristinecss.Target{target}
		caps = target.Caps
	}

	file := flags.Arg(0)
	var src []byte
	var err error
	if file == "-" {
		src, err = io.ReadAll(stdin)
	} else {
		src, err = os.ReadFile(file)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", file, err)
		return 1
	}
	bin, report := pristinecss.ToBinary(src, options)
	starts := lineStarts(src)
	for _, e := range report.Errors {
		fmt.Fprintf(stderr, "%s:%d:%d: parse error: %s\n", file, e.Line(), e.Column(), e.Message)
	}
	for _, c := range report.Changes {
		line, col := position(starts, c.Lo)
		fmt.Fprintf(stderr, "%s:%d:%d: %s\n", file, line, col, c.What)
	}
	for _, g := range report.Gaps {
		line, col := position(starts, g.Lo)
		fmt.Fprintf(stderr, "%s:%d:%d: %s %s: %s\n", file, line, col, g.Target.Browser, g.Target.Version, g.What)
	}
	for _, m := range report.Mappings {
		line, col := position(starts, m.Lo)
		fmt.Fprintf(stderr, "%s:%d:%d: %s %s: colour mapped into the %s gamut\n", file, line, col, m.Target.Browser, m.Target.Version, m.Gamut)
	}
	style := pristinecss.Open(string(bin))
	if _, err := stdout.Write(append(style.CSS(nil, caps), '\n')); err != nil {
		fmt.Fprintf(stderr, "write CSS: %v\n", err)
		return 1
	}
	return 0
}

// lineStarts returns the byte offset at which each line of src begins.
func lineStarts(src []byte) []int {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// position maps a byte offset to a 1-based line and 1-based byte column.
func position(starts []int, offset uint32) (line, col int) {
	next := sort.Search(len(starts), func(i int) bool { return starts[i] > int(offset) })
	return next, int(offset) - starts[next-1] + 1
}

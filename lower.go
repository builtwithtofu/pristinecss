package pristinecss

// Target is a browser to support. It keeps its identity, so build-time checks
// can consult per-browser data (such as which prefixed properties it reads)
// that doesn't belong in Caps.
type Target struct {
	Browser BrowserID // empty when the target is only a capability set
	Version string
	Caps    Caps
}

// At describes this browser version as a build-time target.
// It accepts the same version forms as Browser and retains the original version.
func (b BrowserID) At(version string) Target {
	return Target{Browser: b, Version: version, Caps: Browser(b, version)}
}

package pristinecss

type prefixSupport struct {
	browser  BrowserID
	from, to uint64 // to is exclusive; zero is unbounded
}

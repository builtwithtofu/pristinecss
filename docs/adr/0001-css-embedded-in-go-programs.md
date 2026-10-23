# ADR 0001: CSS embedded in Go programs

## Status

Accepted

## Context

PristineCSS exists so Go apps and servers can carry their CSS with them. Some parse and
change CSS while they run. Others do that work at build time and embed the result with
`go:embed`. Either way the CSS lives inside a Go process, often a long-running server
that holds parsed sheets and serves many requests at once.

## Decision

The library should make CSS cheap and safe to carry inside a Go program.

- Cheap to hold. A sheet a server keeps around shouldn't cost the garbage collector much
  or need locks to read. A sheet nobody is changing is safe for many goroutines at once.
- Cheap to start. Work that can happen at build time can be saved as plain bytes,
  embedded, and loaded without redoing it.
- Safe to trust. Saved bytes are a compatibility contract. Version them and check them
  before use. When a version isn't supported, fail with an error that says how to fix
  it. Never reinterpret old bytes or truncate input silently.
- Don't copy what you can borrow, and say plainly who owns which bytes and for how long.
- The library supplies CSS mechanisms. The program embedding it decides how to use them:
  which CSS to process, what custom syntax means, which transforms are acceptable, and
  how output reaches users. Syntax the library doesn't understand is kept as data, so the
  program can give it meaning.

Before adding something, ask: does it make CSS cheaper or safer to carry in a Go program?
And does it need knowledge only one program has? If it does, it belongs in that program.

## Consequences

- Some things that are easy with a pointer-based object tree are harder here. They're
  weighed case by case, not banned.
- Changing a saved format means handling versions.

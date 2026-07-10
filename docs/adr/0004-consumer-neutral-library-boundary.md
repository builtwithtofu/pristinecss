# ADR 0004: Consumer-neutral library boundary

## Status

Accepted

## Context

PristineCSS is intended to serve different kinds of consumers: applications that parse
CSS directly, generators that embed processed output, analyzers that inspect syntax, and
systems that select precomputed variants. These consumers may have their own platform
branches, component conventions, capability sources, delivery paths, or transformation
policies.

Embedding one consumer's vocabulary into the parser would make the library less useful
elsewhere and would couple CSS mechanics to decisions that the library cannot make with
enough context.

## Decision

PristineCSS is a general-purpose CSS library. It owns reusable CSS mechanisms and data
contracts, not application policy.

The library may provide:

- CSS parsing, recovery, navigation, emission, and bounded mutation;
- generic preservation of unknown syntax, including unknown at-rules;
- reusable syntax and feature analysis;
- table-driven transforms whose semantics are defined in CSS terms;
- versioned AST and rule-selection IR;
- opaque mask algebra and other policy-neutral variant data.

The consuming system owns:

- choosing which CSS input reaches a parse or transformation pass;
- assigning meaning to custom at-rules or other authoring conventions;
- component or document scoping policy;
- mapping capability bits to an environment;
- deciding which transforms and fallbacks are acceptable;
- asset assembly, caching, transport, and delivery; and
- mapping CSS onto a non-CSS presentation model.

Unknown syntax should remain available as data whenever recovery permits. The library
does not need a built-in concept for every convention a consumer can implement by
walking, deleting, unwrapping, replacing, or emitting nodes.

## Consequences

- Public APIs and serialized formats remain useful without adopting a particular
  framework, build system, runtime, or deployment model.
- Demanding consumers can drive performance and API requirements without placing their
  orchestration policy in the library.
- Generic at-rules are an extension seam, not an invitation to hard-code platform
  branches into the parser.
- Feature datasets and transformations may live in PristineCSS when they are expressed
  as reusable CSS knowledge. The decision to apply them remains with the caller.
- Integrations may need their own small orchestration layer. That duplication is
  preferred to coupling the core library to one consumer's lifecycle.

## Related docs

- ADR 0003 defines policy-neutral masks in the IR frame.
- `README.md` describes the public library surface.

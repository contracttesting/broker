# broker

## Setup

```sh
git config core.hooksPath .githooks
```

The `pre-push` hook runs the same gates as CI (`gofmt`, `go vet`, `golangci-lint`, `go test`, `govulncheck`) before every push.

## License

Source-available under the [Business Source License 1.1](LICENSE) — the same
license HashiCorp and MariaDB use.

In plain terms: read, copy, modify and self-host it for any purpose, commercial
work included. The one thing you may not do is offer it — or something
substantially like it — to others as a competing commercial product or service.
Every version turns into plain Apache License 2.0 four years after its release.

Need a competing use? A commercial license is available:
alefaraujocastelo@gmail.com

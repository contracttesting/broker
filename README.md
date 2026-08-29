# broker

## Setup

```sh
git config core.hooksPath .githooks
```

The `pre-push` hook runs the same gates as CI (`gofmt`, `go vet`, `golangci-lint`, `go test`, `govulncheck`) before every push.

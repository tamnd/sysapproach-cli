# sysapproach

Browse [Computer Networks: A Systems Approach](https://book.systemsapproach.org/) from the command line.

`sysapproach` is a single pure-Go binary. No API key required.

## Install

```bash
go install github.com/tamnd/sysapproach-cli/cmd/sysapproach@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/sysapproach-cli/releases), or run the container image:

```bash
docker run --rm ghcr.io/tamnd/sysapproach:latest --help
```

## Usage

```bash
# List all chapters
sysapproach chapters

# Table output
sysapproach chapters -o table

# JSON output, ready for jq
sysapproach chapters -o json

# Get chapter URLs only
sysapproach chapters -o url
```

## Commands

| Command | Description |
|---------|-------------|
| `chapters` | List all chapters of Computer Networks: A Systems Approach |
| `version` | Show version information |

## Global flags

```
-o, --output string    output format: table|json|jsonl|csv|tsv|url|raw (default "auto")
-n, --limit int        limit number of records (0 = all)
    --fields strings   comma-separated columns to include
    --no-header        omit header row
    --template string  Go text/template per record
    --timeout duration per-request timeout (default 30s)
    --delay duration   minimum spacing between requests
    --retries int      retry attempts on 429/5xx (default 3)
```

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents:

```bash
sysapproach serve --addr :7777    # GET /v1/chapters returns NDJSON
sysapproach mcp                   # speak MCP over stdio
```

## Development

```
cmd/sysapproach/   thin main
cli/               assembles the kit App from the sysapproach domain
sysapproach/       HTTP client, data models, domain driver
docs/              documentation site
```

```bash
make build      # ./bin/sysapproach
make test       # go test ./...
make vet        # go vet ./...
```

## License

Apache-2.0. See [LICENSE](LICENSE).

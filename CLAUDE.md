# CLAUDE.md

This file provides guidance to AI coding agents working with this repository.

## Project Overview

GeoTap is a Google Maps geographic data scanner written in Go. It extracts business listings by country, region, or coordinates without requiring an API key or login. Features TLS fingerprinting and anti-blocking measures.

## Common Commands

```bash
make build                    # Build binary with version
make install                  # Install to $GOPATH/bin
go build ./cmd/geotap         # Quick build
go vet ./...                  # Static analysis
go test ./...                 # Run tests
goreleaser --snapshot --clean # Test cross-platform build
```

## Architecture

```
cmd/geotap/           CLI entry point
  main.go             Dispatcher: TUI (default) or subcommand (scan/export/version)
  scan.go             Headless scan: flags → sectors → scraper → DB
  export.go           DB → CSV export

internal/
  model/              Data structures (Business, SearchParams, Sector)
  engine/
    geo/              Grid generation, boundary polygons, geocoding
      geodata/        Embedded ne_110m_countries.geojson (~838KB)
    scraper/          HTTP client (utls), Google Maps parser, worker pool
    storage/          SQLite operations (INSERT OR IGNORE dedup)
  tui/                Bubbletea terminal UI
    views/            home, search (country autocomplete), progress, explorer, recent, filepicker
    components/       Reusable UI components (mapview)
    styles/           Theme colors and lipgloss styles
```

## Key Patterns

- **Bubbletea value receivers**: Mutable shared state must be behind `*pointer` to survive value copies
- **Stats tracking**: `sync/atomic` counters passed via `RunOptions.Stats` pointer
- **TLS fingerprinting**: `utls` with `HelloChrome_Auto` + ALPN `http/1.1` (mandatory for Google)
- **Geo filtering**: `planar.MultiPolygonContains` from `paulmach/orb` filters businesses by country polygon
- **Text normalization**: `golang.org/x/text` NFD decomposition for accent-insensitive search
- **Deduplication**: SQLite `UNIQUE(cid, query)` + `INSERT OR IGNORE`

## Anti-Blocking Rules

1. Never add `tch`, `ech`, or `psi` params to `tbm=map` URLs
2. Always use `utls` TLS fingerprinting (not standard `net/http`)
3. Rotate Chrome user agents
4. Set `CONSENT` cookie to skip consent page
5. Use exponential backoff on 429/403/302 responses

## Testing Protocol

Test incrementally: 1 sector first, then radius 1km, then radius 5km, then full country.

## Code Style

- Go standard formatting (`gofmt`)
- No unnecessary abstractions or over-engineering
- Prefer editing existing files over creating new ones
- Keep error handling at system boundaries

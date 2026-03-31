# CODEX.md

This file gives Codex a quick working map of the `casd` repository.

## Overview

`casd` is a small Go project for building Community Arts and Sciences Day schedules from CSV inputs.

It includes:

- `cmd/sorter`: command-line scheduler
- `cmd/web`: web UI for uploading CSVs and viewing schedules
- `cmd/randomizer`: utility for anonymizing teacher and student names in a groups CSV
- `pkg/sorter`: core scheduling models and assignment logic

The project is lightweight. Most important behavior lives in `pkg/sorter` and is duplicated slightly between the CLI and web entrypoints.

## Repository Layout

- `README.md`: basic usage
- `Makefile`: build, run, fmt, lint, and Docker targets
- `cmd/sorter/main.go`: CLI scheduling flow
- `cmd/web/main.go`: HTTP upload flow and HTML rendering
- `cmd/randomizer/main.go`: CSV name randomizer
- `pkg/sorter/group.go`: group parsing, preferences, output helpers
- `pkg/sorter/workshop.go`: workshop parsing, capacity tracking, HTML/text output
- `pkg/sorter/sorter.go`: shared booking helpers
- `groups.csv`, `artworkshops.csv`, `scienceworkshops.csv`: sample inputs

## What The Scheduler Does

The scheduler reads:

- one groups CSV
- one art workshops CSV
- one science workshops CSV

It then tries to assign each group to four workshop slots total:

- 2 art sessions
- 2 science sessions

High-level assignment flow:

1. Read groups and workshop definitions from CSV.
2. Pre-book any required parent/presenter workshops.
3. Optionally shuffle group order.
4. Book preferred art workshops.
5. Book preferred science workshops.
6. Fill remaining needed sessions from available workshops.
7. Rebalance underutilized sessions when possible.

Important constraints enforced by booking logic:

- group grade must fall within workshop grade range
- a group cannot be booked into the same workshop twice
- a group can only occupy one workshop per time slot
- a workshop session must have enough remaining capacity for the entire group

## Scheduling Details

There are four bookable workshop sessions with a recess displayed between the middle two.

Session display times are defined in `pkg/sorter/workshop.go`:

- `9:40 - 10:10 am`
- `10:15 - 10:45 am`
- `10:50 - 11:05 am (Recess)`
- `11:10 - 11:40 am`
- `11:45 am - 12:15 pm`

Internally, only 4 workshop slots are booked. The recess row is display-only.

Booking behavior to know:

- `BookWorkshopIfAvailable` chooses from sessions with the most remaining space, then picks randomly among ties.
- Preference order matters for direct booking from CSV choices.
- Rebalancing can move a group from one workshop to another if it helps an underutilized session and does not drop the old session below the minimum utilization threshold.

## CSV Expectations

The code expects fixed column positions.

Groups CSV fields used by `pkg/sorter/group.go`:

- column 0: teacher
- column 2: grade
- column 3: group name
- column 4: comma-separated student names
- columns 5-8: preferred art workshop IDs
- columns 9-12: preferred science workshop IDs
- column 13: space-separated parent workshop IDs

Workshop CSV fields used by `pkg/sorter/workshop.go`:

- column 0: workshop identifier and name in the form `ID - Name`
- column 1: grade range like `K-2` or `4-5`
- columns 2-5: whether each session is offered (`y` means offered)
- column 6: capacity
- column 7: room

If editing CSV parsing, verify sample CSVs still match these assumptions.

## Common Commands

Build both main binaries:

```bash
make build
```

Run the web app on `:8080`:

```bash
make run
```

Build the randomizer:

```bash
make build-randomizer
```

Format the code:

```bash
make fmt
```

Run lint:

```bash
make lint
```

Run tests:

```bash
make test
```

Open coverage in the browser:

```bash
make test-coverage
```

Direct CLI example:

```bash
./sorter --groups groups.csv --art-workshops artworkshops.csv --science-workshops scienceworkshops.csv
```

Useful CLI flags in `cmd/sorter/main.go`:

- `--random`
- `--min-utilization`
- `--print-output`

## Release Version Bumps

The release workflow defaults to a patch bump on pushes to `main`, but it can also bump the minor or major version.

For push-based releases, include one of these markers in the commit message:

- `[patch]` or `#patch`
- `[minor]` or `#minor`
- `[major]` or `#major`

Examples:

```bash
git commit -m "[minor] Refresh scheduler UI and workflow updates"
```

```bash
git commit -m "#major Rework release process"
```

If no marker is present, the workflow falls back to a patch bump.

Manual `workflow_dispatch` runs can also choose the bump type directly in GitHub Actions.

## Development Notes

- Go version is pinned in `go.mod`.
- The Makefile can bootstrap its own Go toolchain and `golangci-lint`.
- Scheduler tests live in `pkg/sorter/schedule_test.go`.
- The randomizer is not a generic slice shuffler. It specifically rewrites teacher and student names in a CSV while keeping replacements consistent.

## Good First Files To Read

If you are starting fresh, read these in order:

1. `README.md`
2. `cmd/sorter/main.go`
3. `pkg/sorter/group.go`
4. `pkg/sorter/workshop.go`
5. `cmd/web/main.go`

## Watchouts

- `idToKind` assumes art workshop IDs start with `A`; everything else is treated as science.
- `readAndParseCSV` skips the first row as a header.
- Group schedules are stored as a fixed-length slice of 4 workshop pointers.
- Because some booking decisions use randomness, output can differ between runs when randomization or tie-breaking is involved.

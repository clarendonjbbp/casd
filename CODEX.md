# CODEX.md

This file gives Codex a quick working map of the `casd` repository.

## Overview

`casd` is a small Go project for building Community Arts and Sciences Day schedules from CSV inputs.

It includes:

- `cmd/sorter`: command-line scheduler
- `cmd/web`: web UI for uploading CSVs and viewing schedules
- `cmd/randomizer`: utility for anonymizing teacher and student names in a groups CSV
- `pkg/sorter`: core scheduling models and assignment logic

The project is lightweight. Most important behavior now lives in shared scheduler code in `pkg/sorter`, with the CLI and web entrypoints acting mostly as thin wrappers.

## Repository Layout

- `README.md`: basic usage
- `Makefile`: build, run, fmt, lint, and Docker targets
- `cmd/sorter/main.go`: CLI scheduling flow
- `cmd/web/main.go`: HTTP upload flow and embedded template rendering
- `cmd/web/templates/home.html`: upload page template
- `cmd/web/templates/results.html`: results page template
- `cmd/randomizer/main.go`: CSV name randomizer
- `pkg/sorter/group.go`: group parsing, preferences, output helpers
- `pkg/sorter/workshop.go`: workshop parsing, capacity tracking, HTML/text output
- `pkg/sorter/schedule.go`: shared CSV loading and scheduling flow
- `pkg/sorter/sorter.go`: shared booking helpers
- `pkg/sorter/schedule_test.go`: scheduler and scoring tests
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
2. Normalize duplicate student preferences by keeping the first occurrence of each workshop ID and dropping lower-ranked duplicates.
3. Pre-book any required parent/presenter workshops.
4. Optionally shuffle group order.
5. Book preferred art workshops.
6. Book preferred science workshops.
7. Fill remaining needed sessions from available workshops.
8. Rebalance underutilized sessions when possible.

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
- `cmd/sorter` and `cmd/web` both call the shared `sorter.Schedule(...)` API in `pkg/sorter/schedule.go`.

## Satisfaction Scoring

Group satisfaction is computed internally as a weighted ranking score and then displayed as a percentage.

Current internal scoring in `pkg/sorter/group.go`:

- parent/presenter workshop: 5 points
- 1st choice: 4 points
- 2nd choice: 3 points
- 3rd choice: 2 points
- 4th choice: 1 point
- fallback / unrequested workshop: 0 points

Important implementation details:

- The points are intentionally not shown in the UI anymore. The web and text output show only `Satisfaction: N%`.
- Per-session display still shows labels like `1st Choice`, `2nd Choice`, `Parent Workshop`, and `Fallback`.
- Duplicate preferences are normalized when reading the CSV, so repeated IDs do not inflate the score.
- `MaxSatisfaction()` is based on the best possible score for the slots the group can actually attend, not on all four raw preference ranks.

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

Preference cleanup behavior:

- duplicate workshop IDs in a group's art/science preferences are deduplicated in order
- the first occurrence is kept
- lower-ranked duplicates are ignored
- the preference slice is padded back to 4 entries with blanks so existing booking code can keep its assumptions

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

Run lint and auto-fix what can be fixed:

```bash
make lint-fix
```

Run tests:

```bash
make test
```

Open coverage in the browser:

```bash
make test-coverage
```

Build the Docker image locally without pushing:

```bash
make docker-build
```

Build and push the Docker image:

```bash
make docker-push
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

## CI / PR Workflow

- Always create a branch before making changes so the PR workflow runs before merge.
- Branch names should usually use the `codex/` prefix.
- Open a PR instead of pushing work directly to `main`.
- The PR workflow runs `make lint`, `make test`, `make build`, and `make docker-build`.
- The release workflow runs the real test suite and uses `make docker-push`.

## Development Notes

- Go version is pinned in `go.mod`.
- The Makefile uses repo-local caches for Go and linting.
- `golangci-lint` uses the v2 module path.
- Scheduler tests live in `pkg/sorter/schedule_test.go` and use Testify.
- The randomizer is not a generic slice shuffler. It specifically rewrites teacher and student names in a CSV while keeping replacements consistent.
- The web UI uses embedded templates instead of inline HTML strings.
- The current homepage/result pages use a Clarendon-school-poster-inspired visual style.

## Good First Files To Read

If you are starting fresh, read these in order:

1. `README.md`
2. `pkg/sorter/schedule.go`
3. `pkg/sorter/group.go`
4. `pkg/sorter/workshop.go`
5. `cmd/web/main.go`
6. `cmd/web/templates/home.html`

## Watchouts

- `idToKind` assumes art workshop IDs start with `A`; everything else is treated as science.
- `readAndParseCSV` skips the first row as a header.
- Group schedules are stored as a fixed-length slice of 4 workshop pointers.
- Because some booking decisions use randomness, output can differ between runs when randomization or tie-breaking is involved.
- If a group has a parent workshop, that workshop consumes one of the group's art or science slots for scheduling and satisfaction purposes.

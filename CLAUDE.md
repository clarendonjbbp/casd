# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Overview
The *casd* repository implements a **Community Arts and Sciences Day scheduler** written in Go.  It contains:

- **Command‑line tool** (`cmd/sorter`) that reads CSV files and outputs a schedule.
- **Web UI** (`cmd/web`) that lets a user upload CSVs, configure options, and view the resulting schedule in the browser.
- **Randomizer** (`cmd/randomizer`) – a small helper program that produces a random permutation of a slice (currently unused but present in the repository).
- **Core scheduling logic** in `pkg/sorter` – structures for groups and workshops, CSV parsing helpers, and the algorithm that assigns workshops while respecting grades, preferences, and capacity.
- A **Makefile** that installs the Go toolchain, golangci‑lint, builds binaries, runs linting, and builds a multi‑arch Docker image.

The repository is intentionally lightweight; most of the business logic lives in `pkg/sorter`.

---

## Common Commands
| Command | Purpose | Notes |
|---|---|---|
| `make build` | Builds both the CLI (`sorter`) and the web server (`sorter-web`). | Runs `go build` for each binary. |
| `make run` | Builds and starts the web server. | Equivalent to `go build -o sorter-web ./cmd/web && ./sorter-web`. |
| `make fmt` | Runs `go fmt ./...`. | Keeps code formatted. |
| `make lint` | Runs `golangci-lint run ./...`. | Requires `golangci-lint` to be installed via the Makefile. |
| `make docker-build` | Builds a multi‑architecture Docker image (`${IMAGE_NAME}:${TAG}`). | Uses Docker Buildx; the image is pushed only when `docker-push` is executed. |
| `go test ./... -run TestName` | Run a specific unit test if any exist. | Currently the repo has no tests, so this will be a no‑op. |

> **Tip:** The Makefile automatically downloads the Go toolchain and golangci‑lint if they are missing, so you can just run `make` without any manual setup.

---

## CSV Input Format
The web UI and CLI both expect three CSV files:

1. `groups.csv` – contains teacher, grade, student list, preferred art and science workshop IDs, and parent‑presenter workshop IDs.
2. `artworkshops.csv` – contains art workshop details (ID, name, room, capacity, session times).
3. `scienceworkshops.csv` – contains science workshop details.

The CSV readers are in `pkg/sorter/reader.go` (not shown here).  The format is documented in the README and the sample CSV files.

---

## Running the Web Server
```bash
$ make run
```
The server listens on port `8080`.  Navigate to `http://localhost:8080` to upload the CSV files and schedule workshops.

The UI uploads the files to `/upload`, runs the same scheduling algorithm as the CLI, and renders the schedule in HTML.

---

## Running the CLI Scheduler
```bash
$ make build
$ ./sorter --groups groups.csv --art-workshops artworkshops.csv --science-workshops scienceworkshops.csv
```
The CLI accepts the same flags as the web upload handler.  It prints the schedule to stdout.

---

## Core Packages
| Package | Responsibility |
|---|---|
| `pkg/sorter` | Defines `Group` and `Workshop` structs, CSV parsing helpers, and the scheduling algorithm (`BookWorkshopIfAvailable`, `RebalanceWorkshop`, etc.). |
| `cmd/sorter` | CLI entry point that parses flags, reads CSVs, runs the scheduler, and prints the result. |
| `cmd/web` | HTTP server that serves a simple UI and runs the scheduler on uploaded files. |
| `cmd/randomizer` | Utility to shuffle slices; currently not used by the main flow. |

---

## Development Notes
- The Go version is specified in `go.mod` (currently 1.23.8).  The Makefile downloads this version automatically.
- `golangci-lint` is used for linting; run `make lint` to check for style issues.
- Docker images are built for `linux/amd64` and `linux/arm64` by default.
- No tests are bundled with the repo, but the code is designed to be testable.  To add tests, place files ending in `_test.go` under the appropriate package and run `go test ./...`.

---

## File Structure Highlights
- `Makefile` – central build script.
- `cmd/` – executable commands.
- `pkg/` – library code.
- `README.md` – basic usage instructions (mirrored in this file).
- `artworkshops.csv`, `scienceworkshops.csv`, `groups.csv` – sample data.
- `docker-compose.yml` – optional Compose config for local Docker testing.

---

## Key Files for Quick Access
- `pkg/sorter/group.go` – main data model.
- `pkg/sorter/sorter.go` – scheduling logic.
- `cmd/sorter/main.go` – CLI.
- `cmd/web/main.go` – web server.
- `Makefile` – build and tooling commands.

---

## Summary
Use `make` to set up the environment and build binaries.  Run `make run` to start the web server or `./sorter` to execute the scheduler from the command line.  The core logic lives in `pkg/sorter`; the web UI simply forwards uploads to the same scheduling code.

---

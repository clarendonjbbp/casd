# Community Arts and Sciences Day Assignments  

## How to run web server

```bash
$ make run
go build -o sorter-web ./cmd/web
2025/03/23 22:00:40 Starting server on :8080
```

Then go to [localhost:8080](http://localhost:8080) in your browser. Sample groups and workshops are in the root of this repo.

## How build command line sorter

```bash
$ make build
go build -o sorter ./cmd/sorter
go build -o sorter-web ./cmd/web
```

## Tests

```bash
$ make test
```

To generate and open HTML coverage locally:

```bash
$ make test-coverage
```

## Release Version Bumps

The release workflow defaults to a patch bump on pushes to `main`.

To force a larger bump on a normal push, include one of these markers in the commit message:

- `[patch]` or `#patch`
- `[minor]` or `#minor`
- `[major]` or `#major`

Example:

```bash
$ git commit -m "[minor] Refresh scheduler UI and workflow updates"
```

If no marker is present, the workflow uses a patch bump.

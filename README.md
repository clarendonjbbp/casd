# Community Arts and Sciences Day Assignments  

## How to run web server

```bash
$ make run
go build -o sorter-web ./cmd/web
./sorter-web
2025/03/23 22:00:40 Starting server on :8080
```

Then go to [localhost:8080](http://localhost:8080) in your browser. Sample groups and workshops are in the root of this repo.

## How build command line sorter

```bash
$ make build
go build -o sorter ./cmd/sorter
go build -o sorter-web ./cmd/web
```

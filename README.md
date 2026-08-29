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
go build -o workshop-popularity ./cmd/popularity
```

## How to run workshop popularity report

```bash
$ make build-popularity
$ ./workshop-popularity --groups "testdata/2026/groups_2026_v1 - groups.csv" --art-workshops "testdata/2026/artworkshops_2026_v1 - artworkshops.csv" --science-workshops "testdata/2026/scienceworkshops_2026_v1 - scienceworkshops.csv"
```

For easier copy/paste into email or docs, generate HTML, open it in a browser, and copy the rendered tables:

```bash
$ ./workshop-popularity --format html --groups "testdata/2026/groups_2026_v1 - groups.csv" --art-workshops "testdata/2026/artworkshops_2026_v1 - artworkshops.csv" --science-workshops "testdata/2026/scienceworkshops_2026_v1 - scienceworkshops.csv" > popularity.html
$ open popularity.html
```

Markdown table output is also available with `--format markdown`.

The popularity report scores each class/group's preferences this way:

- 1st choice: 4 points
- 2nd choice: 3 points
- 3rd choice: 2 points
- 4th choice: 1 point

The report sorts by normalized score, which compares each workshop's preference points against the maximum possible points from groups eligible for that workshop's grade range.

## How the algorithm works

The scheduler builds each group's day in stages.

Each group is scheduled into:

- 2 art workshop sessions
- 2 science workshop sessions

The current scheduling flow is:

1. Read groups, art workshops, and science workshops from CSV.
2. Normalize duplicate preferences in each group's art and science lists by keeping the first occurrence and dropping lower-ranked duplicates.
3. Book parent or presenter-linked workshops first.
4. Run a guarantee pass that tries to give every group:
   - 1 preferred art workshop
   - 1 preferred science workshop
5. Fill the remaining art and science slots using the normal greedy preference order.
6. If a group still has open slots, fill them from other available workshops.
7. Rebalance underutilized workshop sessions when possible.

Important booking rules:

- Groups must be within the workshop's grade range.
- A group cannot be assigned to the same workshop twice.
- A group can only attend one workshop per time slot.
- A session must have enough remaining capacity for the entire group.
- Parent workshops are attempted first, but they still must satisfy grade, time-slot, and capacity constraints.

The greedy parts of the algorithm work like this:

- Within a category, workshops are tried in the order the group requested them.
- When a workshop has multiple possible sessions, the scheduler prefers sessions with the most remaining capacity.
- If a preferred workshop cannot be booked, the scheduler moves to the next preference.
- If no preferred workshop can be booked for a needed slot, the scheduler falls back to other available workshops of the same kind, then other available workshops overall if necessary.

Rebalancing behavior:

- After the initial schedule is built, the sorter tries to improve workshop utilization by moving groups into underfilled sessions.
- Rebalancing now avoids moves that would take away a group's only preferred art or only preferred science workshop and replace it with a fallback.

The output also reports summary metrics for the final schedule, including:

- overall satisfaction points
- average satisfaction percent
- groups with at least 1 preferred art workshop
- groups with at least 1 preferred science workshop

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
